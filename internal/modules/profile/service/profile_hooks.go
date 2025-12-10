package service

import (
	"context"
	"hauslet/internal/modules/profile/domain"
	"time"
)

// CreateDefaultProfile satisfies the Auth module's interface
func (s *ProfileServiceImpl) CreateDefaultProfile(ctx context.Context, userID string, name string, birthDate *time.Time) error {

	_, err := s.CreateProfile(ctx, domain.Profile{
		UserID:              userID,
		FullName:            name,
		BirthDate:           birthDate,
		CommunityCommitment: false,
	})
	return err
}
