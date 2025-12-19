package hooks

import (
	"context"

	"hauslet/internal/modules/profile/service"
)

// BusinessProfileAdapter exposes minimal profile data to the business module without coupling it to the full service.
type BusinessProfileAdapter struct {
	svc service.ProfileService
}

// NewBusinessProfileAdapter creates the adapter for business notifications.
func NewBusinessProfileAdapter(svc service.ProfileService) *BusinessProfileAdapter {
	return &BusinessProfileAdapter{svc: svc}
}

// GetProfileName returns the user's full name if available.
func (a *BusinessProfileAdapter) GetProfileName(ctx context.Context, userID string) (string, error) {
	profile, err := a.svc.GetProfileByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if profile == nil {
		return "", nil
	}
	return profile.FullName, nil
}
