package oauth

import (
	"context"
	"strings"

	"hauslet/internal/modules/auth/repository/schema"

	"github.com/go-pkgz/auth/token"
)

type claimsEnricher struct {
	deps Dependencies
}

func newClaimsEnricher(deps Dependencies) *claimsEnricher {
	return &claimsEnricher{deps: deps}
}

// maskEmail masks email addresses for non-debug logs to reduce PII exposure.
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

	if provider == "password" {
		user, err = handlePasswordFlow(ctx, c.deps, claims)
	} else {
		user, err = handleOAuthFlow(ctx, c.deps, claims, provider, providerUserID, email, name, isLinking, linkState)
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
		return false, nil
	}

	if claims.User == nil {
		return false, nil
	}

	stateToken := claims.User.StrAttr("state")
	if !strings.HasPrefix(stateToken, "link.") {
		return false, nil
	}

	linkState, err := c.deps.LinkStateValidator(stateToken)
	if err != nil {
		c.deps.Log.Error("OAuth Linking: Invalid state token", "error", err)
		return false, nil
	}

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
