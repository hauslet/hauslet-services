package service

import (
	"context"
	"slices"
	"strings"
	"time"

	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/repository"

	"github.com/google/uuid"
)

// CreateListing creates a new listing with validation.
func (s *ServiceImpl) CreateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error) {
	if l.PropertyID == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}
	if l.OwnerID == uuid.Nil {
		return nil, domain.ErrInvalidOwnerID
	}

	// Ensure property exists
	exists, err := s.repo.PropertyExists(ctx, l.PropertyID)
	if err != nil {
		s.log.Error("failed to check property existence property=%s: %v", l.PropertyID, err)
		return nil, err
	}
	if !exists {
		s.log.Error("property not found for listing creation property=%s", l.PropertyID)
		return nil, domain.ErrPropertyNotFound
	}

	// Normalize media slice
	if l.Media == nil {
		l.Media = []domain.ListingMedia{}
	}

	schemaListing := domain.MapListingToSchema(&l)
	if schemaListing.ID == uuid.Nil {
		schemaListing.ID = uuid.New()
	}
	if schemaListing.Slug == "" {
		schemaListing.Slug = generateSlug(l.Title) + "-" + shortid()

	}

	if err := s.repo.CreateListing(ctx, schemaListing); err != nil {
		s.log.Error("failed to create listing property=%s owner=%s: %v", l.PropertyID, l.OwnerID, err)
		return nil, err
	}

	created := domain.MapListingFromSchema(schemaListing)
	s.cacheListing(ctx, created, false, created.Slug, s.getPropertyPublicID(ctx, created.PropertyID))

	s.log.Info(" created listing=%s property=%s owner=%s type=%s", schemaListing.ID, l.PropertyID, l.OwnerID, l.ListingType)
	return created, nil
}

// UpdateListing updates an existing listing with validation.
func (s *ServiceImpl) UpdateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error) {
	if l.ID == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}
	if l.PropertyID == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}
	if l.OwnerID == uuid.Nil {
		return nil, domain.ErrInvalidOwnerID
	}

	existing, err := s.ensureListing(ctx, l.ID, false)
	if err != nil {
		s.log.Error("listing not found for update listing=%s: %v", l.ID, err)
		return nil, err
	}

	// Preserve immutable fields if not set
	if l.Slug == "" {
		l.Slug = existing.Slug
	}

	// Auto-unpublish if live and critical changes are detected (excluding minor text tweaks)
	if shouldUnpublish(existing, &l) {
		if err := s.unpublishAndEnqueueModeration(ctx, existing); err != nil {
			s.log.Error("failed to unpublish listing=%s for re-moderation: %v", l.ID, err)
			return nil, err
		}
	}

	schemaListing := domain.MapListingToSchema(&l)
	if err := s.repo.UpdateListing(ctx, schemaListing); err != nil {
		s.log.Error("failed to update listing=%s: %v", l.ID, err)
		return nil, err
	}

	publicID := s.getPropertyPublicID(ctx, l.PropertyID)
	s.invalidateListingCache(ctx, l.ID, existing.Slug, publicID)

	updated := domain.MapListingFromSchema(schemaListing)
	s.cacheListing(ctx, updated, false, updated.Slug, publicID)

	s.log.Info(" updated listing=%s", l.ID)
	return updated, nil
}

// PatchListing applies partial updates to a listing.
func (s *ServiceImpl) PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Listing, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	// Load existing listing to compare changes
	existing, err := s.ensureListing(ctx, id, false)
	if err != nil {
		s.log.Error("listing not found for patch listing=%s: %v", id, err)
		return nil, err
	}

	// Auto-unpublish if live and critical changes are detected (excluding minor text tweaks)
	if shouldUnpublishPatch(existing, updates) {
		if err := s.unpublishAndEnqueueModeration(ctx, existing); err != nil {
			s.log.Error("failed to unpublish listing=%s for re-moderation (patch): %v", id, err)
			return nil, err
		}
	}

	newSlug := existing.Slug
	if v, ok := updates["slug"]; ok {
		if slugStr, ok := v.(string); ok && slugStr != "" {
			newSlug = slugStr
		}
	}
	publicID := s.getPropertyPublicID(ctx, existing.PropertyID)

	if err := s.repo.PatchListing(ctx, id, updates); err != nil {
		s.log.Error("failed to patch listing=%s: %v", id, err)
		return nil, err
	}

	s.invalidateListingCache(ctx, id, existing.Slug, publicID)
	if newSlug != "" && newSlug != existing.Slug {
		s.invalidateListingCache(ctx, id, newSlug, publicID)
	}

	s.log.Info(" patched listing=%s fields=%d", id, len(updates))
	updated, err := s.ensureListing(ctx, id, false)
	if err != nil {
		return nil, err
	}
	s.cacheListing(ctx, updated, false, updated.Slug, publicID)
	return updated, nil
}

