package service

import (
	"context"

	"hauslet/internal/modules/business/notification"
	"hauslet/internal/modules/business/repository"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// BusinessServiceImpl implements BusinessService
type BusinessServiceImpl struct {
	repo     repository.BusinessRepository
	log      *lgr.Logger
	notifier *notification.NotificationService
	profile  ProfileProvider
}

// ProfileProvider exposes just the bits of profile data the business module needs.
type ProfileProvider interface {
	GetProfileName(ctx context.Context, userID string) (string, error)
}

// NewBusinessService creates a new business service
func NewBusinessService(repo repository.BusinessRepository, notifier *notification.NotificationService, profile ProfileProvider, log *lgr.Logger) BusinessService {
	return &BusinessServiceImpl{
		repo:     repo,
		notifier: notifier,
		profile:  profile,
		log:      log,
	}
}

// getProfileName fetches the user's display name if a profile provider is wired.
func (s *BusinessServiceImpl) getProfileName(ctx context.Context, userID uuid.UUID) string {
	if s.profile == nil {
		return ""
	}
	name, err := s.profile.GetProfileName(ctx, userID.String())
	if err != nil {
		if s.log != nil {
			s.log.Logf("WARN Failed to fetch profile name for user %s: %v", userID, err)
		}
		return ""
	}
	return name
}
