package service

import (
	"context"
	"hauslet/internal/modules/profile/domain"
	"time"
)

// CreateDefaultProfile satisfies the Auth module's interface
func (s *ProfileServiceImpl) CreateDefaultProfile(ctx context.Context, userID string, name string, birthDate *time.Time) error {
	s.log.Logf("[INFO] creating default profile for user %s (name: %s)", userID, name)

	_, err := s.CreateProfile(ctx, domain.Profile{
		UserID:              userID,
		FullName:            name,
		BirthDate:           birthDate,
		CommunityCommitment: false,
	})
	if err != nil {
		s.log.Logf("[ERROR] failed to create default profile for user %s: %v", userID, err)
	}
	return err
}
