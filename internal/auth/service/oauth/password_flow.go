package oauth

import (
	"context"
	"fmt"

	"hauslet/internal/auth/repository/schema"

	"github.com/go-pkgz/auth/token"
)

func handlePasswordFlow(ctx context.Context, deps Dependencies, claims token.Claims) (*schema.User, error) {
	email := claims.User.Name
	if email == "" {
		deps.Log.Logf("ERROR Password auth failed - missing email")
		return nil, fmt.Errorf("missing email")
	}

	user, err := deps.Repository.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		deps.Log.Logf("ERROR Auth: Error fetching user for password login: %v", err)
		return nil, fmt.Errorf("user not found")
	}

	deps.Log.Logf("INFO Auth: User %s (ID: %s) logged in via password", maskEmail(user.PrimaryEmail), user.ID)
	return user, nil
}
