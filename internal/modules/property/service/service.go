package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/modules/auth/authorization"
	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

func (s *ServiceImpl) PublishListingRequest(ctx context.Context, listingID uuid.UUID) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	s.log.Info("starting publish request for listing", "listing_id", listingID)

	listing, err := s.ensureListing(ctx, listingID, true)
	if err != nil {
		s.log.Error("failed to fetch listing", "listing_id", listingID, "error", err)
		return err
	}

	if err := s.authorizeSupplyAction(ctx, authorization.SupplyActionPublishListing,
		&authorization.SupplyOptions{
			ListingType: string(listing.ListingType),
		}); err != nil {
		return err
	}

	if listing.Status == domain.StatusActive {
		return fmt.Errorf("listing is already published and active")
	}
	if listing.Status == domain.StatusUnderReview {
		return fmt.Errorf("listing is currently under review")
	}

	// Allow submission from Draft, Inactive, or after a Rejection (RequiresUpdates)
	canPublish := listing.Status == domain.StatusDraft ||
		listing.Status == domain.StatusRequiresUpdates ||
		listing.Status == domain.StatusInactive

	if !canPublish {
		s.log.Warn("listing cannot publish from current state", "listing_id", listingID, "status", listing.Status)
		return fmt.Errorf("cannot publish listing from current state: %s", listing.Status)
	}

	// Enforce business publish permissions
	if listing.OwnerType == domain.OwnerBusiness {
		if s.businessAuthorizer == nil {
			return domain.ErrForbidden
		}
		if err := s.businessAuthorizer.CanPublishListing(ctx, listing.OwnerID); err != nil {
			return domain.ErrForbidden
		}
	}

	// check completeness
	completeness, err := s.GetListingCompleteness(ctx, listingID, listing.OwnerID)
	if err != nil {
		s.log.Error("failed to check completeness for listing", "listing_id", listingID, "error", err)
		return err
	}
	if !completeness.ReadyToPublish {
		s.log.Warn("listing not ready to publish", "listing_id", listingID, "completeness_score", completeness.CompletionScore)
		return fmt.Errorf("listing is not ready to be published, completeness: %v%%", completeness.CompletionScore)
	}

	property, err := s.ensureProperty(ctx, listing.PropertyID)
	if err != nil {
		s.log.Error("failed to fetch property for listing", "property_id", listing.PropertyID, "listing_id", listingID, "error", err)
		return err
	}

	contactName, contactEmail := s.resolveOwnerContact(ctx, listing.OwnerType, listing.OwnerID)

	// Best-effort notify immediately that moderation has been requested.
	if s.notificationService != nil && contactEmail != "" {
		if err := s.notificationService.SendPublishListingRequestNotification(ctx, listing.Title, contactName, contactEmail); err != nil {
			s.log.Error("failed to send publish listing request notification", "listing_id", listingID, "error", err)
		}
	}

	listingPayload := map[string]any{
		"title":               listing.Title,
		"description":         listing.Description,
		"extra_description":   listing.ExtraDescription,
		"currency":            listing.Currency,
		"listing_type":        listing.ListingType,
		"address":             property.Address,
		"city":                property.City,
		"state":               property.State,
		"country":             property.Country,
		"features_commercial": property.FeaturesCommercial,
	}

	switch listing.ListingType {
	case domain.ListingRent:
		listingPayload["rental_terms"] = listing.RentalDetails.RentalTerms
		if len(listing.RentalDetails.RentalRules) > 0 {
			for i, rule := range listing.RentalDetails.RentalRules {
				if rule.Category == domain.RuleCustom {
					listingPayload[fmt.Sprintf("rental_rule_custom_%d", i)] = rule.Rules
				}
			}
		}

	case domain.ListingSale:
		listingPayload["sale_ownership_title"] = listing.SaleDetails.OwnershipTitle
		listingPayload["sale_terms"] = listing.SaleDetails.SaleTerms
		scb := listing.SaleDetails.ServiceChargeBreakdown
		if scb != nil && len(*scb) > 0 {
			for i, charge := range *scb {
				listingPayload[fmt.Sprintf("service_charge_name_%d", i)] = charge.Name
			}
		}
	case domain.ListingShortLet:
		if len(listing.ShortletDetails.Rules) > 0 {
			for i, rule := range listing.ShortletDetails.Rules {
				if rule.Category == domain.RuleCustom {
					listingPayload[fmt.Sprintf("shortlet_rule_custom_%d", i)] = rule.Rules
				}
			}
		}

	default:
		return fmt.Errorf("unknown listing type: %s", listing.ListingType)

	}

	for i, media := range listing.Media {
		listingPayload[fmt.Sprintf("media_key_%d", i)] = media.Key
		listingPayload[fmt.Sprintf("media_caption_%d", i)] = media.Caption
		listingPayload[fmt.Sprintf("media_group_%d", i)] = media.Group
	}

	listingPayloadStr, err := serializeToJSON(listingPayload)
	if err != nil {
		s.log.Error("failed to serialize listing payload", "listing_id", listingID, "error", err)
		return fmt.Errorf("failed to serialize listing payload: %w", err)
	}

	// Enqueue moderation
	if s.moderationHooks == nil {
		s.log.Error("moderation hooks not configured", "listing_id", listingID)
		return fmt.Errorf("moderation hooks not configured")
	}

	// Enqueue listing text moderation
	s.log.Info("enqueueing text moderation for listing", "listing_id", listingID)
	if err := s.moderationHooks.EnqueueAIModeration(ctx, listing.ID, "listing_text", listingPayloadStr); err != nil {
		s.log.Error("failed to enqueue text moderation", "listing_id", listingID, "error", err)
		return fmt.Errorf("failed to enqueue text moderation: %w", err)
	}

	// Fan-out media moderation per asset to align with single-payload AI contract.
	mediaCount := 0
	for _, media := range listing.Media {
		if media.Key == "" {
			continue
		}

		var contentType string
		switch media.Type {
		case domain.MediaTypeImage:
			contentType = "listing_image"
		case domain.MediaTypeVideo:
			contentType = "listing_video"
		default:
			continue
		}

		if err := s.moderationHooks.EnqueueAIModeration(ctx, listing.ID, contentType, media.Key); err != nil {
			s.log.Error("failed to enqueue moderation for media", "media_id", media.ID, "listing_id", listingID, "error", err)
			return fmt.Errorf("failed to enqueue moderation for media %s: %w", media.ID, err)
		}
		mediaCount++
	}
	s.log.Info("enqueued moderation for media items", "count", mediaCount, "listing_id", listingID)

	statusChangedAt := time.Now()
	updates := map[string]any{
		"status":               domain.StatusUnderReview,
		"latest_review_status": domain.ReviewPending,
		"published":            false,
		"published_at":         nil,
		"status_changed_at":    statusChangedAt,
	}

	if err := s.repo.PatchListing(ctx, listing.ID, updates); err != nil {
		s.log.Error("failed to update listing status", "listing_id", listingID, "error", err)
		return fmt.Errorf("failed to update listing status: %w", err)
	}

	// Invalidate cache to ensure subsequent reads get the updated status
	publicID := s.getPropertyPublicID(ctx, listing.PropertyID)
	s.invalidateListingCache(ctx, listing.ID, listing.Slug, publicID)

	// No additional notification here; it was sent best-effort before enqueue.

	s.log.Info("successfully submitted listing for moderation", "listing_id", listingID, "status", domain.StatusUnderReview)
	return nil
}

