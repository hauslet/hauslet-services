package port

import (
	"context"
	"hauslet/internal/modules/profile/service"
)

// PropertyProfileAdapter exposes minimal profile data to the property module without coupling it to the full service.
type PropertyProfileAdapter struct {
	svc service.ProfileService
}

// NewPropertyProfileAdapter creates the adapter for property notifications.
func NewPropertyProfileAdapter(svc service.ProfileService) *PropertyProfileAdapter {
	return &PropertyProfileAdapter{svc: svc}
}

// GetProfileData returns the full name and email of the user profile.
func (a *PropertyProfileAdapter) GetProfileData(ctx context.Context, userID string) (string, string, error) {
	profile, err := a.svc.GetProfileByUserID(ctx, userID)
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
