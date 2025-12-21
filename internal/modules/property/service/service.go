package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

func (s *ServiceImpl) PublishListingRequest(ctx context.Context, listingID uuid.UUID) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	s.log.Logf("INFO starting publish request for listing=%s", listingID)

	listing, err := s.ensureListing(ctx, listingID, true)
	if err != nil {
		s.log.Logf("ERROR failed to fetch listing=%s: %v", listingID, err)
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
		s.log.Logf("WARN listing=%s is in status %s, cannot publish", listingID, listing.Status)
		return fmt.Errorf("cannot publish listing from current state: %s", listing.Status)
	}

	// check completeness
	completeness, err := s.GetListingCompleteness(ctx, listingID, listing.OwnerID)
	if err != nil {
		s.log.Logf("ERROR failed to check completeness for listing=%s: %v", listingID, err)
		return err
	}
	if !completeness.ReadyToPublish {
		s.log.Logf("WARN listing=%s not ready to publish, completeness=%d%%", listingID, completeness.CompletionScore)
		return fmt.Errorf("listing is not ready to be published, completeness: %v%%", completeness.CompletionScore)
	}

	property, err := s.ensureProperty(ctx, listing.PropertyID)
	if err != nil {
		s.log.Logf("ERROR failed to fetch property=%s for listing=%s: %v", listing.PropertyID, listingID, err)
		return err
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
		s.log.Logf("ERROR failed to serialize listing payload for listing=%s: %v", listingID, err)
		return fmt.Errorf("failed to serialize listing payload: %w", err)
	}

	// Enqueue moderation
	if s.moderationHooks == nil {
		s.log.Logf("ERROR moderation hooks not configured for listing=%s", listingID)
		return fmt.Errorf("moderation hooks not configured")
	}

	// Enqueue listing text moderation
	s.log.Logf("INFO enqueueing text moderation for listing=%s", listingID)
	if err := s.moderationHooks.EnqueueAIModeration(ctx, listing.ID, "listing_text", listingPayloadStr); err != nil {
		s.log.Logf("ERROR failed to enqueue text moderation for listing=%s: %v", listingID, err)
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
			s.log.Logf("ERROR failed to enqueue moderation for media=%s listing=%s: %v", media.ID, listingID, err)
			return fmt.Errorf("failed to enqueue moderation for media %s: %w", media.ID, err)
		}
		mediaCount++
	}
	s.log.Logf("INFO enqueued moderation for %d media items for listing=%s", mediaCount, listingID)

	statusChangedAt := time.Now()
	updates := map[string]any{
		"status":               domain.StatusUnderReview,
		"latest_review_status": domain.ReviewPending,
		"published":            false,
		"published_at":         nil,
		"status_changed_at":    statusChangedAt,
	}

	if err := s.repo.PatchListing(ctx, listing.ID, updates); err != nil {
		s.log.Logf("ERROR failed to update listing status for listing=%s: %v", listingID, err)
		return fmt.Errorf("failed to update listing status: %w", err)
	}

	// Invalidate cache to ensure subsequent reads get the updated status
	publicID := s.getPropertyPublicID(ctx, listing.PropertyID)
	s.invalidateListingCache(ctx, listing.ID, listing.Slug, publicID)

	//  Fetch owner data for notification
	profileName, profileEmail, err := s.profiles.GetProfileData(ctx, listing.OwnerID.String())
	if err != nil || profileEmail == "" {
		s.log.Logf("WARN failed to fetch profile data for user=%s: %v", listing.OwnerID, err)
		return fmt.Errorf("failed to fetch listing owner profile data")
	}
	// Notify listing owner
	if s.notificationService != nil {
		s.log.Logf("INFO sending publish listing request notification for listing=%s", listingID)
		if err := s.notificationService.SendPublishListingRequestNotification(ctx, listing.Title, profileName, profileEmail); err != nil {
			s.log.Logf("ERROR failed to send publish listing request notification for listing=%s: %v", listingID, err)
			// Non-fatal
		}
	}

	s.log.Logf("INFO successfully submitted listing=%s for moderation, status=%s", listingID, domain.StatusUnderReview)
	return nil
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
