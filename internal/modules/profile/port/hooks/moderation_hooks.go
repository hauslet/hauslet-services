package hooks

import (
	"context"
	"fmt"
	moderationservice "hauslet/internal/modules/moderation/service"
	"hauslet/internal/modules/profile/domain"
	"hauslet/internal/modules/profile/notification"
	"hauslet/internal/modules/profile/repository"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// NotificationSender abstracts sending moderation outcome notifications.
type ProfileProvider interface {
	GetProfileData(ctx context.Context, userID string) (string, string, error)
}

// ModerationProfileAdapter applies moderation outcomes to user profiles.
type ModerationProfileAdapter struct {
	repo repository.ProfileRepository

	notifier *notification.NotificationService
	nowFunc  func() time.Time
	log      *lgr.Logger
}

// NewModerationProfileAdapter constructs the adapter with its dependencies.
func NewModerationProfileAdapter(repo repository.ProfileRepository,
	notifier *notification.NotificationService, log *lgr.Logger) *ModerationProfileAdapter {
	return &ModerationProfileAdapter{
		repo:     repo,
		notifier: notifier,
		nowFunc:  time.Now,
		log:      log,
	}
}

// OnModerationCompleted updates profile status/review fields based
func (a *ModerationProfileAdapter) OnModerationCompleted(ctx context.Context,
	aggregate moderationservice.AggregatedModeration) error {
	if aggregate.TargetID == uuid.Nil {
		return fmt.Errorf("profile ID is required for moderation completion")
	}

	profile, err := a.repo.GetProfileByID(ctx, aggregate.TargetID.String())
	if err != nil {
		return err
	}
	if profile == nil {
		a.log.Logf("[WARN] profile not found for ID %s", aggregate.TargetID.String())
		return domain.ErrProfileNotFound

	}
	aggDomain := domain.MapModerationAggToDomain(aggregate)
	if aggDomain.FinalStatus() == domain.ModerationStatusAccepted {
		// Profile approved
		a.log.Logf("[INFO] profile %s approved by moderation", profile.UserID)
		a.repo.UpdateModerationStatus(ctx, profile.ID, true)
		return nil
	}

	switch aggDomain.FinalStatus() {
	case domain.ModerationStatusRejected:
		// Profile rejected
		a.log.Logf("[INFO] profile %s rejected by moderation", profile.UserID)
		a.repo.UpdateModerationStatus(ctx, profile.ID, false)

		// Notify profile owner of rejection
		if profile.Email != nil && a.notifier != nil {
			a.log.Logf("[INFO] sending profile moderation rejection email to user %s", profile.UserID)
			err := a.notifier.SendProfileModerationRejectedEmail(ctx, *profile.Email, profile.FullName, profile.ID.String(), aggDomain.Reasons)
			if err != nil {
				a.log.Logf("[ERROR] failed to send profile moderation rejection email to user %s: %v", profile.UserID, err)
			}
		}

	case domain.ModerationStatusAccepted:
		// Update profile as approved only
		a.log.Logf("[INFO] profile %s approved by moderation", profile.UserID)
		a.repo.UpdateModerationStatus(ctx, profile.ID, true)
		return nil
	default:
		a.log.Logf("[WARN] unhandled moderation status for profile %s: %s", profile.UserID, aggDomain.FinalStatus())
	}

	return nil
}