// GetListingByID retrieves a listing by its ID.
func (s *ServiceImpl) GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	var cached domain.Listing
	if ok, err := s.getCachedValue(ctx, listingIDCacheKey(id, preloadMedia), &cached); err == nil && ok {
		s.log.Info(" listing cache hit id=%s", id)
		return &cached, nil
	} else if err != nil {
		s.log.Warn("listing cache read failed id=%s: %v", id, err)
	}

	l, err := s.repo.GetListingByID(ctx, id, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	domainListing := domain.MapListingFromSchema(l)
	s.cacheListing(ctx, domainListing, preloadMedia, domainListing.Slug, s.getPropertyPublicID(ctx, domainListing.PropertyID))
	return domainListing, nil
}

// GetListingByPublicID retrieves a listing by its public identifier (alias of slug).
func (s *ServiceImpl) GetListingByPublicID(ctx context.Context, publicID string, preloadMedia bool) (*domain.Listing, error) {
	if publicID == "" {
		return nil, domain.ErrInvalidSlug
	}

	var cached domain.Listing
	if ok, err := s.getCachedValue(ctx, listingPublicIDCacheKey(publicID, preloadMedia), &cached); err == nil && ok {
		return &cached, nil
	} else if err != nil {
		s.log.Warn("listing cache read failed public_id=%s: %v", publicID, err)
	}

	l, err := s.repo.GetListingByPublicID(ctx, publicID, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	domainListing := domain.MapListingFromSchema(l)
	s.cacheListing(ctx, domainListing, preloadMedia, domainListing.Slug, publicID)
	return domainListing, nil
}

// GetListingBySlug retrieves a listing by its slug.
func (s *ServiceImpl) GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*domain.Listing, error) {
	if slug == "" {
		return nil, domain.ErrInvalidSlug
	}

	var cached domain.Listing
	if ok, err := s.getCachedValue(ctx, listingSlugCacheKey(slug, preloadMedia), &cached); err == nil && ok {
		return &cached, nil
	} else if err != nil {
		s.log.Warn("listing cache read failed slug=%s: %v", slug, err)
	}

	l, err := s.repo.GetListingBySlug(ctx, slug, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	domainListing := domain.MapListingFromSchema(l)
	s.cacheListing(ctx, domainListing, preloadMedia, slug, s.getPropertyPublicID(ctx, domainListing.PropertyID))
	return domainListing, nil
}

// GetListingsByIDs retrieves multiple listings by their IDs.
func (s *ServiceImpl) GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]domain.Listing, error) {
	if len(ids) == 0 {
		return []domain.Listing{}, nil
	}

	// Validate all IDs
	if slices.Contains(ids, uuid.Nil) {
		return nil, domain.ErrInvalidListingID
	}

	schemaListings, err := s.repo.GetListingsByIDs(ctx, ids, preloadMedia)
	if err != nil {
		return nil, err
	}

	return domain.MapListingsFromSchema(schemaListings), nil
}

// GetListingsByPropertyIDs retrieves all listings for multiple properties.
func (s *ServiceImpl) GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]domain.Listing, error) {
	if len(propertyIDs) == 0 {
		return []domain.Listing{}, nil
	}

	// Validate all IDs
	if slices.Contains(propertyIDs, uuid.Nil) {
		return nil, domain.ErrInvalidPropertyID
	}

	schemaListings, err := s.repo.GetListingsByPropertyIDs(ctx, propertyIDs)
	if err != nil {
		return nil, err
	}

	return domain.MapListingsFromSchema(schemaListings), nil
}

// ListListings retrieves listings based on filter and pagination.
func (s *ServiceImpl) ListListings(ctx context.Context, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error) {
	repoFilter := mapListingFilterToRepo(filter)
	repoPagination := repository.Pagination{
		Limit:  page.Limit,
		Offset: page.Offset,
	}

	result, err := s.repo.ListListings(ctx, repoFilter, repoPagination)
	if err != nil {
		return nil, 0, err
	}

	listings := domain.MapListingsFromSchema(result.Items)
	return listings, result.TotalCount, nil
}

