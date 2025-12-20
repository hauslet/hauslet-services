package hooks

import (
	"context"
	"fmt"
	moderationservice "hauslet/internal/modules/moderation/service"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/notification"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"
	aiembeddings "hauslet/internal/platform/ai/embeddings"
	"maps"
	"strings"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// Ensure compile-time conformance with the moderation service hooks.
var _ moderationservice.PropertyHooks = (*ModerationPropertyAdapter)(nil)

// NotificationSender abstracts sending moderation outcome notifications.
type ProfileProvider interface {
	GetProfileData(ctx context.Context, userID string) (string, string, error)
}

// ModerationPropertyAdapter applies moderation outcomes back to listings and notifies owners.
type ModerationPropertyAdapter struct {
	repo      repository.Repository
	profiles  ProfileProvider
	notifier  *notification.NotificationService
	nowFunc   func() time.Time
	log       *lgr.Logger
	embedding *aiembeddings.Client
}

// NewModerationPropertyAdapter constructs the adapter with its dependencies.
func NewModerationPropertyAdapter(repo repository.Repository, profiles ProfileProvider,
	notifier *notification.NotificationService, embedding *aiembeddings.Client, log *lgr.Logger) *ModerationPropertyAdapter {
	return &ModerationPropertyAdapter{
		repo:      repo,
		profiles:  profiles,
		notifier:  notifier,
		nowFunc:   time.Now,
		embedding: embedding,
		log:       log,
	}
}

// OnModerationCompleted updates listing status/review fields based
// on the aggregate moderation outcome and notifies the owner.
func (a *ModerationPropertyAdapter) OnModerationCompleted(ctx context.Context,
	aggregate moderationservice.AggregatedModeration) error {
	if aggregate.TargetID == uuid.Nil {
		return fmt.Errorf("listing ID is required for moderation completion")
	}

	// Fetch listing to ensure it exists and to identify the owner.
	listing, err := a.repo.GetListingByID(ctx, aggregate.TargetID, false)
	if err != nil {
		return err
	}
	if listing == nil {
		a.log.Logf("[WARN] listing %s not found during moderation completion", aggregate.TargetID.String())
		return domain.ErrListingNotFound
	}

	property, err := a.repo.GetPropertyByID(ctx, listing.PropertyID)
	if err != nil {
		a.log.Logf("[WARN] failed to fetch property %s for listing %s: %v", listing.PropertyID.String(), listing.ID.String(), err)
		return domain.ErrPropertyNotFound
	}
	if property == nil {
		a.log.Logf("[WARN] property %s not found for listing %s during moderation completion", listing.PropertyID.String(), listing.ID.String())
		return domain.ErrPropertyNotFound
	}

	ownerProfileName, ownerProfileEmail, err := a.profiles.GetProfileData(ctx, listing.OwnerID.String())
	if err != nil {
		a.log.Logf("[WARN] failed to fetch profile data for owner %s: %v", listing.OwnerID.String(), err)
	}
	if ownerProfileName == "" {
		ownerProfileName = "User"
	}

	aggDomain := domain.MapModerationAggToDomain(aggregate)

	moderationStatus := aggDomain.FinalStatus()
	switch moderationStatus {
	case domain.ModerationStatusAccepted:
		// Notify owner of acceptance.
		if ownerProfileEmail != "" && a.notifier != nil {
			err := a.notifier.SendListingAcceptedNotification(ctx, listing.Title, ownerProfileName, ownerProfileEmail)
			if err != nil {
				a.log.Logf("[ERROR] failed to send listing accepted notification to %s: %v", ownerProfileEmail, err)
			}
		}

	case domain.ModerationStatusRejected:
		// Notify owner of rejection with reasons.
		if ownerProfileEmail != "" && a.notifier != nil {
			err := a.notifier.SendListingRejectedNotification(ctx,
				listing.Title, ownerProfileName, ownerProfileEmail, aggDomain.Reasons)
			if err != nil {
				a.log.Logf("[ERROR] failed to send listing rejected notification to %s: %v", ownerProfileEmail, err)
			}
		}
	case domain.ModerationStatusEscalated:
		// Escalation would be handled in moderation service directly
		// Silent on the user side, log only.
		a.log.Logf("[INFO] listing %s escalated for human review", listing.ID)
	default:
		a.log.Logf("[WARN] listing %s reached unknown moderation status: %s", listing.ID, moderationStatus)
	}

	updates := a.buildListingUpdates(aggDomain)
	if aggDomain.FinalStatus() == domain.ModerationStatusAccepted && a.embedding != nil {
		if embUpdates := a.generateEmbeddingUpdates(ctx, listing, property); len(embUpdates) > 0 {
			if updates == nil {
				updates = make(map[string]any, len(embUpdates))
			}
			maps.Copy(updates, embUpdates)
		}
		a.log.Logf("[INFO] updated listing %s with new embedding after acceptance", listing.ID)
	}
	if updates == nil {
		return nil
	}

	if err := a.repo.PatchListing(ctx, aggregate.TargetID, updates); err != nil {
		return err
	}
	return nil
}

func (a *ModerationPropertyAdapter) generateEmbeddingUpdates(
	ctx context.Context,
	listing *schema.Listing,
	property *schema.Property,
) map[string]any {

	if listing == nil || a.embedding == nil {
		return nil
	}

	doc := domain.NewEmbeddingDocumentBuilder().
		WithListing(domain.MapListingFromSchema(listing)).
		WithProperty(domain.MapPropertyFromSchema(property)).
		Build()

	if doc.Text == "" {
		return nil
	}

	vec, err := a.embedding.Embed(ctx, doc.Text)
	if err != nil {
		a.log.Logf(
			"[ERROR] failed to generate embedding for listing %s: %v",
			listing.ID,
			err,
		)
		return nil
	}
	if len(vec) == 0 {
		return nil
	}

	now := a.nowFunc()

	return map[string]any{
		"text_embedding":         schema.NewVectorEmbedding(vec),
		"embedding_model":        a.embedding.GetModelName(),
		"embedding_version":      doc.Version,
		"embedding_generated_at": now,
	}
}

// buildListingUpdates maps moderation status to listing field updates.
func (a *ModerationPropertyAdapter) buildListingUpdates(aggregate *domain.AggregatedModeration) map[string]any {
	if aggregate == nil {
		return nil
	}

	now := a.nowFunc()

	switch aggregate.FinalStatus() {
	case domain.ModerationStatusAccepted:
		return map[string]any{
			"status":               domain.StatusActive,
			"latest_review_status": domain.ReviewApproved,
			"published":            true,
			"published_at":         now,
			"status_changed_at":    now,
			"change_reason":        nil,
		}

	case domain.ModerationStatusRejected:
		return map[string]any{
			"status":               domain.StatusRequiresUpdates,
			"latest_review_status": domain.ReviewRejected,
			"published":            false,
			"published_at":         nil,
			"status_changed_at":    now,
			"change_reason":        strings.Join(aggregate.Reasons, "; "),
		}

	case domain.ModerationStatusEscalated:
		return map[string]any{
			"status":               domain.StatusUnderReview,
			"latest_review_status": domain.ReviewInconclusive,
			"status_changed_at":    now,
		}

	default:
		return nil
	}
}
