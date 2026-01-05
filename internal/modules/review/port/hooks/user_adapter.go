package hooks

import (
	"context"
	"hauslet/internal/modules/profile/service"

	"github.com/google/uuid"
)

// ReviewUserAdapter exposes user contact data for review notifications.
type ReviewUserAdapter struct {
	svc service.ProfileService
}

// NewReviewUserAdapter creates the adapter for review notifications.
func NewReviewUserAdapter(svc service.ProfileService) *ReviewUserAdapter {
	return &ReviewUserAdapter{svc: svc}
}

// GetUserContact returns the name and email for a user.
func (a *ReviewUserAdapter) GetUserContact(ctx context.Context, userID uuid.UUID) (string, string, error) {
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
