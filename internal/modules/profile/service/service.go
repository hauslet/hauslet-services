package service

import (
	"context"

	"hauslet/internal/modules/profile/domain"
)

// ensureProfile fetches a profile by user ID, returning an error if not found.
func (s *ProfileServiceImpl) ensureProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	p, err := s.repo.GetProfileByUserID(ctx, userID)
	if err != nil {
		s.log.Logf("[ERROR] failed to fetch profile for user %s: %v", userID, err)
		return nil, err
	}
	if p == nil {
		s.log.Logf("[WARN] profile not found for user %s", userID)
		return nil, domain.ErrProfileNotFound
	}

	return domain.MapProfileFromSchema(p), nil
}

func (s *ProfileServiceImpl) ensureProfileByID(ctx context.Context, id string) (*domain.Profile, error) {
	if id == "" {
		return nil, domain.ErrInvalidUserID
	}

	p, err := s.repo.GetProfileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrProfileNotFound
	}

	return domain.MapProfileFromSchema(p), nil
}

func enforcePhoneLimit(numbers []string) error {
	if len(numbers) > 2 {
		return domain.ErrMaxPhoneNumbersReached
	}
	return nil
}
