package hooks

import (
	"context"
	"fmt"
	moderationservice "hauslet/internal/modules/moderation/service"
	"hauslet/internal/modules/profile/notification"
	"hauslet/internal/modules/profile/repository"
	"time"

	"github.com/go-pkgz/lgr"
)

// NotificationSender abstracts sending moderation outcome notifications.
type ProfileProvider interface {
	GetProfileData(ctx context.Context, userID string) (string, string, error)
}

// ModerationProfileAdapter applies moderation outcomes to user profiles.
type ModerationProfileAdapter struct {
	repo     repository.ProfileRepository
	profiles ProfileProvider
	notifier *notification.NotificationService
	nowFunc  func() time.Time
	log      *lgr.Logger
}

// NewModerationProfileAdapter constructs the adapter with its dependencies.
func NewModerationProfileAdapter(repo repository.ProfileRepository, profiles ProfileProvider,
	notifier *notification.NotificationService) *ModerationProfileAdapter {
	return &ModerationProfileAdapter{
		repo:     repo,
		profiles: profiles,
		notifier: notifier,
		nowFunc:  time.Now,
	}
}

func (a *ModerationProfileAdapter) OnModerationCompleted(ctx context.Context,
	aggregate moderationservice.AggregatedModeration) error {

	return fmt.Errorf("Not Implemented")
}
