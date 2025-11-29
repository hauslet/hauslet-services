package service

import (
	"context"
	"crypto/sha256"
	"strings"
	"time"

	"hauslet/internal/auth/domain"
	"hauslet/internal/auth/repository/schema"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// maskEmail masks email addresses for non-debug logs to reduce PII exposure
// Example: "user@example.com" -> "u***@example.com"
func maskEmail(email string) string {
	if email == "" {
		return "***"
	}
	if idx := strings.Index(email, "@"); idx > 0 {
		return email[:1] + "***@" + email[idx+1:]
	}
	return "***"
}

func (s *AuthServiceImpl) OAuthService() *auth.Service {
	options := auth.Opts{
		Logger: s.log,
		SecretReader: token.SecretFunc(func(id string) (string, error) {
			return s.cfg.JWTSecret, nil
		}),

		// ClaimsUpd is called after authentication (OAuth or Direct)
		// This is where we create/link user and create session
		ClaimsUpd: token.ClaimsUpdFunc(func(claims token.Claims) token.Claims {
			return s.enrichClaims(claims)
		}),

		TokenDuration:  s.cfg.TokenDuration,
		CookieDuration: s.cfg.CookieDuration,
		Issuer:         "Hauslet",
		URL:            s.cfg.RedirectURL,
		AvatarStore:    avatar.NewLocalFS(s.cfg.AvatarStorePath),
		SendJWTHeader:  false, // send JWT in header to simplify XSRF handling for clients
		DisableXSRF:    s.cfg.DisableXSRF,

		// Validate users before allowing access
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			ctx := context.Background()
			if claims.User == nil || claims.User.ID == "" {
				return false
			}

			// Use canonical user ID (real UUID)
			userID := claims.User.ID
			if userID == "" {
				s.log.Logf("WARN Token rejected - missing user ID")
				return false
			}

			// STEP 1: Validate session exists in Redis
			sessionID := claims.User.StrAttr("sid")
			if sessionID == "" {
				// No session ID in token - reject (all tokens must have sid)
				s.log.Logf("WARN Token rejected - missing session ID for user %s", userID)
				return false
			}

			session, err := s.repository.GetSessionByID(ctx, sessionID)
			if err != nil {
				// FAIL-CLOSED: Redis is mandatory for session validation
				// If Redis is unavailable, reject authentication
				s.log.Logf("ERROR Session validation failed (Redis unavailable): %v - rejecting token", err)
				return false
			}
			if session == nil {
				// Session was explicitly deleted/revoked - reject immediately
				s.log.Logf("INFO Token rejected - session %s revoked for user %s", sessionID, userID)
				return false
			}
			// Session exists and valid - continue to user validation

			// STEP 2: Check if user is active (existing logic)
			user, err := s.repository.GetUserByID(ctx, userID)
			if err != nil || user == nil {
				return false
			}

			return user.IsActive
		}),
	}

	service := auth.NewService(options)

	// Add Google OAuth provider with email scope
	// Using AddCustomProvider to specify scopes (profile + email)
	service.AddCustomProvider(
		"google",
		auth.Client{
			Cid:     s.cfg.GoogleClientID,
			Csecret: s.cfg.GoogleCLSecret,
		},
		provider.CustomHandlerOpt{
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
			InfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:  []string{"openid", "email", "profile"},
			MapUserFn: func(data provider.UserData, _ []byte) token.User {
				userInfo := token.User{
					ID:   "google_" + token.HashID(sha256.New(), data.Value("sub")),
					Name: data.Value("name"),
				}
				// Extract email from the correct field
				if email := data.Value("email"); email != "" {
					userInfo.Email = email
				}
				return userInfo
			},
		},
	)

	// Add Direct (password) authentication provider
	service.AddDirectProvider("password", provider.CredCheckerFunc(func(user, password string) (ok bool, err error) {

		ctx := context.Background()
		_, authErr := s.AuthenticatePassword(ctx, user, password)
		if authErr != nil {
			return false, nil // Invalid credentials
		}
		return true, nil
	}))

	// Log XSRF protection status
	if s.cfg.DisableXSRF {
		s.log.Logf("WARN XSRF protection is DISABLED - only use in development")
	} else {
		s.log.Logf("INFO XSRF protection is ENABLED")
	}

	return service
}

