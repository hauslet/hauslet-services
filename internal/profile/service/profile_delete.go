package service

import (
	"context"

	"hauslet/internal/profile/domain"
)

// DeleteProfile performs a soft delete.
func (s *ProfileServiceImpl) DeleteProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	return s.repo.DeleteProfile(ctx, userID)
}

// RestoreProfile restores a soft-deleted profile.
func (s *ProfileServiceImpl) RestoreProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	return s.repo.RestoreProfile(ctx, userID)
}

// HardDeleteProfile permanently deletes a profile.
func (s *ProfileServiceImpl) HardDeleteProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	return s.repo.HardDeleteProfile(ctx, userID)
}

// GetDeletedProfiles returns soft-deleted profiles.
func (s *ProfileServiceImpl) GetDeletedProfiles(ctx context.Context) ([]domain.Profile, error) {
	p, err := s.repo.GetDeletedProfiles(ctx)
	if err != nil {
		return nil, err
	}
	return domain.MapProfilesFromSchema(p), nil
}
