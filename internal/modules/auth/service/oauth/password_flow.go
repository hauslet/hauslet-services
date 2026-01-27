package oauth

import (
	"context"
	"errors"
	"fmt"

	"hauslet/internal/modules/auth/repository/schema"

	"github.com/go-pkgz/auth/v2/token"
)

// ErrTwoFactorRequired is a sentinel error indicating 2FA is required
var ErrTwoFactorRequired = errors.New("2FA_REQUIRED")

// Pending2FAInfo contains information about the pending 2FA verification
type Pending2FAInfo struct {
	TempToken string
	Method    TwoFactorMethod
	Email     string
}

// pendingTwoFAContext is a context key for storing pending 2FA info
type pendingTwoFAContextKey struct{}

// SetPending2FAInfo stores pending 2FA info in context (for claims enricher to access)
func SetPending2FAInfo(ctx context.Context, info *Pending2FAInfo) context.Context {
	return context.WithValue(ctx, pendingTwoFAContextKey{}, info)
}

// GetPending2FAInfo retrieves pending 2FA info from context
func GetPending2FAInfo(ctx context.Context) *Pending2FAInfo {
	if info, ok := ctx.Value(pendingTwoFAContextKey{}).(*Pending2FAInfo); ok {
		return info
	}
	return nil
}

func handlePasswordFlow(ctx context.Context, deps Dependencies, claims token.Claims) (*schema.User, error) {
	// Prefer explicit email, then attribute, then name (legacy), with validation.
	var email string
	switch {
	case claims.User.Email != "":
		email = claims.User.Email
	case claims.User.Attributes != nil:
		if attrEmail, ok := claims.User.Attributes["email"]; ok {
			if v, ok := attrEmail.(string); ok {
				email = v
			}
		}
	}
	// Last resort: fall back to Name if nothing else is present.
	if email == "" {
		email = claims.User.Name
	}
	if email == "" {
		deps.Log.Error("Password auth failed - missing email in claims")
		return nil, fmt.Errorf("missing email")
	}

	user, err := deps.Repository.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		deps.Log.Error("Auth: Error fetching user for password login", "error", err)
		return nil, fmt.Errorf("user not found")
	}

	// Skip 2FA check if this is a post-2FA verification token issuance
	// (the claims will have "2fa_verified" attribute set by the Verify2FALogin handler)
	if claims.User.StrAttr("2fa_verified") == "true" {
		deps.Log.Info("Skipping 2FA check - already verified", "user_id", user.ID)
		return user, nil
	}

	// Check if user has 2FA enabled
	if deps.GetUser2FA != nil {
		enabled, method, err := deps.GetUser2FA(ctx, user.ID.String())
		if err != nil {
			deps.Log.Warn("Failed to check 2FA status", "user_id", user.ID, "error", err)
			// Continue without 2FA on error (fail open for this check)
		} else if enabled {
			deps.Log.Info("2FA required for user", "user_id", user.ID, "method", method)

			// Create pending state and get temp token
			if deps.Create2FAPendingState != nil {
				tempToken, err := deps.Create2FAPendingState(ctx, user.ID.String(), user.PrimaryEmail, user.Name, "password", method)
				if err != nil {
					deps.Log.Error("Failed to create 2FA pending state", "error", err)
					return nil, fmt.Errorf("failed to initiate 2FA: %w", err)
				}

				// Send 2FA code for email/SMS methods
				if method != TwoFactorMethod("authenticator") && deps.Send2FACode != nil {
					if err := deps.Send2FACode(ctx, user.ID.String()); err != nil {
						deps.Log.Warn("Failed to send 2FA code", "user_id", user.ID, "error", err)
						// Continue anyway - user can request resend
					}
				}

				// Store pending 2FA info in user attributes for the HTTP layer to detect
				claims.User.SetStrAttr("2fa_pending", "true")
				claims.User.SetStrAttr("2fa_temp_token", tempToken)
				claims.User.SetStrAttr("2fa_method", string(method))
				claims.User.SetStrAttr("2fa_email", user.PrimaryEmail)

				// Return the special 2FA error
				return nil, ErrTwoFactorRequired
			}
		}
	}

	deps.Log.Info("Auth: User",
		"user", user.ID,
		"email", maskEmail(user.PrimaryEmail),
		"method", "password")
	return user, nil
}
