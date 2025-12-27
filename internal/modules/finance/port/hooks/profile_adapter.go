package hooks

import (
	"context"
	"hauslet/internal/modules/profile/service"

	"github.com/google/uuid"
)

// FinanceProfileAdapter exposes minimal profile data to the finance module without coupling it to the full service.
type FinanceProfileAdapter struct {
	svc service.ProfileService
}

// NewFinanceProfileAdapter creates the adapter for finance notifications.
func NewFinanceProfileAdapter(svc service.ProfileService) *FinanceProfileAdapter {
	return &FinanceProfileAdapter{svc: svc}
}

// GetProfileData returns the full name and email of the user profile.
func (a *FinanceProfileAdapter) GetProfileData(ctx context.Context, userID uuid.UUID) (string, string, error) {
	profile, err := a.svc.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		return "", "", err
	}
	if profile == nil {
		return "", "", nil
	}
	if profile.Email == nil {
		return profile.FullName, "", nil
	}
	return profile.FullName, *profile.Email, nil
}
