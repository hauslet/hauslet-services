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
		s.log.Error("failed to add badge", "badge", badge, "user_id", userID, "error", err)
		return err
	}

	s.log.Info("added badge", "badge", badge, "user_id", userID, "new_trust_score", profile.TrustScore)
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
		s.log.Error("failed to remove badge", "badge", badge, "user_id", userID, "error", err)
		return err
	}

	s.log.Info("removed badge", "badge", badge, "user_id", userID, "new_trust_score", profile.TrustScore)
	return nil
}
