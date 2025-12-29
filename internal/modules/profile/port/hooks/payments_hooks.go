package hooks

import (
	"context"
	"hauslet/internal/modules/profile/service"
)

// PaymentsProfileAdapter exposes minimal profile data to the payments module.
type PaymentsProfileAdapter struct {
	svc service.ProfileService
}

// NewPaymentsProfileAdapter creates the adapter for payments notifications.
func NewPaymentsProfileAdapter(svc service.ProfileService) *PaymentsProfileAdapter {
	return &PaymentsProfileAdapter{svc: svc}
}

// GetUserContactEmail returns the profile email for the given user ID.
func (a *PaymentsProfileAdapter) GetUserContactEmail(ctx context.Context, userID string) (string, error) {
	profile, err := a.svc.GetProfileByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if profile == nil || profile.Email == nil {
		return "", nil
	}
	return *profile.Email, nil
}
