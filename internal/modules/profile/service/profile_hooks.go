package service

import (
	"context"
	"hauslet/internal/modules/profile/domain"
	"time"
)

// CreateDefaultProfile satisfies the Auth module's interface
func (s *ProfileServiceImpl) CreateDefaultProfile(ctx context.Context, userID string, name string, birthDate *time.Time) error {
	s.log.Info("creating default profile for user", "user_id", userID, "name", name)

	_, err := s.CreateProfile(ctx, domain.Profile{
		UserID:              userID,
		FullName:            name,
		BirthDate:           birthDate,
		CommunityCommitment: false,
	})
	if err != nil {
		s.log.Error("failed to create default profile", "user_id", userID, "error", err)
	}
	return err
}