// DeleteListing deletes a listing (soft or hard).
func (s *ServiceImpl) DeleteListing(ctx context.Context, id uuid.UUID, hard bool) error {
	if id == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	var existing *domain.Listing
	if l, err := s.repo.GetListingByID(ctx, id, false); err == nil && l != nil {
		existing = domain.MapListingFromSchema(l)
		// Enforce business delete permission when applicable
		if existing.OwnerType == domain.OwnerBusiness {
			if s.businessAuthorizer == nil {
				return domain.ErrForbidden
			}
			if err := s.businessAuthorizer.CanDeleteListing(ctx, existing.OwnerID); err != nil {
				return domain.ErrForbidden
			}
		}
	}

	if hard {
		s.log.Warn("hard deleting listing=%s", id)
		if err := s.repo.HardDeleteListing(ctx, id); err != nil {
			s.log.Error("failed to hard delete listing=%s: %v", id, err)
			return err
		}
		s.log.Info(" hard deleted listing=%s", id)
		if existing != nil {
			s.invalidateListingCache(ctx, id, existing.Slug, s.getPropertyPublicID(ctx, existing.PropertyID))
		} else {
			s.invalidateListingCache(ctx, id, "", "")
		}
		return nil
	}

	if err := s.repo.SoftDeleteListing(ctx, id); err != nil {
		s.log.Error("failed to soft delete listing=%s: %v", id, err)
		return err
	}
	s.log.Info(" soft deleted listing=%s", id)
	if existing != nil {
		s.invalidateListingCache(ctx, id, existing.Slug, s.getPropertyPublicID(ctx, existing.PropertyID))
	} else {
		s.invalidateListingCache(ctx, id, "", "")
	}
	return nil
}

