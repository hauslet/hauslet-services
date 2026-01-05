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
		s.log.Error("failed to update travel companion", "companion_id", companion.ID, "user_id", userID, "error", err)
		return err
	}

	if companion.Name != "" {
		// Enqueue name moderation
		payload := fmt.Sprintf("{travel_companion_name: %s}", companion.Name)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "profile_bio", payload); err != nil {
			s.log.Error("failed to enqueue travel companion name moderation", "user_id", userID, "companion_id", companion.ID, "error", err)
			return err
		}
	}

	if companion.PhotoURL != nil && *companion.PhotoURL != "" {
		// Enqueue photo moderation
		payload := fmt.Sprintf("{travel_companion_image_key: %s}", *companion.PhotoURL)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "tc_image", payload); err != nil {
			s.log.Error("failed to enqueue travel companion photo moderation", "user_id", userID, "companion_id", companion.ID, "error", err)
			return err
		}
	}

	s.log.Info("updated travel companion", "companion_id", companion.ID, "user_id", userID)
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
		s.log.Error("failed to add travel companion", "user_id", userID, "error", err)
		return err
	}

	if companion.Name != "" {
		// Enqueue name moderation
		payload := fmt.Sprintf("{travel_companion_name: %s}", companion.Name)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "profile_bio", payload); err != nil {
			s.log.Error("failed to enqueue travel companion name moderation", "user_id", userID, "companion_id", companion.ID, "error", err)
			return err
		}
	}

	if companion.PhotoURL != nil && *companion.PhotoURL != "" {
		// Enqueue photo moderation
		payload := fmt.Sprintf("{travel_companion_image_key: %s}", *companion.PhotoURL)
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "tc_image", payload); err != nil {
			s.log.Error("failed to enqueue travel companion photo moderation", "user_id", userID, "companion_id", companion.ID, "error", err)
			return err
		}
	}
	s.log.Info("added travel companion", "companion_id", companion.ID, "user_id",	 userID)
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
		s.log.Error("invalid companion ID", "companion_id", companionID, "user_id", userID, "error", err)
		return errors.New("invalid companion ID format")
	}
	tc, err := s.repo.GetTravelCompanionByID(ctx, userID, companionIDUUID)
	if err != nil {
		return err
	}
	if tc == nil {
		s.log.Warn("travel companion not found", "companion_id", companionID, "user_id", userID)
		return domain.ErrTravelCompanionNotFound
	}

	if err := s.repo.DeleteTravelCompanion(ctx, userID, companionID); err != nil {
		s.log.Error("failed to delete travel companion", "companion_id", companionID, "user_id", userID, "error", err)
		return err
	}
	go func() {
		// Delete photo from storage if exists
		if tc.PhotoURL != nil && *tc.PhotoURL != "" {
			if err := s.storage.DeleteObject(context.Background(), *tc.PhotoURL); err != nil {
				s.log.Error("failed to delete travel companion photo", "photo_url", *tc.PhotoURL, "user_id", userID, "error", err)
			}
		}
	}()
	s.log.Info("deleted travel companion", "companion_id", companionID, "user_id", userID)
	return nil
}
