package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"hauslet/internal/modules/profile/domain"
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
		s.log.Logf("[ERROR] failed to check profile existence for user %s: %v", profile.UserID, err)
		return nil, err
	}
	if exists {
		s.log.Logf("[WARN] profile already exists for user %s", profile.UserID)
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
		s.log.Logf("[ERROR] failed to create profile for user %s: %v", profile.UserID, err)
		return nil, err
	}

	s.log.Logf("[INFO] created profile for user %s  profileID=%s (trust score: %.2f)", profile.UserID, profile.ID, profile.TrustScore)
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
		s.log.Logf("[ERROR] failed to update profile for user %s: %v", profile.UserID, err)
		return nil, err
	}

	payload := domain.ProfileModerationPayload{
		FullName:       profile.FullName,
		Occupation:     derefString(profile.Occupation),
		Education:      derefString(profile.Education),
		Bio:            derefString(profile.Bio),
		Skills:         profile.Skills,
		Languages:      profile.Languages,
		Interests:      profile.Interests,
		Hobbies:        profile.Hobbies,
		FunFact:        derefString(profile.FunFact),
		ObsessedWith:   derefString(profile.ObsessedWith),
		City:           derefString(profile.City),
		State:          derefString(profile.State),
		Country:        derefString(profile.Country),
		HouseNumber:    derefString(profile.HouseNumber),
		Street:         derefString(profile.Street),
		Area:           derefString(profile.Area),
		LGA:            derefString(profile.LGA),
		District:       derefString(profile.District),
		DigitalAddress: derefString(profile.DigitalAddress),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize profile payload for moderation: %w", err)
	}
	// Enqueue profile text data moderation
	if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "profile_bio", string(payloadJSON)); err != nil {
		s.log.Logf("[ERROR] failed to enqueue profile moderation for user %s: %v", profile.UserID, err)
		return nil, err
	}

	if profile.PhotoURL != nil && *profile.PhotoURL != "" {
		// Enqueue profile photo moderation
		if err := s.moderationHooks.EnqueueAIModeration(ctx, profile.ID, "profile_image", *profile.PhotoURL); err != nil {
			s.log.Logf("[ERROR] failed to enqueue profile photo moderation for user %s: %v", profile.UserID, err)
			return nil, err
		}
	}

	s.log.Logf("[INFO] updated profile for user %s (new trust score: %.2f)", profile.UserID, profile.TrustScore)
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

	payload := domain.ProfileModerationPayload{
		FullName:       updated.FullName,
		Occupation:     derefString(updated.Occupation),
		Education:      derefString(updated.Education),
		Bio:            derefString(updated.Bio),
		Skills:         updated.Skills,
		Languages:      updated.Languages,
		Interests:      updated.Interests,
		Hobbies:        updated.Hobbies,
		FunFact:        derefString(updated.FunFact),
		ObsessedWith:   derefString(updated.ObsessedWith),
		City:           derefString(updated.City),
		State:          derefString(updated.State),
		Country:        derefString(updated.Country),
		HouseNumber:    derefString(updated.HouseNumber),
		Street:         derefString(updated.Street),
		Area:           derefString(updated.Area),
		LGA:            derefString(updated.LGA),
		District:       derefString(updated.District),
		DigitalAddress: derefString(updated.DigitalAddress),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize profile payload for moderation: %w", err)
	}
	// Enqueue profile text data moderation
	if err := s.moderationHooks.EnqueueAIModeration(ctx, updated.ID, "profile_bio", string(payloadJSON)); err != nil {
		s.log.Logf("[ERROR] failed to enqueue profile moderation for user %s: %v", updated.UserID, err)
		return nil, err
	}

	if updated.PhotoURL != nil && *updated.PhotoURL != "" {
		// Enqueue profile photo moderation
		if err := s.moderationHooks.EnqueueAIModeration(ctx, updated.ID, "profile_image", *updated.PhotoURL); err != nil {
			s.log.Logf("[ERROR] failed to enqueue profile photo moderation for user %s: %v", updated.UserID, err)
			return nil, err
		}
	}

	return updated, nil
}

// DeleteProfile performs a soft delete.
func (s *ProfileServiceImpl) DeleteProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	if err := s.repo.DeleteProfile(ctx, userID); err != nil {
		s.log.Logf("[ERROR] failed to delete profile for user %s: %v", userID, err)
		return err
	}
	s.log.Logf("[INFO] soft deleted profile for user %s", userID)
	return nil
}

// RestoreProfile restores a soft-deleted profile.
func (s *ProfileServiceImpl) RestoreProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	if err := s.repo.RestoreProfile(ctx, userID); err != nil {
		s.log.Logf("[ERROR] failed to restore profile for user %s: %v", userID, err)
		return err
	}
	s.log.Logf("[INFO] restored profile for user %s", userID)
	return nil
}

// HardDeleteProfile permanently deletes a profile.
func (s *ProfileServiceImpl) HardDeleteProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	if err := s.repo.HardDeleteProfile(ctx, userID); err != nil {
		s.log.Logf("[ERROR] failed to hard delete profile for user %s: %v", userID, err)
		return err
	}
	s.log.Logf("[WARN] permanently deleted profile for user %s", userID)
	return nil
}

// GetDeletedProfiles returns soft-deleted profiles.
func (s *ProfileServiceImpl) GetDeletedProfiles(ctx context.Context) ([]domain.Profile, error) {
	p, err := s.repo.GetDeletedProfiles(ctx)
	if err != nil {
		return nil, err
	}
	return domain.MapProfilesFromSchema(p), nil
}

func derefString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func sliceToString(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	return strings.Join(slice, ", ")
}