// enrichClaims is called after authentication (OAuth or Direct)
// It handles user creation/linking, session creation, and role assignment
func (s *AuthServiceImpl) enrichClaims(claims token.Claims) token.Claims {
	ctx := context.Background()

	// Extract auth info from claims
	if claims.User == nil {
		s.log.Logf("WARN Auth: No user in claims")
		return claims
	}

	// Check if this is a linking request (state starts with "link.")
	isLinking := false
	var linkState *LinkState
	if claims.User != nil && strings.HasPrefix(claims.User.StrAttr("state"), "link.") {
		stateToken := claims.User.StrAttr("state")
		if s.linkStateManager != nil {
			var err error
			linkState, err = s.linkStateManager.ValidateState(stateToken)
			if err != nil {
				s.log.Logf("ERROR OAuth Linking: Invalid state token: %v", err)
				return claims // Return empty claims on error
			}
			isLinking = true
		}
	}

	// Detect provider type with fallback chain for reliability
	providerUserID := claims.User.ID
	email := claims.User.Email
	name := claims.User.Name

	// direct provider sets Name to submitted login; use it if email is missing
	if email == "" && name != "" {
		email = name
	}

	// Priority 1: Explicit provider attribute (most reliable)
	provider := claims.User.StrAttr("provider")

	// Priority 2: Parse from ID format for backward compatibility (e.g., "google_123456")
	if provider == "" {
		if idx := strings.Index(claims.User.ID, "_"); idx > 0 {
			provider = claims.User.ID[:idx]
		}
	}

	// Priority 3: Default to password (fail-safe for Direct provider)
	if provider == "" || provider == "unknown" {
		provider = "password"
	}

	var user *schema.User
	var err error

	if provider == "password" {
		// Password login - Name is the email
		email = claims.User.Name
		// PASSWORD AUTHENTICATION
		// User already exists and was validated by AuthenticatePassword
		user, err = s.repository.GetUserByEmail(ctx, email)
		if err != nil || user == nil {
			s.log.Logf("ERROR Auth: Error fetching user for password login: %v", err)
			return claims
		}

		s.log.Logf("INFO Auth: User %s (ID: %s) logged in via password", maskEmail(user.PrimaryEmail), user.ID)

	} else {
		// OAUTH AUTHENTICATION
		// Handle linking mode if active
		if isLinking && linkState != nil {
			// Validate provider matches
			if linkState.Provider != provider {
				s.log.Logf("ERROR OAuth Linking: Provider mismatch: expected %s, got %s", linkState.Provider, provider)
				return claims
			}

			// Link identity to authenticated user
			if err := s.linkIdentityToUser(ctx, linkState, provider, providerUserID, email); err != nil {
				s.log.Logf("ERROR OAuth Linking: Failed to link identity: %v", err)
				return claims
			}

			// Get user and continue with session creation
			user, err = s.repository.GetUserByID(ctx, linkState.UserID)
			if err != nil || user == nil {
				s.log.Logf("ERROR OAuth Linking: Error fetching user after linking: %v", err)
				return claims
			}

			s.log.Logf("INFO OAuth Linking: Successfully linked %s to user %s (ID: %s)", provider, maskEmail(user.PrimaryEmail), user.ID)

		} else {
			// Normal OAuth login (not linking)
			// Step 1: Check if UserIdentity already exists for this provider
			identity, err := s.repository.GetUserIdentityByProvider(ctx, provider, providerUserID)
			if err != nil {
				s.log.Logf("ERROR OAuth: Error checking identity: %v", err)
				return claims
			}

			if identity != nil {
				// EXISTING OAUTH LOGIN
				user, err = s.repository.GetUserByID(ctx, identity.UserID.String())
				if err != nil || user == nil {
					s.log.Logf("ERROR OAuth: Error fetching user for existing identity: %v", err)
					return claims
				}

				// FIX: Update corrupted Google OAuth email (from before email scope was added)
				// Check if this is a Google identity with corrupted email (no @ symbol)
				if provider == "google" && !strings.Contains(identity.Email, "@") && email != "" {
					s.log.Logf("INFO Fixing corrupted Google OAuth email for user %s: %q -> %q",
						user.ID, identity.Email, email)
					identity.Email = email
				}

				// Update last used timestamp for this identity
				now := time.Now()
				identity.LastUsedAt = &now
				_ = s.repository.UpdateUserIdentity(ctx, identity)

				s.log.Logf("INFO OAuth: Existing user %s (ID: %s) logged in via %s", maskEmail(user.PrimaryEmail), user.ID, provider)

			} else {
				// NEW OAUTH LOGIN
				// Step 2: Check if user exists by email
				user, err = s.repository.GetUserByEmail(ctx, email)
				if err != nil {
					s.log.Logf("ERROR OAuth: Error checking user by email: %v", err)
					return claims
				}

				if user == nil {
					// BRAND NEW USER - create user + identity together
					user = &schema.User{
						ID:           uuid.New(),
						Name:         name,
						PrimaryEmail: email,
						Role:         schema.RoleUser,
						IsActive:     true,
					}

					identity = &schema.UserIdentity{
						ID:            uuid.New(),
						UserID:        user.ID,
						Provider:      provider,
						ProviderID:    providerUserID,
						Email:         email,
						EmailVerified: true, // OAuth providers verify emails
					}

					// Create user + identity atomically
					if err := s.repository.CreateUserWithIdentity(ctx, user, identity); err != nil {
						s.log.Logf("ERROR OAuth: Error creating user+identity: %v", err)
						return claims
					}

					s.log.Logf("INFO OAuth: Created new user %s (ID: %s) via %s", maskEmail(user.PrimaryEmail), user.ID, provider)

					// Send welcome email (no OTP for OAuth users - already verified)
					// Non-blocking, fire-and-forget
					go func(email, name string) {
						if err := s.SendWelcomeEmail(context.Background(), email, name, ""); err != nil {
							s.log.Logf("WARN Failed to send OAuth welcome email to %s: %v", email, err)
						} else {
							s.log.Logf("INFO OAuth welcome email sent to %s", email)
						}
					}(user.PrimaryEmail, user.Name)

				} else {
					// USER EXISTS - REJECT (no auto-linking for security)
					// Only allow linking via explicit "Link Account" flow
					s.log.Logf("WARN OAuth: User exists with email %s but provider not linked. Auto-linking disabled.", maskEmail(email))
					return claims // Empty claims = auth failure
				}
			}
		}
	}

	// Update last login timestamp
	_ = s.repository.UpdateUserLastLogin(ctx, user.ID.String())

	// Retrieve request metadata (IP, User-Agent) if available
	var ip, userAgent string
	if metadata := s.requestMetadata.Get(email); metadata != nil {
		ip = metadata.IP
		userAgent = metadata.UserAgent
	}

	// Create session in Redis
	sessionID := uuid.New().String()
	session := &domain.Session{
		ID:        sessionID,
		UserID:    user.ID,
		Provider:  provider,
		ExpiresAt: time.Now().Add(s.cfg.SessionDuration),
		CreatedAt: time.Now(),
		IP:        ip,
		UserAgent: userAgent,
	}

	if err := s.repository.CreateSession(ctx, session); err != nil {
		s.log.Logf("ERROR Auth: Error creating session: %v", err)
	} else {
		s.log.Logf("INFO Auth: Created session %s for user ID: %s", sessionID, user.ID)
	}

	// Link JWT to session for validation
	claims.User.SetStrAttr("sid", sessionID)

	// Set canonical user ID (real UUID - not provider-prefixed)
	claims.User.ID = user.ID.String()
	claims.User.Email = user.PrimaryEmail
	claims.User.Name = user.Name
	claims.User.SetStrAttr("email", user.PrimaryEmail)

	// Set role for RBAC
	claims.User.SetStrAttr("role", string(user.Role))

	// Store provider-prefixed ID separately for reference (if needed)
	claims.User.SetStrAttr("pid", providerUserID)

	// Store provider name explicitly
	claims.User.SetStrAttr("provider", provider)

	return claims
}
