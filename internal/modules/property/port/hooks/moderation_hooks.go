package hooks

import (
	"context"
	"fmt"
	businessservice "hauslet/internal/modules/business/service"
	moderationservice "hauslet/internal/modules/moderation/service"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/notification"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"
	aiembeddings "hauslet/internal/platform/ai/embeddings"
	"maps"
	"strings"
	"time"

	"hauslet/internal/platform/redis"

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
	repo        repository.Repository
	profiles    ProfileProvider
	notifier    *notification.NotificationService
	nowFunc     func() time.Time
	cache       redis.RedisClient
	log         *lgr.Logger
	embedding   *aiembeddings.Client
	businessSvc businessservice.BusinessService
}

// NewModerationPropertyAdapter constructs the adapter with its dependencies.
func NewModerationPropertyAdapter(repo repository.Repository, profiles ProfileProvider,
	notifier *notification.NotificationService, embedding *aiembeddings.Client, cache redis.RedisClient, log *lgr.Logger,
	businessSvc businessservice.BusinessService) *ModerationPropertyAdapter {
	return &ModerationPropertyAdapter{
		repo:        repo,
		profiles:    profiles,
		notifier:    notifier,
		nowFunc:     time.Now,
		embedding:   embedding,
		cache:       cache,
		log:         log,
		businessSvc: businessSvc,
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

	ownerProfileName, ownerProfileEmail := a.resolveOwnerContact(ctx, listing)

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

	// Invalidate cache to ensure subsequent reads get the updated status
	a.invalidateListingCache(ctx, listing.ID, listing.Slug, property.PublicID)

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

// resolveOwnerContact determines the display name and email for the listing owner, handling business owners.
func (a *ModerationPropertyAdapter) resolveOwnerContact(ctx context.Context, listing *schema.Listing) (string, string) {
	if listing == nil {
		return "User", ""
	}

	// Individual owner path
	if listing.OwnerType != schema.OwnerBusiness {
		name, email, err := a.profiles.GetProfileData(ctx, listing.OwnerID.String())
		if err != nil {
			a.log.Logf("[WARN] failed to fetch profile data for owner %s: %v", listing.OwnerID, err)
		}
		if name == "" {
			name = "User"
		}
		return name, email
	}

	// Business owner path: notify an owner/admin with publish permission.
	if a.businessSvc == nil {
		a.log.Logf("[WARN] business service not configured; cannot resolve contact for business %s", listing.OwnerID)
		return "User", ""
	}

	members, err := a.businessSvc.GetBusinessMembers(ctx, listing.OwnerID)
	if err != nil {
		a.log.Logf("[WARN] failed to fetch business members for %s: %v", listing.OwnerID, err)
		return "User", ""
	}

	for _, m := range members {
		if !m.IsActive {
			continue
		}
		if !(m.IsOwner() || m.IsAdmin() || m.Permissions.CanPublishListings) {
			continue
		}
		name, email, err := a.profiles.GetProfileData(ctx, m.UserID.String())
		if err != nil || email == "" {
			continue
		}
		if name == "" {
			name = "User"
		}
		return name, email
	}

	a.log.Logf("[WARN] no eligible business contact found for business %s", listing.OwnerID)
	return "User", ""
}

// Cache invalidation helpers
func (a *ModerationPropertyAdapter) invalidateListingCache(ctx context.Context, id uuid.UUID, slug string, publicID string) {
	if a.cache == nil {
		return
	}
	if id == uuid.Nil && slug == "" && publicID == "" {
		return
	}

	keys := make([]string, 0, 6)
	if id != uuid.Nil {
		keys = append(keys, listingIDCacheKey(id, true), listingIDCacheKey(id, false))
	}
	if slug != "" {
		keys = append(keys, listingSlugCacheKey(slug, true), listingSlugCacheKey(slug, false))
	}
	if publicID != "" {
		keys = append(keys, listingPublicIDCacheKey(publicID, true), listingPublicIDCacheKey(publicID, false))
	}

	if len(keys) > 0 {
		if err := a.cache.Del(ctx, keys...).Err(); err != nil {
			a.log.Logf("[WARN] cache invalidation failed for listing %s: %v", id, err)
		}
	}
}

func listingIDCacheKey(id uuid.UUID, preloadMedia bool) string {
	return fmt.Sprintf("listing:%s:media:%t", id.String(), preloadMedia)
}

func listingSlugCacheKey(slug string, preloadMedia bool) string {
	return fmt.Sprintf("listing:slug:%s:media:%t", slug, preloadMedia)
}

func listingPublicIDCacheKey(publicID string, preloadMedia bool) string {
	return fmt.Sprintf("listing:public:%s:media:%t", publicID, preloadMedia)
}
