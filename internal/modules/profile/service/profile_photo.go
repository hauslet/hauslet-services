package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/profile/domain"

	"github.com/google/uuid"
)

// AddProfilePhoto updates/attaches a photo to the user's profile.
func (s *ProfileServiceImpl) UploadProfilePhoto(ctx context.Context, userID, filename string) (*domain.UploadResult, error) {
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	_, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	objKey := fmt.Sprintf("profiles/%s/profile_photos/%s", userID, time.Now().Format("20060102T150405")+"_"+filename)
	url, err := s.storage.GenerateSignedUploadURL(ctx, objKey, "image/jpeg", 15*time.Minute)
	if err != nil {
		s.log.Logf("[ERROR] failed to generate upload URL for profile photo (user: %s): %v", userID, err)
		return nil, err
	}

	s.log.Logf("[INFO] generated profile photo upload URL for user %s (key: %s)", userID, objKey)

	result := &domain.UploadResult{
		UserID:   userID,
		Filename: filename,
		URL:      url,
		Key:      objKey,
	}
	return result, nil
}

// UploadTravelCompanionPhoto adds a photo to a specific travel companion in the user's profile.
func (s *ProfileServiceImpl) UploadTravelCompanionPhoto(ctx context.Context, companionID uuid.UUID, userID, filename string) (*domain.UploadResult, error) {
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}

	_, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	c, err := s.repo.GetTravelCompanionByID(ctx, userID, companionID)
	if err != nil {
		s.log.Logf("[ERROR] failed to fetch travel companion %s for user %s: %v", companionID, userID, err)
		return nil, err
	}
	if c == nil {
		s.log.Logf("[WARN] travel companion %s not found for user %s", companionID, userID)
		return nil, domain.ErrTravelCompanionNotFound
	}

	objKey := fmt.Sprintf("profiles/%s/travel_companions/%s/photos/%s", userID, companionID.String(),
		time.Now().Format("20060102T150405")+"_"+filename)
	url, err := s.storage.GenerateSignedUploadURL(ctx, objKey, "image/jpeg", 15*time.Minute)
	if err != nil {
		s.log.Logf("[ERROR] failed to generate upload URL for companion photo (user: %s, companion: %s): %v",
			userID, companionID, err)
		return nil, err
	}

	s.log.Logf("[INFO] generated companion photo upload URL for user %s (companion: %s, key: %s)",
		userID, companionID, objKey)

	result := &domain.UploadResult{
		TravelCompanionID: companionID,
		UserID:            userID,
		Filename:          filename,
		URL:               url,
		Key:               objKey,
	}

	return result, nil
}
