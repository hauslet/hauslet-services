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
	"log/slog"
	"maps"
	"strings"
	"time"

	"hauslet/internal/platform/redis"

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
	log         *slog.Logger
	embedding   *aiembeddings.Client
	businessSvc businessservice.BusinessService
}

// NewModerationPropertyAdapter constructs the adapter with its dependencies.
func NewModerationPropertyAdapter(repo repository.Repository, profiles ProfileProvider,
	notifier *notification.NotificationService, embedding *aiembeddings.Client, cache redis.RedisClient, log *slog.Logger,
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
		a.log.Warn("listing not found during moderation completion", "listing_id", aggregate.TargetID.String())
		return domain.ErrListingNotFound
	}

	property, err := a.repo.GetPropertyByID(ctx, listing.PropertyID)
	if err != nil {
		a.log.Warn("failed to fetch property for listing during moderation completion", "property_id", listing.PropertyID.String(), "listing_id", listing.ID.String(), "error", err)
		return domain.ErrPropertyNotFound
	}
	if property == nil {
		a.log.Warn("property not found for listing during moderation completion", "property_id", listing.PropertyID.String(), "listing_id", listing.ID.String())
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
				a.log.Error("failed to send listing accepted notification", "email", ownerProfileEmail, "error", err)
			}
		}

	case domain.ModerationStatusRejected:
		// Notify owner of rejection with reasons.
		if ownerProfileEmail != "" && a.notifier != nil {
			err := a.notifier.SendListingRejectedNotification(ctx,
				listing.Title, ownerProfileName, ownerProfileEmail, aggDomain.Reasons)
			if err != nil {
				a.log.Error("failed to send listing rejected notification", "email", ownerProfileEmail, "error", err)
			}
		}
	case domain.ModerationStatusEscalated:
		// Escalation would be handled in moderation service directly
		// Silent on the user side, log only.
		a.log.Info("listing escalated for human review", "listing_id", listing.ID)
	default:
		a.log.Warn("listing reached unknown moderation status", "listing_id", listing.ID, "status", moderationStatus)
	}

	updates := a.buildListingUpdates(aggDomain)
	if aggDomain.FinalStatus() == domain.ModerationStatusAccepted && a.embedding != nil {
		if embUpdates := a.generateEmbeddingUpdates(ctx, listing, property); len(embUpdates) > 0 {
			if updates == nil {
				updates = make(map[string]any, len(embUpdates))
			}
			maps.Copy(updates, embUpdates)
		}
		a.log.Info("updated listing with new embedding after acceptance", "listing_id", listing.ID)
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
		a.log.Error("failed to generate embedding for listing", "listing_id", listing.ID, "error", err)
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
			a.log.Warn("failed to fetch profile data for owner", "owner_id", listing.OwnerID, "error", err)
		}
		if name == "" {
			name = "User"
		}
		return name, email
	}

	// Business owner path: notify an owner/admin with publish permission.
	if a.businessSvc == nil {
		a.log.Warn("business service not configured; cannot resolve contact for business", "business_id", listing.OwnerID)
		return "User", ""
	}

	members, err := a.businessSvc.GetBusinessMembers(ctx, listing.OwnerID)
	if err != nil {
		a.log.Warn("failed to fetch business members for business", "business_id", listing.OwnerID, "error", err)
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

	a.log.Warn("no eligible business contact found for business", "business_id", listing.OwnerID)
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
			a.log.Warn("cache invalidation failed for listing", "listing_id", id, "error", err)
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
