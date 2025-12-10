package port

import (
	"context"
	"time"

	"hauslet/internal/modules/profile/domain"
	"hauslet/internal/modules/profile/service"
)

// AuthHooksAdapter translates calls from the Auth module into Profile domain logic.
type AuthHooksAdapter struct {
	svc service.ProfileService
}

// NewAuthHooksAdapter creates the adapter
func NewAuthHooksAdapter(svc service.ProfileService) *AuthHooksAdapter {
	return &AuthHooksAdapter{svc: svc}
}

// CreateDefaultProfile satisfies the Auth module's "ProfileHooks" interface
func (a *AuthHooksAdapter) CreateDefaultProfile(ctx context.Context, userID string, name string, birthDate *time.Time) error {
	// 1. Translate "scalars" (strings/dates) into "Domain Object"
	newProfile := domain.Profile{
		UserID:    userID,
		UserTypes: []domain.UserType{domain.Guest},
		FullName:  name,
		BirthDate: birthDate,
	}

	// 2. Call the internal service
	_, err := a.svc.CreateProfile(ctx, newProfile)
	return err
}
