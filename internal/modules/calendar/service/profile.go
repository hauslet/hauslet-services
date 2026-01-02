package service

import (
	"context"
	"hauslet/internal/modules/calendar/domain"

	"github.com/google/uuid"
)

// GetUserProfile fetches user profile data via ProfileHooks.
func (s *CalendarServiceImpl) GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	if s.profileHooks == nil {
		return nil, domain.ErrProfileHooksNotConfigured
	}

	profile, err := s.profileHooks.GetUserProfile(ctx, userID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get user profile", "user_id", userID, "error", err)
		}
		return nil, err
	}

	return profile, nil
}
