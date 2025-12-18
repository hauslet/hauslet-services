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
		s.log.Logf("ERROR failed to check property existence property=%s: %v", l.PropertyID, err)
		return nil, err
	}
	if !exists {
		s.log.Logf("ERROR property not found for listing creation property=%s", l.PropertyID)
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
		s.log.Logf("ERROR failed to create listing property=%s owner=%s: %v", l.PropertyID, l.OwnerID, err)
		return nil, err
	}

	s.log.Logf("INFO created listing=%s property=%s owner=%s type=%s", schemaListing.ID, l.PropertyID, l.OwnerID, l.ListingType)
	return domain.MapListingFromSchema(schemaListing), nil
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
		s.log.Logf("ERROR listing not found for update listing=%s: %v", l.ID, err)
		return nil, err
	}

	// Preserve immutable fields if not set
	if l.Slug == "" {
		l.Slug = existing.Slug
	}

	// Auto-unpublish if live and critical changes are detected (excluding minor text tweaks)
	if shouldUnpublish(existing, &l) {
		if err := s.unpublishAndEnqueueModeration(ctx, existing); err != nil {
			s.log.Logf("ERROR failed to unpublish listing=%s for re-moderation: %v", l.ID, err)
			return nil, err
		}
	}

	schemaListing := domain.MapListingToSchema(&l)
	if err := s.repo.UpdateListing(ctx, schemaListing); err != nil {
		s.log.Logf("ERROR failed to update listing=%s: %v", l.ID, err)
		return nil, err
	}

	s.log.Logf("INFO updated listing=%s", l.ID)
	return domain.MapListingFromSchema(schemaListing), nil
}

// PatchListing applies partial updates to a listing.
func (s *ServiceImpl) PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Listing, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	// Load existing listing to compare changes
	existing, err := s.ensureListing(ctx, id, false)
	if err != nil {
		s.log.Logf("ERROR listing not found for patch listing=%s: %v", id, err)
		return nil, err
	}

	// Auto-unpublish if live and critical changes are detected (excluding minor text tweaks)
	if shouldUnpublishPatch(existing, updates) {
		if err := s.unpublishAndEnqueueModeration(ctx, existing); err != nil {
			s.log.Logf("ERROR failed to unpublish listing=%s for re-moderation (patch): %v", id, err)
			return nil, err
		}
	}

	if err := s.repo.PatchListing(ctx, id, updates); err != nil {
		s.log.Logf("ERROR failed to patch listing=%s: %v", id, err)
		return nil, err
	}

	s.log.Logf("INFO patched listing=%s fields=%d", id, len(updates))
	return s.ensureListing(ctx, id, false)
}

// GetListingByID retrieves a listing by its ID.
func (s *ServiceImpl) GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	return s.ensureListing(ctx, id, preloadMedia)
}

// GetListingByPublicID retrieves a listing by its public identifier (alias of slug).
func (s *ServiceImpl) GetListingByPublicID(ctx context.Context, publicID string, preloadMedia bool) (*domain.Listing, error) {
	if publicID == "" {
		return nil, domain.ErrInvalidSlug
	}

	l, err := s.repo.GetListingByPublicID(ctx, publicID, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	return domain.MapListingFromSchema(l), nil
}

// GetListingBySlug retrieves a listing by its slug.
func (s *ServiceImpl) GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*domain.Listing, error) {
	if slug == "" {
		return nil, domain.ErrInvalidSlug
	}

	l, err := s.repo.GetListingBySlug(ctx, slug, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	return domain.MapListingFromSchema(l), nil
}

// GetListingsByIDs retrieves multiple listings by their IDs.
func (s *ServiceImpl) GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]domain.Listing, error) {
	if len(ids) == 0 {
		return []domain.Listing{}, nil
	}

	// Validate all IDs
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, domain.ErrInvalidListingID
		}
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

	if hard {
		s.log.Logf("WARN hard deleting listing=%s", id)
		if err := s.repo.HardDeleteListing(ctx, id); err != nil {
			s.log.Logf("ERROR failed to hard delete listing=%s: %v", id, err)
			return err
		}
		s.log.Logf("INFO hard deleted listing=%s", id)
		return nil
	}

	if err := s.repo.SoftDeleteListing(ctx, id); err != nil {
		s.log.Logf("ERROR failed to soft delete listing=%s: %v", id, err)
		return err
	}
	s.log.Logf("INFO soft deleted listing=%s", id)
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
		s.log.Logf("ERROR failed to fetch listing for completeness listing=%s: %v", listingID, err)
		return nil, err
	}
	if listing.OwnerID != requesterID {
		s.log.Logf("WARN unauthorized completeness check listing=%s requester=%s owner=%s", listingID, requesterID, listing.OwnerID)
		return nil, domain.ErrForbidden
	}

	property, err := s.ensureProperty(ctx, listing.PropertyID)
	if err != nil {
		s.log.Logf("ERROR failed to fetch property for completeness property=%s listing=%s: %v", listing.PropertyID, listingID, err)
		return nil, err
	}

	hasBasicInfo := listing.Title != "" && listing.Slug != "" && listing.ListingType != ""
	hasPropertyInfo := property.Address != "" && property.City != "" &&
		property.State != "" && property.Country != "" && property.PropertyType != ""

	hasPricingInfo := false
	switch listing.ListingType {
	case domain.ListingShortLet:
		hasPricingInfo = listing.ShortletDetails != nil && listing.ShortletDetails.NightlyRate > 0
	case domain.ListingRent:
		hasPricingInfo = listing.RentalDetails != nil && listing.RentalDetails.RentalPrice > 0
	case domain.ListingSale:
		hasPricingInfo = listing.SaleDetails != nil && listing.SaleDetails.SalePrice > 0
	}

	hasImages := len(listing.Media) > 0
	hasDescription := strings.TrimSpace(listing.Description) != ""

	trueCount := 0
	missingFields := make([]string, 0, 5)
	recommendations := make([]string, 0, 5)

	if hasBasicInfo {
		trueCount++
	} else {
		missingFields = append(missingFields, "basic_info")
		recommendations = append(recommendations, "Add a title and listing type.")
	}

	if hasPropertyInfo {
		trueCount++
	} else {
		missingFields = append(missingFields, "property_info")
		recommendations = append(recommendations, "Provide address, city, state, country, and property type.")
	}

	if hasPricingInfo {
		trueCount++
	} else {
		missingFields = append(missingFields, "pricing")
		recommendations = append(recommendations, "Set pricing details based on the listing type.")
	}

	if hasImages {
		trueCount++
	} else {
		missingFields = append(missingFields, "images")
		recommendations = append(recommendations, "Upload at least one image.")
	}

	if hasDescription {
		trueCount++
	} else {
		missingFields = append(missingFields, "description")
		recommendations = append(recommendations, "Add a description to highlight the property.")
	}

	readyToPublish := hasBasicInfo && hasPropertyInfo && hasPricingInfo && hasImages && hasDescription
	completionScore := trueCount * 20

	s.log.Logf("INFO listing completeness listing=%s score=%d%% ready=%v", listingID, completionScore, readyToPublish)

	return &domain.ListingCompleteness{
		ListingID:        listingID,
		HasBasicInfo:     hasBasicInfo,
		HasPropertyInfo:  hasPropertyInfo,
		HasPricingInfo:   hasPricingInfo,
		HasImages:        hasImages,
		HasDescription:   hasDescription,
		CompletionScore:  completionScore,
		ReadyToPublish:   readyToPublish,
		MissingFields:    missingFields,
		Recommendations:  recommendations,
		LastCalculatedAt: time.Now(),
	}, nil
}
