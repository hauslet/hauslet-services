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

// PublishListingRequest handles the lifecycle transition of a listing to 'Under Review'.
func (s *ServiceImpl) PublishListingRequest(ctx context.Context, listingID uuid.UUID) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	s.log.Info("starting publish request for listing", "listing_id", listingID)

	// 1. Fetch Data
	listing, err := s.ensureListing(ctx, listingID, true)
	if err != nil {
		s.log.Error("failed to fetch listing", "listing_id", listingID, "error", err)
		return err
	}

	property, err := s.ensureProperty(ctx, listing.PropertyID)
	if err != nil {
		s.log.Error("failed to fetch property for listing", "property_id", listing.PropertyID, "listing_id", listingID, "error", err)
		return err
	}

	// 2. Validate State & Permissions
	if err := s.validatePublishEligibility(ctx, listing); err != nil {
		return err
	}

	// 3. Notify Owner (Best Effort)
	s.notifyOwnerOfPublishRequest(ctx, listing)

	// 4. Prepare Moderation Data
	payloadJSON, err := s.buildModerationPayload(listing, property)
	if err != nil {
		s.log.Error("failed to serialize listing payload", "listing_id", listingID, "error", err)
		return fmt.Errorf("failed to serialize listing payload: %w", err)
	}

	// 5. Enqueue External Moderation Tasks
	if s.moderationHooks == nil {
		s.log.Error("moderation hooks not configured", "listing_id", listingID)
		return fmt.Errorf("moderation hooks not configured")
	}

	if err := s.enqueueModerationTasks(ctx, listing, payloadJSON); err != nil {
		return err
	}

	// 6. Update Database Status
	if err := s.updateListingStatus(ctx, listing, domain.StatusUnderReview, domain.ReviewPending); err != nil {
		return err
	}

	// 7. Cache Invalidation
	publicID := s.getPropertyPublicID(ctx, listing.PropertyID)
	s.invalidateListingCache(ctx, listing.ID, listing.Slug, publicID)

	s.log.Info("successfully submitted listing for moderation", "listing_id", listingID, "status", domain.StatusUnderReview)
	return nil
}

// UnpublishListing retracts an active listing.
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
		s.log.Warn("listing not currently published", "listing_id", listingID, "status", existing.Status)
		return nil, fmt.Errorf("listing is not currently published")
	}

	if err := s.updateListingStatus(ctx, existing, domain.StatusInactive, domain.ReviewPending); err != nil {
		return nil, err
	}

	updated, err := s.ensureListing(ctx, listingID, false)
	if err != nil {
		s.log.Error("failed to fetch listing after unpublish", "listing_id", listingID, "error", err)
		return nil, err
	}

	s.log.Info("listing unpublished", "listing_id", listingID, "status", updated.Status)
	return updated, nil
}

// ---------------------------------------------------------------------
// Helper Methods (Extracted Logic)
// ---------------------------------------------------------------------

func (s *ServiceImpl) validatePublishEligibility(ctx context.Context, listing *domain.Listing) error {
	// Authorization
	if err := s.authorizeSupplyAction(ctx, authorization.SupplyActionPublishListing,
		&authorization.SupplyOptions{ListingType: string(listing.ListingType)}); err != nil {
		return err
	}

	// Status Checks
	switch listing.Status {
	case domain.StatusActive:
		return fmt.Errorf("listing is already published and active")
	case domain.StatusUnderReview:
		return fmt.Errorf("listing is currently under review")
	case domain.StatusDraft, domain.StatusRequiresUpdates, domain.StatusInactive:
		// Allowed states
	default:
		s.log.Warn("listing cannot publish from current state", "listing_id", listing.ID, "status", listing.Status)
		return fmt.Errorf("cannot publish listing from current state: %s", listing.Status)
	}

	// Business Permissions
	if listing.OwnerType == domain.OwnerBusiness {
		if s.businessAuthorizer == nil {
			return domain.ErrForbidden
		}
		if err := s.businessAuthorizer.CanPublishListing(ctx, listing.OwnerID); err != nil {
			return domain.ErrForbidden
		}
	}

	// Completeness Check
	completeness, err := s.GetListingCompleteness(ctx, listing.ID, listing.OwnerID)
	if err != nil {
		s.log.Error("failed to check completeness for listing", "listing_id", listing.ID, "error", err)
		return err
	}
	if !completeness.ReadyToPublish {
		s.log.Warn("listing not ready to publish", "listing_id", listing.ID, "completeness_score", completeness.CompletionScore)
		return fmt.Errorf("listing is not ready to be published, completeness: %v%%", completeness.CompletionScore)
	}

	return nil
}

