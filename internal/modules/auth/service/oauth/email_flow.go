package oauth

import (
	"context"
	"fmt"

	"hauslet/internal/modules/auth/repository/schema"

	"github.com/go-pkgz/auth/v2/token"
)

// handleEmailFlow handles passwordless email login via the verified provider.
// Unlike OAuth, this does NOT create new users - the user must already exist.
func handleEmailFlow(ctx context.Context, deps Dependencies, claims token.Claims) (*schema.User, error) {
	// Extract email from claims - the verified provider sets this from the address
	email := claims.User.Email
	if email == "" {
		// Fallback to name (the verified provider may set user info in Name)
		email = claims.User.Name
	}
	if email == "" {
		deps.Log.Error("Passwordless auth failed - missing email in claims")
		return nil, fmt.Errorf("missing email")
	}

	user, err := deps.Repository.GetUserByEmail(ctx, email)
	if err != nil {
		deps.Log.Error("Passwordless auth: Error fetching user", "error", err)
		return nil, fmt.Errorf("authentication failed")
	}
	if user == nil {
		deps.Log.Warn("Passwordless auth: User not found", "email", maskEmail(email))
		return nil, fmt.Errorf("user not found")
	}

	deps.Log.Info("Passwordless auth: User authenticated",
		"user_id", user.ID,
		"email", maskEmail(user.PrimaryEmail),
		"method", "email")

	return ensureActiveOnLogin(ctx, deps, user)
}
