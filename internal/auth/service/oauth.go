package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"hauslet/internal/auth/domain"
	"hauslet/internal/auth/repository/schema"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
)

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

		TokenDuration:  time.Minute * 5, // JWT token expires in 5 minutes
		CookieDuration: time.Hour * 24,  // Cookie expires in 1 day
		Issuer:         "Hauslet",
		URL:            s.cfg.RedirectURL,
		AvatarStore:    avatar.NewLocalFS("/tmp"),
		SendJWTHeader:  false, // send JWT in header to simplify XSRF handling for clients

		// Validate users before allowing access
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			ctx := context.Background()
			if claims.User == nil || claims.User.ID == "" {
				return false
			}

			// Prefer real user ID stored in attributes (uid); fallback to User.ID
			userID := claims.User.StrAttr("uid")
			if userID == "" {
				userID = claims.User.ID
			}

			// STEP 1: Validate session exists in Redis
			sessionID := claims.User.StrAttr("sid")
			if sessionID == "" {
				// No session ID in token - reject (all tokens must have sid)
				log.Printf("WARN: Token rejected - missing session ID for user %s", userID)
				return false
			}

			session, err := s.repository.GetSessionByID(ctx, sessionID)
			if err != nil {
				// Redis error - log warning and fall back to user check
				log.Printf("WARN: Session validation failed (Redis error): %v - falling back to user check", err)
			} else if session == nil {
				// Session was explicitly deleted/revoked - reject immediately
				log.Printf("INFO: Token rejected - session %s was revoked for user %s", sessionID, userID)
				return false
			}
			// Session exists - continue to user validation

			// STEP 2: Check if user is active (existing logic)
			user, err := s.repository.GetUserByID(ctx, userID)
			if err != nil || user == nil {
				return false
			}

			return user.IsActive
		}),
	}

	service := auth.NewService(options)

	// Add Google OAuth provider
	service.AddProvider(
		"google",
		s.cfg.GoogleClientID,
		s.cfg.GoogleCLSecret,
	)

	// Add Direct (password) authentication provider
	service.AddDirectProvider("password", provider.CredCheckerFunc(func(user, password string) (ok bool, err error) {
		fmt.Println("Authenticating user:", user)
		ctx := context.Background()
		_, authErr := s.AuthenticatePassword(ctx, user, password)
		if authErr != nil {
			return false, nil // Invalid credentials
		}
		return true, nil
	}))

	return service
}

// enrichClaims is called after authentication (OAuth or Direct)
// It handles user creation/linking, session creation, and role assignment
func (s *AuthServiceImpl) enrichClaims(claims token.Claims) token.Claims {
	ctx := context.Background()

	// Extract auth info from claims
	if claims.User == nil {
		log.Println("Auth: No user in claims")
		return claims
	}

	// Detect provider type
	provider := "unknown"
	providerUserID := claims.User.ID
	email := claims.User.Email
	name := claims.User.Name

	// direct provider sets Name to submitted login; use it if email is missing
	if email == "" && name != "" {
		email = name
	}

	// Check if this is OAuth (format: "google_123456") or password (plain email)
	if idx := strings.Index(claims.User.ID, "_"); idx > 0 {
		// OAuth login
		provider = claims.User.ID[:idx]
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
			log.Printf("Auth: Error fetching user for password login: %v", err)
			return claims
		}

		log.Printf("Auth: User %s logged in via password", user.PrimaryEmail)

	} else {
		// OAUTH AUTHENTICATION
		// Step 1: Check if UserIdentity already exists for this provider
		identity, err := s.repository.GetUserIdentityByProvider(ctx, provider, providerUserID)
		if err != nil {
			log.Printf("OAuth: Error checking identity: %v", err)
			return claims
		}

		if identity != nil {
			// EXISTING OAUTH LOGIN
			user, err = s.repository.GetUserByID(ctx, identity.UserID.String())
			if err != nil || user == nil {
				log.Printf("OAuth: Error fetching user for existing identity: %v", err)
				return claims
			}

			// Update last used timestamp for this identity
			now := time.Now()
			identity.LastUsedAt = &now
			_ = s.repository.UpdateUserIdentity(ctx, identity)

			log.Printf("OAuth: Existing user %s logged in via %s", user.PrimaryEmail, provider)

		} else {
			// NEW OAUTH LOGIN
			// Step 2: Check if user exists by email
			user, err = s.repository.GetUserByEmail(ctx, email)
			if err != nil {
				log.Printf("OAuth: Error checking user by email: %v", err)
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
					log.Printf("OAuth: Error creating user+identity: %v", err)
					return claims
				}

				log.Printf("OAuth: Created new user %s (%s) via %s", user.Name, user.PrimaryEmail, provider)

			} else {
				// USER EXISTS - link new OAuth identity to existing user
				identity = &schema.UserIdentity{
					ID:            uuid.New(),
					UserID:        user.ID,
					Provider:      provider,
					ProviderID:    providerUserID,
					Email:         email,
					EmailVerified: true,
				}

				if err := s.repository.CreateUserIdentity(ctx, identity); err != nil {
					log.Printf("OAuth: Error linking identity to existing user: %v", err)
					return claims
				}

				log.Printf("OAuth: Linked %s identity to existing user %s", provider, user.PrimaryEmail)
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
		log.Printf("Auth: Error creating session: %v", err)
	} else {
		log.Printf("Auth: Created session %s for user %s", sessionID, user.PrimaryEmail)
	}

	// Link JWT to session for validation
	claims.User.SetStrAttr("sid", sessionID)

	// Enrich claims with our user data
	claims.User.ID = user.ID.String()
	claims.User.Email = user.PrimaryEmail
	claims.User.Name = user.Name
	claims.User.SetStrAttr("email", user.PrimaryEmail)

	// Set role for RBAC
	claims.User.SetStrAttr("role", string(user.Role))
	claims.User.SetStrAttr("uid", user.ID.String())
	// keep provider-prefixed ID so middleware provider check passes
	claims.User.ID = providerUserID
	return claims
}
