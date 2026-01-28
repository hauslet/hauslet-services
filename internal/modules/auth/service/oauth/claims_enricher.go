package oauth

import (
	"context"
	"net/url"
	"strings"

	"hauslet/internal/modules/auth/repository/schema"

	"github.com/go-pkgz/auth/v2/token"
)

type claimsEnricher struct {
	deps Dependencies
}

func newClaimsEnricher(deps Dependencies) *claimsEnricher {
	return &claimsEnricher{deps: deps}
}

func (c *claimsEnricher) EnrichClaims(claims token.Claims) token.Claims {
	ctx := context.Background()

	if claims.User == nil {
		c.deps.Log.Warn("Auth: No user in claims")
		return claims
	}

	isLinking, linkState := c.detectLinking(claims)
	providerUserID := claims.User.ID
	email := claims.User.Email
	name := claims.User.Name

	if email == "" && name != "" {
		email = name
	}

	provider := extractProvider(claims)

	var user *schema.User
	var err error

	switch provider {
	case "password":
		user, err = handlePasswordFlow(ctx, c.deps, claims)
	case "email":
		user, err = handleEmailFlow(ctx, c.deps, claims)
	default:
		user, err = handleOAuthFlow(ctx, c.deps, claims, provider, providerUserID, email, name, isLinking, linkState)
	}

	// Handle 2FA required case - return claims with 2FA pending attributes
	if err == ErrTwoFactorRequired {
		// The password flow has already set the 2FA attributes on claims.User
		// We need to keep claims.User so the attributes can be passed to the response
		// But we mark it as a 2FA pending state, not a full login
		c.deps.Log.Info("2FA verification required, returning pending state")

		// Set a special marker that the HTTP layer can detect
		claims.User.SetStrAttr("login_state", "2fa_pending")

		// Don't create a session - user is not fully authenticated yet
		// Don't set uid, role, etc. - those are for authenticated users
		return claims
	}

	// Handle auto-link blocked error - return claims with error for frontend display
	if err != nil && strings.Contains(err.Error(), "account with this email already exists") {
		c.deps.Log.Info("OAuth: Returning auto-link blocked error to client")
		claims.User.SetStrAttr("login_state", "auto_link_blocked")
		claims.User.SetStrAttr("error", "account_exists")
		claims.User.SetStrAttr("error_message", err.Error())
		return claims
	}

	if err != nil || user == nil {
		// Prevent issuing a token without a backed session/user (e.g., provider not linked)
		claims.User = nil
		return claims
	}

	sessionID := createSession(c.deps, user, provider, email)

	claims.User.SetStrAttr("sid", sessionID)
	claims.User.SetStrAttr("uid", user.ID.String()) // canonical user ID for API/database lookups
	claims.User.ID = providerUserID                 // keep provider-prefixed ID for go-pkgz/auth provider checks
	claims.User.Email = user.PrimaryEmail
	claims.User.Name = user.Name
	claims.User.SetStrAttr("email", user.PrimaryEmail)
	claims.User.SetStrAttr("role", string(user.Role))
	claims.User.Role = string(user.Role) // Also set top-level field for clean logging
	claims.User.SetStrAttr("pid", providerUserID)
	claims.User.SetStrAttr("provider", provider)

	// Attach avatar URL from profile (best-effort, non-blocking on errors)
	if c.deps.ProfileAvatarFetcher != nil {
		if avatarURL, err := c.deps.ProfileAvatarFetcher(ctx, user.ID.String()); err != nil {
			c.deps.Log.Warn("Auth: failed to fetch avatar for user", "user_id", user.ID.String(), "error", err)
		} else if avatarURL != nil && *avatarURL != "" {
			claims.User.SetStrAttr("avatar_url", *avatarURL)
			claims.User.Picture = *avatarURL
		}
	}

	return claims
}

func (c *claimsEnricher) detectLinking(claims token.Claims) (bool, *LinkState) {
	if c.deps.LinkStateValidator == nil {
		c.deps.Log.Info("detectLinking: LinkStateValidator is nil")
		return false, nil
	}

	// Link state is embedded in claims.Handshake.From as a query parameter
	// e.g., "/settings?link_state=link.{base64}.{sig}"
	if claims.Handshake == nil {
		c.deps.Log.Info("detectLinking: claims.Handshake is nil")
		return false, nil
	}

	if claims.Handshake.From == "" {
		c.deps.Log.Info("detectLinking: claims.Handshake.From is empty")
		return false, nil
	}

	c.deps.Log.Info("detectLinking: Handshake.From", "from", claims.Handshake.From)

	// Parse the From URL to extract link_state parameter
	fromURL, err := url.Parse(claims.Handshake.From)
	if err != nil {
		c.deps.Log.Info("detectLinking: failed to parse From URL", "error", err)
		return false, nil
	}

	stateToken := fromURL.Query().Get("link_state")
	c.deps.Log.Info("detectLinking: extracted link_state", "state_token", stateToken)

	if !strings.HasPrefix(stateToken, "link.") {
		c.deps.Log.Info("detectLinking: state token doesn't have link. prefix")
		return false, nil
	}

	linkState, err := c.deps.LinkStateValidator(stateToken)
	if err != nil {
		c.deps.Log.Error("OAuth Linking: Invalid state token", "error", err)
		return false, nil
	}

	c.deps.Log.Info("OAuth Linking: Detected link state", "user_id", linkState.UserID, "provider", linkState.Provider)
	return true, linkState
}

func extractProvider(claims token.Claims) string {
	provider := claims.User.StrAttr("provider")

	if provider == "" {
		if idx := strings.Index(claims.User.ID, "_"); idx > 0 {
			provider = claims.User.ID[:idx]
		}
	}

	if provider == "" || provider == "unknown" {
		provider = "password"
	}

	return provider
}
