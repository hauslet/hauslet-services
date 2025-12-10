package service

import (
	"context"

	"hauslet/internal/modules/profile/domain"
)

// UpdateTravelCompanions replaces travel companion data.
func (s *ProfileServiceImpl) UpdateTravelCompanions(ctx context.Context, userID string, companions []domain.TravelCompanion) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}

	profile.TravelCompanions = companions

	schemaProfile, err := domain.MapProfileToSchema(profile)
	if err != nil {
		return err
	}

	return s.repo.UpdateProfile(ctx, schemaProfile)
}