func (s *ServiceImpl) buildModerationPayload(listing *domain.Listing, property *domain.Property) (string, error) {
	payload := map[string]any{
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
		payload["rental_terms"] = listing.RentalDetails.RentalTerms
		for i, rule := range listing.RentalDetails.RentalRules {
			if rule.Category == domain.RuleCustom {
				payload[fmt.Sprintf("rental_rule_custom_%d", i)] = rule.Rules
			}
		}
		for i, fee := range listing.RentalDetails.Fees {
			payload[fmt.Sprintf("rental_fee_name_%d", i)] = fee.Name
			payload[fmt.Sprintf("rental_fee_category_%d", i)] = fee.Category
		}
		for i, discount := range listing.RentalDetails.Discounts {
			payload[fmt.Sprintf("rental_discount_name_%d", i)] = discount.Name
		}

	case domain.ListingSale:
		payload["sale_ownership_title"] = listing.SaleDetails.OwnershipTitle
		payload["sale_terms"] = listing.SaleDetails.SaleTerms
		for i, fee := range listing.SaleDetails.Fees {
			payload[fmt.Sprintf("sale_fee_name_%d", i)] = fee.Name
			payload[fmt.Sprintf("sale_fee_category_%d", i)] = fee.Category
		}
		for i, discount := range listing.SaleDetails.Discounts {
			payload[fmt.Sprintf("sale_discount_name_%d", i)] = discount.Name
		}

	case domain.ListingShortLet:
		for i, rule := range listing.ShortletDetails.Rules {
			if rule.Category == domain.RuleCustom {
				payload[fmt.Sprintf("shortlet_rule_custom_%d", i)] = rule.Rules
			}
		}
		for i, fee := range listing.ShortletDetails.Fees {
			payload[fmt.Sprintf("shortlet_fee_name_%d", i)] = fee.Name
			payload[fmt.Sprintf("shortlet_fee_category_%d", i)] = fee.Category
		}
		for i, discount := range listing.ShortletDetails.Discounts {
			payload[fmt.Sprintf("shortlet_discount_name_%d", i)] = discount.Name
		}
	default:
		return "", fmt.Errorf("unknown listing type: %s", listing.ListingType)
	}

	for i, media := range listing.Media {
		payload[fmt.Sprintf("media_key_%d", i)] = media.Key
		payload[fmt.Sprintf("media_caption_%d", i)] = media.Caption
		payload[fmt.Sprintf("media_group_%d", i)] = media.Group
	}

	return serializeToJSON(payload)
}

func (s *ServiceImpl) enqueueModerationTasks(ctx context.Context, listing *domain.Listing, payloadJSON string) error {
	s.log.Info("enqueueing text moderation for listing", "listing_id", listing.ID)
	if err := s.moderationHooks.EnqueueAIModeration(ctx, listing.ID, "listing_text", payloadJSON); err != nil {
		s.log.Error("failed to enqueue text moderation", "listing_id", listing.ID, "error", err)
		return fmt.Errorf("failed to enqueue text moderation: %w", err)
	}

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
			s.log.Error("failed to enqueue moderation for media", "media_id", media.ID, "listing_id", listing.ID, "error", err)
			return fmt.Errorf("failed to enqueue moderation for media %s: %w", media.ID, err)
		}
		mediaCount++
	}
	s.log.Info("enqueued moderation for media items", "count", mediaCount, "listing_id", listing.ID)
	return nil
}

func (s *ServiceImpl) updateListingStatus(ctx context.Context, listing *domain.Listing, status domain.ListingStatus, reviewStatus domain.ReviewStatus) error {
	updates := map[string]any{
		"status":               status,
		"latest_review_status": reviewStatus,
		"published":            false,
		"published_at":         nil,
		"status_changed_at":    time.Now(),
	}

	if err := s.repo.PatchListing(ctx, listing.ID, updates); err != nil {
		s.log.Error("failed to update listing status", "listing_id", listing.ID, "target_status", status, "error", err)
		return fmt.Errorf("failed to update listing status: %w", err)
	}
	return nil
}

func (s *ServiceImpl) notifyOwnerOfPublishRequest(ctx context.Context, listing *domain.Listing) {
	contactName, contactEmail := s.resolveOwnerContact(ctx, listing.OwnerType, listing.OwnerID)
	if s.notificationService != nil && contactEmail != "" {
		if err := s.notificationService.SendPublishListingRequestNotification(ctx, listing.Title, contactName, contactEmail); err != nil {
			s.log.Error("failed to send publish listing request notification", "listing_id", listing.ID, "error", err)
		}
	}
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
	defaultName := "User"
	defaultEmail := ""

	// 1. Individual Owner
	if ownerType != domain.OwnerBusiness {
		if s.profiles == nil {
			return defaultName, defaultEmail
		}
		name, email, err := s.profiles.GetProfileData(ctx, ownerID.String())
		if err != nil {
			s.log.Warn("failed to fetch profile data for owner", "owner_id", ownerID, "error", err)
			return defaultName, defaultEmail
		}
		if name == "" {
			name = defaultName
		}
		return name, email
	}

	// 2. Business Owner
	if s.businessService == nil {
		s.log.Warn("business service not configured; cannot resolve contact", "business_id", ownerID)
		return defaultName, defaultEmail
	}

	members, err := s.businessService.GetBusinessMembers(ctx, ownerID)
	if err != nil {
		s.log.Warn("failed to fetch business members", "business_id", ownerID, "error", err)
		return defaultName, defaultEmail
	}

	// 3. Find Eligible Business Member
	for _, m := range members {
		if !m.IsActive {
			continue
		}
		// Check permissions
		if !(m.IsOwner() || m.IsAdmin() || m.Permissions.CanPublishListings) {
			continue
		}
		if s.profiles == nil {
			continue
		}

		name, email, err := s.profiles.GetProfileData(ctx, m.UserID.String())
		if err != nil || email == "" {
			continue
		}
		if name == "" {
			name = defaultName
		}
		return name, email
	}

	s.log.Warn("no eligible business contact found", "business_id", ownerID)
	return defaultName, defaultEmail
}
