package service

import (
	"context"

	"hauslet/internal/modules/profile/domain"
)

// AddBadge adds a badge to a profile, avoiding duplicates and recalculating trust.
func (s *ProfileServiceImpl) AddBadge(ctx context.Context, userID string, badge domain.Badge) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}
	if !domain.IsValidBadge(badge) {
		return domain.ErrInvalidBadge
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}

	if profile.HasBadge(badge) {
		return nil
	}

	profile.Badges = append(profile.Badges, badge)
	profile.TrustScore = profile.CalculateTrustScore()

	schemaProfile, err := domain.MapProfileToSchema(profile)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateProfile(ctx, schemaProfile); err != nil {
		s.log.Logf("[ERROR] failed to add badge %s for user %s: %v", badge, userID, err)
		return err
	}

	s.log.Logf("[INFO] added badge %s for user %s (new trust score: %.2f)", badge, userID, profile.TrustScore)
	return nil
}

// RemoveBadge removes a badge and recalculates trust score.
func (s *ProfileServiceImpl) RemoveBadge(ctx context.Context, userID string, badge domain.Badge) error {
	if userID == "" {
		return domain.ErrInvalidUserID
	}

	profile, err := s.ensureProfile(ctx, userID)
	if err != nil {
		return err
	}

	if !profile.HasBadge(badge) {
		return nil
	}

	filtered := make([]domain.Badge, 0, len(profile.Badges))
	for _, b := range profile.Badges {
		if b != badge {
			filtered = append(filtered, b)
		}
	}
	profile.Badges = filtered
	profile.TrustScore = profile.CalculateTrustScore()

	schemaProfile, err := domain.MapProfileToSchema(profile)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateProfile(ctx, schemaProfile); err != nil {
		s.log.Logf("[ERROR] failed to remove badge %s for user %s: %v", badge, userID, err)
		return err
	}

	s.log.Logf("[INFO] removed badge %s for user %s (new trust score: %.2f)", badge, userID, profile.TrustScore)
	return nil
}