// GetListingCompleteness calculates the completeness of a listing.
func (s *ServiceImpl) GetListingCompleteness(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID) (*domain.ListingCompleteness, error) {
	if listingID == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}
	if requesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	listing, err := s.ensureListing(ctx, listingID, true)
	if err != nil {
		s.log.Error("failed to fetch listing for completeness listing=%s: %v", listingID, err)
		return nil, err
	}
	if listing.OwnerID != requesterID {
		// For business-owned listings, allow authorized business members (via authorizer) to view completeness.
		if listing.OwnerType == domain.OwnerBusiness {
			if s.businessAuthorizer == nil {
				s.log.Warn("business authorizer not configured for completeness listing=%s", listingID)
				return nil, domain.ErrForbidden
			}
			if err := s.businessAuthorizer.CanEditListing(ctx, listing.OwnerID); err != nil {
				s.log.Warn("unauthorized completeness check listing=%s requester=%s owner=%s", listingID, requesterID, listing.OwnerID)
				return nil, domain.ErrForbidden
			}
		} else {
			s.log.Warn("unauthorized completeness check listing=%s requester=%s owner=%s", listingID, requesterID, listing.OwnerID)
			return nil, domain.ErrForbidden
		}
	}

	property, err := s.ensureProperty(ctx, listing.PropertyID)
	if err != nil {
		s.log.Error("failed to fetch property for completeness property=%s listing=%s: %v", listing.PropertyID, listingID, err)
		return nil, err
	}

	// Basic listing information
	hasBasicInfo := listing.Title != "" && listing.Slug != "" && listing.ListingType != ""

	// Property location and type information
	hasPropertyInfo := property.Address != "" && property.City != "" &&
		property.State != "" && property.Country != "" && property.PropertyType != ""

	// Room details (bedrooms and bathrooms) - only required for residential properties
	hasRoomDetails := false
	if property.PropertyClass == domain.ClassCommercial {
		// Commercial properties don't need bedrooms/bathrooms
		hasRoomDetails = true
	} else {
		// Residential properties require bedrooms and bathrooms
		if property.Bedrooms != nil && *property.Bedrooms > 0 &&
			property.Bathrooms != nil && *property.Bathrooms > 0 {
			hasRoomDetails = true
		}
	}

	// Property size information
	hasPropertySize := property.SquareMeters > 0 || (property.FloorArea != nil && *property.FloorArea > 0)

	// Amenities check
	hasAmenities := false
	for _, amenityGroup := range property.Amenities {
		if len(amenityGroup.Items) > 0 {
			hasAmenities = true
			break
		}
	}

	// Owner type validation
	hasOwnerType := listing.OwnerType != ""

	// Furnishing type (important for rent/shortlet)
	hasFurnishingType := property.FurnishingType != ""

	// Pricing information based on listing type
	hasPricingInfo := false
	switch listing.ListingType {
	case domain.ListingShortLet:
		hasPricingInfo = listing.ShortletDetails != nil && listing.ShortletDetails.NightlyRate > 0
	case domain.ListingRent:
		hasPricingInfo = listing.RentalDetails != nil && listing.RentalDetails.RentalPrice > 0
	case domain.ListingSale:
		hasPricingInfo = listing.SaleDetails != nil && listing.SaleDetails.SalePrice > 0
	}

	// Media validation
	hasImages := len(listing.Media) > 0

	// Description quality check (minimum 100 characters)
	const minDescriptionLength = 100
	hasQualityDescription := strings.TrimSpace(listing.Description) != "" &&
		len(strings.TrimSpace(listing.Description)) >= minDescriptionLength

	// Listing-type specific details validation
	hasListingSpecificDetails := false
	switch listing.ListingType {
	case domain.ListingShortLet:
		if listing.ShortletDetails != nil {
			hasListingSpecificDetails = listing.ShortletDetails.MaxGuests > 0 &&
				listing.ShortletDetails.MinNights > 0 &&
				listing.ShortletDetails.AccommodationType != ""
		}
	case domain.ListingRent:
		if listing.RentalDetails != nil {
			hasListingSpecificDetails = listing.RentalDetails.MinRentalPeriod > 0 &&
				listing.RentalDetails.RentalPricePeriod != ""
		}
	case domain.ListingSale:
		if listing.SaleDetails != nil {
			hasListingSpecificDetails = listing.SaleDetails.OwnershipTitle != ""
		}
	}

	missingFields := make([]string, 0, 11)
	recommendations := make([]string, 0, 11)

	// Calculate weighted score and build recommendations
	completionScore := 0

	if hasBasicInfo {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "basic_info")
		recommendations = append(recommendations, "Add a title and listing type.")
	}

	if hasPropertyInfo {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "property_info")
		recommendations = append(recommendations, "Provide address, city, state, country, and property type.")
	}

	if hasRoomDetails {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "room_details")
		if property.PropertyClass == domain.ClassResidential {
			recommendations = append(recommendations, "Specify the number of bedrooms and bathrooms for this residential property.")
		}
	}

	if hasPropertySize {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "property_size")
		recommendations = append(recommendations, "Add property size (square meters or floor area).")
	}

	if hasAmenities {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "amenities")
		recommendations = append(recommendations, "Add property amenities (e.g., parking, gym, pool, etc.).")
	}

	if hasOwnerType {
		completionScore += 5
	} else {
		missingFields = append(missingFields, "owner_type")
		recommendations = append(recommendations, "Specify the owner type (landlord, agent, business, or individual).")
	}

	if hasFurnishingType {
		completionScore += 5
	} else {
		missingFields = append(missingFields, "furnishing_type")
		recommendations = append(recommendations, "Specify the furnishing type (furnished, semi-furnished, or unfurnished).")
	}

	if hasPricingInfo {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "pricing")
		recommendations = append(recommendations, "Set pricing details based on the listing type.")
	}

	if hasImages {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "images")
		recommendations = append(recommendations, "Upload at least one image.")
	}

	if hasQualityDescription {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "description")
		if strings.TrimSpace(listing.Description) == "" {
			recommendations = append(recommendations, "Add a detailed description to highlight the property (minimum 100 characters).")
		} else {
			recommendations = append(recommendations, "Expand the description to at least 100 characters for better visibility.")
		}
	}

	if hasListingSpecificDetails {
		completionScore += 10
	} else {
		missingFields = append(missingFields, "listing_specific_details")
		switch listing.ListingType {
		case domain.ListingShortLet:
			recommendations = append(recommendations, "Add shortlet-specific details: max guests, minimum nights, and accommodation type.")
		case domain.ListingRent:
			recommendations = append(recommendations, "Add rental-specific details: minimum rental period and rental price period.")
		case domain.ListingSale:
			recommendations = append(recommendations, "Add sale-specific details: ownership title information.")
		}
	}

	// Ready to publish requires all critical fields
	readyToPublish := hasBasicInfo && hasPropertyInfo && hasRoomDetails && hasPropertySize &&
		hasAmenities && hasOwnerType && hasFurnishingType && hasPricingInfo && hasImages &&
		hasQualityDescription && hasListingSpecificDetails

	s.log.Info(" listing completeness listing=%s score=%d%% ready=%v", listingID, completionScore, readyToPublish)

	return &domain.ListingCompleteness{
		ListingID:        listingID,
		HasBasicInfo:     hasBasicInfo,
		HasPropertyInfo:  hasPropertyInfo,
		HasPricingInfo:   hasPricingInfo,
		HasImages:        hasImages,
		HasDescription:   hasQualityDescription,
		CompletionScore:  completionScore,
		ReadyToPublish:   readyToPublish,
		MissingFields:    missingFields,
		Recommendations:  recommendations,
		LastCalculatedAt: time.Now(),
	}, nil
}
