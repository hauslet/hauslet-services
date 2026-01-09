package graphql

import (
	"context"
	"errors"

	"hauslet/internal/modules/auth/domain"
	"hauslet/internal/modules/auth/service"
	"hauslet/internal/transport/graph/viewer"
)

// Resolver handles auth-specific GraphQL fields.
type Resolver struct {
	authService service.AuthService
}

func NewResolver(authService service.AuthService) *Resolver {
	return &Resolver{authService: authService}
}

// Me resolves the currently authenticated user.
func (r *Resolver) Me(ctx context.Context) (*domain.User, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, errors.New("unauthenticated")
	}

	return r.authService.GetUser(ctx, userID.String())
}
