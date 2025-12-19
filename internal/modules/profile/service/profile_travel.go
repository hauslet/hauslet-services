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
	if err := s.repo.UpdateTravelCompanion(ctx, profile.UserID, schemaCompanion); err != nil {
		s.log.Logf("[ERROR] failed to update travel companion %s for user %s: %v", companion.ID, userID, err)
		return err
	}
	s.log.Logf("[INFO] updated travel companion %s for user %s", companion.ID, userID)
	return nil
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
	if err := s.repo.AddTravelCompanion(ctx, profile.UserID, schemaCompanion); err != nil {
		s.log.Logf("[ERROR] failed to add travel companion for user %s: %v", userID, err)
		return err
	}
	s.log.Logf("[INFO] added travel companion %s for user %s", companion.ID, userID)
	return nil
}

// DeleteTravelCompanion removes a travel companion belonging to a user.
func (s *ProfileServiceImpl) DeleteTravelCompanion(ctx context.Context, userID string, companionID string) error {
	if userID == "" || companionID == "" {
		return errors.New("userID and companionID are required")
	}

	if _, err := s.ensureProfile(ctx, userID); err != nil {
		return err
	}

	if err := s.repo.DeleteTravelCompanion(ctx, userID, companionID); err != nil {
		s.log.Logf("[ERROR] failed to delete travel companion %s for user %s: %v", companionID, userID, err)
		return err
	}
	s.log.Logf("[INFO] deleted travel companion %s for user %s", companionID, userID)
	return nil
}
