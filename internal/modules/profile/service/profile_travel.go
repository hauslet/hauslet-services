package service

import (
	"context"
	"errors"

	"hauslet/internal/modules/profile/domain"
)

// UpdateTravelCompanion updates or adds a single travel companion for a user.
func (s *ProfileServiceImpl) UpdateTravelCompanion(ctx context.Context, userID string, companion domain.TravelCompanion) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}

	schemaCompanion := domain.MapTravelCompanionToSchema(companion)
	return s.repo.UpdateTravelCompanion(ctx, profile.UserID, schemaCompanion)
}

// AddTravelCompanion creates a new travel companion for a user.
func (s *ProfileServiceImpl) AddTravelCompanion(ctx context.Context, userID string, companion domain.TravelCompanion) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}

	schemaCompanion := domain.MapTravelCompanionToSchema(companion)
	return s.repo.AddTravelCompanion(ctx, profile.UserID, schemaCompanion)
}

// DeleteTravelCompanion removes a travel companion belonging to a user.
func (s *ProfileServiceImpl) DeleteTravelCompanion(ctx context.Context, userID string, companionID string) error {
	if userID == "" || companionID == "" {
		return errors.New("userID and companionID are required")
	}

	if _, err := s.ensureProfile(ctx, userID); err != nil {
		return err
	}

	return s.repo.DeleteTravelCompanion(ctx, userID, companionID)
}
