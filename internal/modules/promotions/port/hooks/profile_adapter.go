package hooks

import (
	"context"
	"hauslet/internal/modules/profile/service"

	"github.com/google/uuid"
)

// PromotionProfileAdapter exposes minimal profile data to the promotions module without coupling it to the full service.
type PromotionProfileAdapter struct {
	svc service.ProfileService
}

// NewPromotionProfileAdapter creates the adapter for promotions module.
func NewPromotionProfileAdapter(svc service.ProfileService) *PromotionProfileAdapter {
	return &PromotionProfileAdapter{svc: svc}
}

// GetProfileData returns the full name and email of the user profile.
func (a *PromotionProfileAdapter) GetProfileData(ctx context.Context, userID uuid.UUID) (string, string, error) {
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
