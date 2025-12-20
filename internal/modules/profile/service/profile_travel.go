package service

import (
	"context"
	"errors"
	"fmt"

	"hauslet/internal/modules/profile/domain"

	"github.com/google/uuid"
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

	if companion.Name != "" {
		// Enqueue name moderation
		payload := fmt.Sprintf("{travel_companion_name: %s}", companion.Name)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "profile_bio", payload); err != nil {
			s.log.Logf("[ERROR] failed to enqueue travel companion name moderation for user %s companion %s: %v",
				userID, companion.ID, err)
			return err
		}
	}

	if companion.PhotoURL != nil && *companion.PhotoURL != "" {
		// Enqueue photo moderation
		payload := fmt.Sprintf("{travel_companion_image_key: %s}", *companion.PhotoURL)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "tc_image", payload); err != nil {
			s.log.Logf("[ERROR] failed to enqueue travel companion photo moderation for user %s companion %s: %v",
				userID, companion.ID, err)
			return err
		}
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

	if companion.Name != "" {
		// Enqueue name moderation
		payload := fmt.Sprintf("{travel_companion_name: %s}", companion.Name)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "profile_bio", payload); err != nil {
			s.log.Logf("[ERROR] failed to enqueue travel companion name moderation for user %s companion %s: %v",
				userID, companion.ID, err)
			return err
		}
	}

	if companion.PhotoURL != nil && *companion.PhotoURL != "" {
		// Enqueue photo moderation
		payload := fmt.Sprintf("{travel_companion_image_key: %s}", *companion.PhotoURL)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "tc_image", payload); err != nil {
			s.log.Logf("[ERROR] failed to enqueue travel companion photo moderation for user %s companion %s: %v",
				userID, companion.ID, err)
			return err
		}
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

	companionIDUUID, err := uuid.Parse(companionID)
	if err != nil {
		s.log.Logf("[ERROR] invalid companion ID %s for user %s: %v", companionID, userID, err)
		return errors.New("invalid companion ID format")
	}
	tc, err := s.repo.GetTravelCompanionByID(ctx, userID, companionIDUUID)
	if err != nil {
		return err
	}
	if tc == nil {
		s.log.Logf("[WARN] travel companion %s not found for user %s", companionID, userID)
		return domain.ErrTravelCompanionNotFound
	}

	if err := s.repo.DeleteTravelCompanion(ctx, userID, companionID); err != nil {
		s.log.Logf("[ERROR] failed to delete travel companion %s for user %s: %v", companionID, userID, err)
		return err
	}
	go func() {
		// Delete photo from storage if exists
		if tc.PhotoURL != nil && *tc.PhotoURL != "" {
			if err := s.storage.DeleteObject(context.Background(), *tc.PhotoURL); err != nil {
				s.log.Logf("[ERROR] failed to delete travel companion photo %s for user %s: %v", *tc.PhotoURL, userID, err)
			}
		}
	}()
	s.log.Logf("[INFO] deleted travel companion %s for user %s", companionID, userID)
	return nil
}
