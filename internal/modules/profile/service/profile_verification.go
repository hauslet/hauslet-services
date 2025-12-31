package service

import (
	"context"
	"errors"
	"time"

	"hauslet/internal/modules/profile/domain"
)

// UpdateTrustScore recomputes trust score from persisted fields.
func (s *ProfileServiceImpl) UpdateTrustScore(ctx context.Context, userID string, _ float64) error {
	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}
	score := profile.CalculateTrustScore()
	return s.repo.UpdateTrustScore(ctx, userID, score)
}

// IncrementReviewStats updates rating/reviews and trust score.
func (s *ProfileServiceImpl) IncrementReviewStats(ctx context.Context, userID string, ratingDelta float64, reviewsDelta int) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	if ratingDelta < 0 || ratingDelta > 5 {
		return domain.ErrInvalidRating
	}
	if reviewsDelta <= 0 {
		return errors.New("reviewsDelta must be positive")
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}

	totalRating := profile.Rating*float64(profile.ReviewsCount) + ratingDelta*float64(reviewsDelta)
	profile.ReviewsCount += reviewsDelta
	if profile.ReviewsCount > 0 {
		profile.Rating = totalRating / float64(profile.ReviewsCount)
	}

	profile.TrustScore = profile.CalculateTrustScore()

	schemaProfile, err := domain.MapProfileToSchema(profile)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateProfile(ctx, schemaProfile); err != nil {
		s.log.Error("failed to update review stats", "user_id", userID, "error", err)
		return err
	}

	s.log.Info("updated review stats", "user_id", userID, "rating", profile.Rating, "reviews", profile.ReviewsCount, "trust_score", profile.TrustScore)
	return nil
}

// SetVerificationStatus updates verification fields and trust score.
func (s *ProfileServiceImpl) SetVerificationStatus(ctx context.Context, userID string, level string, verified bool, verificationDate *time.Time) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}

	profile.IDVerified = verified
	if verified {
		if verificationDate != nil {
			profile.VerificationDate = verificationDate
		} else {
			now := time.Now()
			profile.VerificationDate = &now
		}
	} else {
		profile.VerificationDate = nil
	}

	if level != "" {
		profile.VerificationLevel = level
	} else {
		switch {
		case profile.IDVerified && profile.PhoneVerified:
			profile.VerificationLevel = "trusted"
		case profile.IDVerified:
			profile.VerificationLevel = "identity"
		case profile.PhoneVerified:
			profile.VerificationLevel = "basic"
		default:
			profile.VerificationLevel = ""
		}
	}

	profile.TrustScore = profile.CalculateTrustScore()

	schemaProfile, err := domain.MapProfileToSchema(profile)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateProfile(ctx, schemaProfile); err != nil {
		s.log.Error("failed to set verification status", "user_id", userID, "error", err)
		return err
	}

	s.log.Info("updated verification", "user_id", userID, "level", profile.VerificationLevel, "verified", verified, "trust_score", profile.TrustScore)
	return nil
}
