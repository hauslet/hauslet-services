package service

import (
	"context"

	"hauslet/internal/modules/profile/domain"
)

// ListProfiles returns paginated profiles.
func (s *ProfileServiceImpl) ListProfiles(ctx context.Context, limit, offset int) ([]domain.Profile, error) {
	p, err := s.repo.ListProfiles(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return domain.MapProfilesFromSchema(p), nil
}

// SearchProfiles searches profiles by text query.
func (s *ProfileServiceImpl) SearchProfiles(ctx context.Context, query string, limit, offset int) ([]domain.Profile, error) {
	p, err := s.repo.SearchProfiles(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return domain.MapProfilesFromSchema(p), nil
}

// GetProfilesByUserType returns profiles filtered by user type.
func (s *ProfileServiceImpl) GetProfilesByUserType(ctx context.Context, userType string, limit, offset int) ([]domain.Profile, error) {
	p, err := s.repo.GetProfilesByUserType(ctx, userType, limit, offset)
	if err != nil {
		return nil, err
	}
	return domain.MapProfilesFromSchema(p), nil
}

// GetVerifiedProfiles returns verified profiles, optionally filtered by level.
func (s *ProfileServiceImpl) GetVerifiedProfiles(ctx context.Context, level string, limit, offset int) ([]domain.Profile, error) {
	p, err := s.repo.GetVerifiedProfiles(ctx, level, limit, offset)
	if err != nil {
		return nil, err
	}
	return domain.MapProfilesFromSchema(p), nil
}

// GetProfilesByUserIDs fetches profiles for multiple user IDs.
func (s *ProfileServiceImpl) GetProfilesByUserIDs(ctx context.Context, userIDs []string) ([]domain.Profile, error) {
	p, err := s.repo.GetProfilesByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	return domain.MapProfilesFromSchema(p), nil
}

// ProfileExists checks for profile existence by user ID.
func (s *ProfileServiceImpl) ProfileExists(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, domain.ErrInvalidUserID
	}
	return s.repo.ProfileExists(ctx, userID)
}