// UnpublishListing retracts an active listing and, if ready, re-queues it for moderation.
func (s *ServiceImpl) UnpublishListing(ctx context.Context, listingID uuid.UUID) (*domain.Listing, error) {
	if listingID == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	s.log.Info("starting unpublish for listing", "listing_id", listingID)

	existing, err := s.ensureListing(ctx, listingID, false)
	if err != nil {
		s.log.Error("failed to fetch listing for unpublish", "listing_id", listingID, "error", err)
		return nil, err
	}

	if !existing.Published || existing.Status != domain.StatusActive {
		s.log.Warn("listing not currently published", "listing_id", listingID, "status", existing.Status, "published", existing.Published)
		return nil, fmt.Errorf("listing is not currently published")
	}

	updated, err := s.ensureListing(ctx, listingID, false)
	if err != nil {
		s.log.Error("failed to fetch listing after unpublish", "listing_id", listingID, "error", err)
		return nil, err
	}

	s.log.Info("listing unpublished", "listing_id", listingID, "status", updated.Status)
	return updated, nil
}

// ensureProperty fetches a property by ID, returning an error if not found.
func (s *ServiceImpl) ensureProperty(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	p, err := s.repo.GetPropertyByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrPropertyNotFound
	}

	return domain.MapPropertyFromSchema(p), nil
}

// ensureListing fetches a listing by ID, returning an error if not found.
func (s *ServiceImpl) ensureListing(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error) {
	l, err := s.repo.GetListingByID(ctx, id, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	return domain.MapListingFromSchema(l), nil
}

func serializeToJSON(data any) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// resolveOwnerContact determines the contact name/email for a listing owner, handling business owners.
func (s *ServiceImpl) resolveOwnerContact(ctx context.Context, ownerType domain.OwnerType, ownerID uuid.UUID) (string, string) {
	name := "User"
	email := ""

	if ownerType != domain.OwnerBusiness {
		if s.profiles != nil {
			if n, e, err := s.profiles.GetProfileData(ctx, ownerID.String()); err == nil {
				if n != "" {
					name = n
				}
				email = e
			} else {
				s.log.Warn("failed to fetch profile data for owner", "owner_id", ownerID, "error", err)
			}
		}
		return name, email
	}

	// Business owner path: find a member to notify.
	if s.businessService == nil {
		s.log.Warn("business service not configured; cannot resolve contact", "business_id", ownerID)
		return name, email
	}

	members, err := s.businessService.GetBusinessMembers(ctx, ownerID)
	if err != nil {
		s.log.Warn("failed to fetch business members", "business_id", ownerID, "error", err)
		return name, email
	}

	for _, m := range members {
		if !m.IsActive {
			continue
		}
		if !(m.IsOwner() || m.IsAdmin() || m.Permissions.CanPublishListings) {
			continue
		}
		if s.profiles == nil {
			continue
		}
		n, e, err := s.profiles.GetProfileData(ctx, m.UserID.String())
		if err != nil || e == "" {
			continue
		}
		if n != "" {
			name = n
		}
		email = e
		return name, email
	}

	s.log.Warn("no eligible business contact found", "business_id", ownerID)
	return name, email
}
