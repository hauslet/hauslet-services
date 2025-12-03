package graphql

import (
	"context"
	"errors"

	"hauslet/internal/auth/domain"
	"hauslet/internal/auth/service"
	"hauslet/internal/graph/viewer"
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
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, errors.New("unauthenticated")
	}

	return r.authService.GetUser(ctx, v.UserID)
}
