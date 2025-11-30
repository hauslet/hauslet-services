package service

import (
	"context"

	"hauslet/internal/profile/domain"
)

// CreateProfile creates a new profile with validation and defaults.
func (s *ProfileServiceImpl) CreateProfile(ctx context.Context, profile domain.Profile) (*domain.Profile, error) {
	if profile.UserID == "" {
		return nil, domain.ErrInvalidUserID
	}
	if err := enforcePhoneLimit(profile.PhoneNumbers); err != nil {
		return nil, err
	}

	exists, err := s.repo.ProfileExists(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrProfileAlreadyExists
	}

	// Default user types if none provided
	if len(profile.UserTypes) == 0 {
		profile.UserTypes = []domain.UserType{domain.Guest}
	}

	// Normalize nil slices
	if profile.PhoneNumbers == nil {
		profile.PhoneNumbers = []string{}
	}
	if profile.Skills == nil {
		profile.Skills = []string{}
	}
	if profile.Languages == nil {
		profile.Languages = []string{}
	}
	if profile.Interests == nil {
		profile.Interests = []string{}
	}
	if profile.Hobbies == nil {
		profile.Hobbies = []string{}
	}
	if profile.Badges == nil {
		profile.Badges = []domain.Badge{}
	}
	if profile.TravelCompanions == nil {
		profile.TravelCompanions = []domain.TravelCompanion{}
	}

	profile.TrustScore = profile.CalculateTrustScore()

	schemaProfile, err := domain.MapProfileToSchema(&profile)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreateProfile(ctx, schemaProfile); err != nil {
		return nil, err
	}

	return domain.MapProfileFromSchema(schemaProfile), nil
}

// GetProfileByUserID fetches a profile by user ID.
func (s *ProfileServiceImpl) GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	return s.ensureProfile(ctx, userID)
}

// GetProfileByID fetches a profile by its ID.
func (s *ProfileServiceImpl) GetProfileByID(ctx context.Context, id string) (*domain.Profile, error) {
	return s.ensureProfileByID(ctx, id)
}

// UpdateProfile updates a profile record with validation and trust score recalculation.
func (s *ProfileServiceImpl) UpdateProfile(ctx context.Context, profile domain.Profile) (*domain.Profile, error) {
	if profile.UserID == "" {
		return nil, domain.ErrInvalidUserID
	}
	if err := enforcePhoneLimit(profile.PhoneNumbers); err != nil {
		return nil, err
	}

	existing, err := s.ensureProfile(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}

	profile.ID = existing.ID
	if len(profile.UserTypes) == 0 {
		profile.UserTypes = existing.UserTypes
	}

	profile.TrustScore = profile.CalculateTrustScore()

	schemaProfile, err := domain.MapProfileToSchema(&profile)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateProfile(ctx, schemaProfile); err != nil {
		return nil, err
	}

	return domain.MapProfileFromSchema(schemaProfile), nil
}

// PatchProfile applies partial updates then re-syncs trust score.
func (s *ProfileServiceImpl) PatchProfile(ctx context.Context, id string, updates map[string]interface{}) (*domain.Profile, error) {
	if err := s.repo.PatchProfile(ctx, id, updates); err != nil {
		return nil, err
	}

	updated, err := s.ensureProfileByID(ctx, id)
	if err != nil {
		return nil, err
	}

	newScore := updated.CalculateTrustScore()
	if err := s.repo.UpdateTrustScore(ctx, updated.UserID, newScore); err != nil {
		return nil, err
	}
	updated.TrustScore = newScore

	return updated, nil
}
