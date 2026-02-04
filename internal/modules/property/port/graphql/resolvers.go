package graphql

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"hauslet/config"
	profiledomain "hauslet/internal/modules/profile/domain"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/service"
	"hauslet/internal/transport/graph/helpers"
	"hauslet/internal/transport/graph/loaders"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles property-specific GraphQL fields.
type Resolver struct {
	propertyService service.PropertyService
	log             *slog.Logger
	cdnHost         string
}

func NewResolver(propertyService service.PropertyService, cfg *config.StorageConfig, log *slog.Logger) *Resolver {
	return &Resolver{
		propertyService: propertyService,
		cdnHost:         cfg.R2.CDNHost,
		log:             log,
	}
}

// ===========================
// QUERY RESOLVERS
// ===========================

// ListingByPublicId retrieves a listing by its public ID.
func (r *Resolver) ListingByPublicId(ctx context.Context, publicId string) (*domain.Listing, error) {
	userID, _ := viewer.GetOptionalUserIDFromContext(ctx)

	listing, err := r.propertyService.GetListingByPublicID(ctx, publicId, true)
	if err != nil {
		r.log.Error("failed to get listing by public ID", "public_id", publicId, "error", err)
		return nil, err
	}
	if listing != nil && len(listing.Media) > 0 {
		listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
	}
	r.propertyService.LocalizeListing(ctx, listing)
	return sanitizeListingForViewer(ctx, listing, userID), nil
}

// Listing retrieves a listing by ID.
func (r *Resolver) Listing(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	userID, _ := viewer.GetOptionalUserIDFromContext(ctx)

	if l := loaders.For(ctx); l != nil && l.Listing != nil {
		if listing, err := l.Listing.Load(ctx, id); err == nil {
			if listing != nil && len(listing.Media) > 0 {
				listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
			}
			r.propertyService.LocalizeListing(ctx, listing)
			return sanitizeListingForViewer(ctx, listing, userID), nil
		}
	}

	listing, err := r.propertyService.GetListingByID(ctx, id, true)
	if err != nil {
		r.log.Error("failed to get listing by ID", "listing_id", id, "error", err)
		return nil, err
	}
	if listing != nil && len(listing.Media) > 0 {
		listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
	}
	r.propertyService.LocalizeListing(ctx, listing)
	return sanitizeListingForViewer(ctx, listing, userID), nil
}

// ListingBySlug retrieves a listing by its slug.
func (r *Resolver) ListingBySlug(ctx context.Context, slug string) (*domain.Listing, error) {
	userID, _ := viewer.GetOptionalUserIDFromContext(ctx)
	listing, err := r.propertyService.GetListingBySlug(ctx, slug, true)
	if err != nil {
		r.log.Error("failed to get listing by slug", "slug", slug, "error", err)
		return nil, err
	}
	if listing != nil && len(listing.Media) > 0 {
		listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
	}
	r.propertyService.LocalizeListing(ctx, listing)
	return sanitizeListingForViewer(ctx, listing, userID), nil
}

// Listings retrieves listings with cursor-based pagination.
func (r *Resolver) Listings(ctx context.Context, filter *model.ListingFilterInput, first *int, after *string) (*model.ListingConnection, error) {
	userID, _ := viewer.GetOptionalUserIDFromContext(ctx)
	limit := 20
	offset := 0

	if first != nil && *first > 0 {
		limit = min(*first, 100)
	}

	if after != nil {
		var err error
		offset, err = decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	serviceFilter := mapListingFilterToService(filter)
	servicePage := service.Pagination{
		Limit:  limit,
		Offset: offset,
	}

	listings, total, err := r.propertyService.ListListings(ctx, serviceFilter, servicePage)
	if err != nil {
		r.log.Error("failed to list listings", "error", err)
		return nil, err
	}

	r.warmListingLoaders(ctx, listings)
	for i := range listings {
		if len(listings[i].Media) > 0 {
			listings[i].Media = helpers.BuildListingMediaURLs(listings[i].Media, r.cdnHost)
		}
		r.propertyService.LocalizeListing(ctx, &listings[i])
	}

	return buildListingConnection(listings, total, offset, ctx, userID), nil
}

// ListingsByProperty retrieves listings for a specific property.
func (r *Resolver) ListingsByProperty(ctx context.Context, propertyID uuid.UUID, first *int, after *string) (*model.ListingConnection, error) {
	filter := &model.ListingFilterInput{
		PropertyID: &propertyID,
	}
	return r.Listings(ctx, filter, first, after)
}

// MyListings retrieves listings owned by the authenticated user.
func (r *Resolver) MyListings(ctx context.Context, filter *model.ListingFilterInput, first *int, after *string) (*model.ListingConnection, error) {
	ownerID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if filter == nil {
		filter = &model.ListingFilterInput{}
	}
	filter.OwnerID = &ownerID

	return r.Listings(ctx, filter, first, after)
}

// ListingCompleteness calculates listing completeness.
func (r *Resolver) ListingCompleteness(ctx context.Context, listingID uuid.UUID) (*domain.ListingCompleteness, error) {

	requesterID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	completeness, err := r.propertyService.GetListingCompleteness(ctx, listingID, requesterID)
	if err != nil {
		r.log.Error("failed to get listing completeness", "listing_id", listingID, "error", err)
		return nil, err
	}

	return completeness, nil
}

// BusinessListings retrieves all listings owned by a specific business.
func (r *Resolver) BusinessListings(ctx context.Context, businessID uuid.UUID,
	filter *model.ListingFilterInput, first *int, after *string) (*model.ListingConnection, error) {

	if businessID == uuid.Nil {
		r.log.Warn("BusinessListings called with nil businessID")
		return nil, fmt.Errorf("businessID is required")
	}

	limit := 20
	offset := 0

	if first != nil && *first > 0 {
		limit = min(*first, 100)
	}

	if after != nil {
		var err error
		offset, err = decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	serviceFilter := mapListingFilterToService(filter)
	servicePage := service.Pagination{
		Limit:  limit,
		Offset: offset,
	}

	// filter scoped to business owner
	ownerType := domain.OwnerBusiness
	serviceFilter.OwnerID = &businessID
	serviceFilter.OwnerTypes = []domain.OwnerType{ownerType}

	listings, total, err := r.propertyService.ListListings(ctx, serviceFilter, servicePage)
	if err != nil {
		r.log.Error("failed to get business listings", "business_id", businessID, "error", err)
		return nil, err
	}

	r.warmListingLoaders(ctx, listings)
	for i := range listings {
		if len(listings[i].Media) > 0 {
			listings[i].Media = helpers.BuildListingMediaURLs(listings[i].Media, r.cdnHost)
		}
		r.propertyService.LocalizeListing(ctx, &listings[i])
	}

	return buildListingConnection(listings, total, offset, ctx, businessID), nil
}

// MyIndividualListings retrieves individual listings owned by the authenticated user.
func (r *Resolver) MyIndividualListings(ctx context.Context, filter *model.ListingFilterInput, first *int, after *string) (*model.ListingConnection, error) {

	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Error("invalid user ID in myIndividualListings", "user_id", userID)
		return nil, err
	}

	limit := 20
	offset := 0

	if first != nil && *first > 0 {
		limit = min(*first, 100)
	}

	if after != nil {
		var err error
		offset, err = decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	serviceFilter := mapListingFilterToService(filter)
	servicePage := service.Pagination{
		Limit:  limit,
		Offset: offset,
	}

	ownerType := domain.OwnerIndividual
	serviceFilter.OwnerID = &userID
	serviceFilter.OwnerTypes = []domain.OwnerType{ownerType}

	listings, total, err := r.propertyService.ListListings(ctx, serviceFilter, servicePage)
	if err != nil {
		r.log.Error("failed to get individual listings", "user_id", userID, "error", err)
		return nil, err
	}

	r.warmListingLoaders(ctx, listings)
	for i := range listings {
		if len(listings[i].Media) > 0 {
			listings[i].Media = helpers.BuildListingMediaURLs(listings[i].Media, r.cdnHost)
		}
		r.propertyService.LocalizeListing(ctx, &listings[i])
	}

	return buildListingConnection(listings, total, offset, ctx, userID), nil
}

// ===========================
// MUTATION RESOLVERS
// ===========================

// CreateListing creates a new listing with its property in a transaction.
func (r *Resolver) CreateListing(ctx context.Context, input model.CreateListingInput) (*domain.Listing, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Error("invalid user ID in myIndividualListings", "user_id", userID)
		return nil, err
	}

	if input.Property == nil {
		r.log.Warn("createListing called without property payload", "user_id", userID)
		return nil, fmt.Errorf("property payload is required")
	}

	// Determine ownerID based on ownerType
	var ownerID uuid.UUID
	if input.OwnerType == domain.OwnerBusiness {
		// For business listings, businessID must be provided
		if input.BusinessID == nil {
			r.log.Warn("createListing called with business ownerType but no businessID", "user_id", userID)
			return nil, fmt.Errorf("businessID is required when ownerType is business")
		}
		ownerID = *input.BusinessID
		r.log.Info("creating business listing", "business_id", ownerID, "user_id", userID)
	} else {
		// For individual listings, use userID
		ownerID = userID
		r.log.Info("creating individual listing", "user_id", userID)
	}

	// Map inputs to domain models
	property := mapCreateListingPropertyInput(input.Property, ownerID)
	listing := mapCreateListingInput(input, ownerID)

	createdProperty, createdListing, err := r.propertyService.CreatePropertyWithListing(ctx, *property, listing)
	if err != nil {
		r.log.Error("failed to create property with listing", "user_id", userID, "error", err)
		return nil, err
	}

	r.log.Info("property and listing created", "property_id", createdProperty.ID, "listing_id", createdListing.ID, "user_id", userID)
	return sanitizeListingForViewer(ctx, createdListing, userID), nil
}

// UpdateListing updates an existing listing.
func (r *Resolver) UpdateListing(ctx context.Context, id uuid.UUID, input model.UpdateListingInput) (*domain.Listing, error) {

	requesterID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Error("invalid user ID in myIndividualListings", "user_id", requesterID)
		return nil, err
	}

	propUpdates := mapUpdateListingPropertyInput(input.Property)
	listingUpdates := mapListingUpdateInput(&input)

	// Extract and remove JSON patches to avoid overwrites by the main update
	var shortletPatch, rentalPatch, salePatch map[string]any

	if patch, ok := listingUpdates["shortlet_details"]; ok {
		if patchMap, ok := patch.(map[string]any); ok {
			shortletPatch = patchMap
			delete(listingUpdates, "shortlet_details")
		}
	}
	if patch, ok := listingUpdates["rental_details"]; ok {
		if patchMap, ok := patch.(map[string]any); ok {
			rentalPatch = patchMap
			delete(listingUpdates, "rental_details")
		}
	}
	if patch, ok := listingUpdates["sale_details"]; ok {
		if patchMap, ok := patch.(map[string]any); ok {
			salePatch = patchMap
			delete(listingUpdates, "sale_details")
		}
	}

	updated, err := r.propertyService.UpdateListingWithProperty(ctx, id, listingUpdates, propUpdates, requesterID)
	if err != nil {
		r.log.Error("failed to update listing", "listing_id", id, "error", err)
		return nil, err
	}

	// Apply patches safely
	patched := false
	if shortletPatch != nil {
		if err := r.propertyService.PatchShortletDetails(ctx, id, shortletPatch); err != nil {
			r.log.Error("failed to patch shortlet details", "listing_id", id, "error", err)
			return nil, err
		}
		patched = true
	}
	if rentalPatch != nil {
		if err := r.propertyService.PatchRentalDetails(ctx, id, rentalPatch); err != nil {
			r.log.Error("failed to patch rental details", "listing_id", id, "error", err)
			return nil, err
		}
		patched = true
	}
	if salePatch != nil {
		if err := r.propertyService.PatchSaleDetails(ctx, id, salePatch); err != nil {
			r.log.Error("failed to patch sale details", "listing_id", id, "error", err)
			return nil, err
		}
		patched = true
	}

	if patched {
		// Refetch to get the fully merged state
		refetched, err := r.propertyService.GetListingByID(ctx, id, true)
		if err != nil {
			r.log.Error("failed to refetch patched listing", "listing_id", id, "error", err)
			return nil, err
		}
		updated = refetched
	}

	r.log.Info("listing updated", "listing_id", id, "user_id", requesterID)
	return sanitizeListingForViewer(ctx, updated, requesterID), nil
}

// DeleteListing deletes a listing.
func (r *Resolver) DeleteListing(ctx context.Context, id uuid.UUID, hard *bool) (bool, error) {
	requesterID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Error("invalid user ID in myIndividualListings", "user_id", requesterID)
		return false, err
	}

	isHard := false
	if hard != nil {
		isHard = *hard
	}

	if err := r.propertyService.DeleteListing(ctx, id, isHard); err != nil {
		r.log.Error("failed to delete listing", "listing_id", id, "error", err)
		return false, err
	}

	r.log.Info("listing deleted", "listing_id", id, "hard_delete", isHard, "user_id", requesterID)
	return true, nil
}

// PublishListing publishes a listing.
func (r *Resolver) PublishListing(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	// Requirement already checked, but extracting ID for service call
	requesterID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if err := r.propertyService.PublishListingRequest(ctx, id, requesterID); err != nil {
		r.log.Error("failed to publish listing", "listing_id", id, "error", err)
		return nil, err
	}
	updatedListing, err := r.propertyService.GetListingByID(ctx, id, false)
	if err != nil {
		r.log.Error("failed to get listing for publishing", "listing_id", id, "error", err)
		return nil, err
	}

	r.log.Info("listing publish request successful", "listing_id", id, "requester_id", requesterID)
	return sanitizeListingForViewer(ctx, updatedListing, requesterID), nil
}

// UnpublishListing unpublishes a listing.
func (r *Resolver) UnpublishListing(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	// Requirement already checked, but extracting ID for service call
	requesterID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	updatedListing, err := r.propertyService.UnpublishListing(ctx, id, requesterID)
	if err != nil {
		r.log.Error("failed to unpublish listing", "listing_id", id, "error", err)
		return nil, err
	}

	r.log.Info("listing unpublished", "listing_id", id, "requester_id", requesterID)
	return sanitizeListingForViewer(ctx, updatedListing, requesterID), nil
}

// GenerateListingDescription generates a listing description using AI.
func (r *Resolver) GenerateListingDescription(ctx context.Context, input model.GenerateListingDescriptionInput) (string, error) {
	_, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return "", err
	}
	domainInput := domain.GenerateListingDescriptionInput{
		PropertyType: input.PropertyType,
		City:         input.City,
		State:        input.State,
		Bedrooms:     input.Bedrooms,
		Bathrooms:    input.Bathrooms,
		Amenities:    input.Amenities,
		Highlights:   input.Highlights,
	}

	if input.Tone != nil {
		domainInput.Tone = *input.Tone
	}

	return r.propertyService.GenerateListingDescription(ctx, domainInput)
}

// ===========================
// FIELD RESOLVERS
// ===========================

// propertyForListing fetches the property linked to a listing, honoring viewer visibility.
func (r *Resolver) propertyForListing(ctx context.Context, listing *domain.Listing) (*domain.Property, error) {
	if listing == nil {
		return nil, domain.ErrPropertyNotFound
	}

	userID, _ := viewer.GetOptionalUserIDFromContext(ctx)
	if l := loaders.For(ctx); l != nil && l.Property != nil {
		if p, err := l.Property.Load(ctx, listing.PropertyID); err == nil {
			return sanitizePropertyForViewer(ctx, p, userID), nil
		}
	}

	property, err := r.propertyService.GetPropertyByID(ctx, listing.PropertyID)
	if err != nil {
		r.log.Error("failed to get property", "property_id", listing.PropertyID, "error", err)
		return nil, err
	}
	return sanitizePropertyForViewer(ctx, property, userID), nil
}

// Address resolves property.address via the listing relationship.
func (r *Resolver) ListingAddress(ctx context.Context, obj *domain.Listing) (string, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.Address, nil
}

func (r *Resolver) ListingState(ctx context.Context, obj *domain.Listing) (string, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.State, nil
}

func (r *Resolver) ListingPostalCode(ctx context.Context, obj *domain.Listing) (string, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.PostalCode, nil
}

func (r *Resolver) ListingCountry(ctx context.Context, obj *domain.Listing) (domain.CountryCode, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.Country, nil
}

func (r *Resolver) ListingLocation(ctx context.Context, obj *domain.Listing) (*domain.Location, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.Location, nil
}

func (r *Resolver) ListingPropertyClass(ctx context.Context, obj *domain.Listing) (domain.PropertyClass, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.PropertyClass, nil
}

func (r *Resolver) ListingPropertyType(ctx context.Context, obj *domain.Listing) (domain.PropertyType, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.PropertyType, nil
}

func (r *Resolver) ListingFurnishingType(ctx context.Context, obj *domain.Listing) (domain.FurnishingType, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.FurnishingType, nil
}

func (r *Resolver) ListingPropertyCondition(ctx context.Context, obj *domain.Listing) (domain.PropertyCondition, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return "", err
	}
	return property.PropertyCondition, nil
}

func (r *Resolver) ListingBedrooms(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.Bedrooms, nil
}

func (r *Resolver) ListingBathrooms(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.Bathrooms, nil
}

func (r *Resolver) ListingToilets(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.Toilets, nil
}

func (r *Resolver) ListingHalfBathrooms(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.HalfBathrooms, nil
}

func (r *Resolver) ListingFloors(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.Floors, nil
}

func (r *Resolver) ListingUnits(ctx context.Context, obj *domain.Listing) (int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return 0, err
	}
	return property.Units, nil
}

func (r *Resolver) ListingSquareMeters(ctx context.Context, obj *domain.Listing) (float64, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return 0, err
	}
	return property.SquareMeters, nil
}

func (r *Resolver) ListingFloorArea(ctx context.Context, obj *domain.Listing) (*float64, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.FloorArea, nil
}

func (r *Resolver) ListingAmenities(ctx context.Context, obj *domain.Listing) ([]domain.AmenityGroup, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.Amenities, nil
}

func (r *Resolver) ListingFeaturesCommercial(ctx context.Context, obj *domain.Listing) ([]domain.AmenityGroup, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("failed to get property for listing", "listing_id", obj.ID, "error", err)
		return nil, err
	}
	return property.FeaturesCommercial, nil
}

// Listings resolves the listings field on Property.
func (r *Resolver) PropertyListings(ctx context.Context, obj *domain.Property, first *int, after *string) (*model.ListingConnection, error) {
	filter := &model.ListingFilterInput{
		PropertyID: &obj.ID,
	}
	return r.Listings(ctx, filter, first, after)
}

// Property resolves the property field on Listing.
func (r *Resolver) ListingProperty(ctx context.Context, obj *domain.Listing) (*domain.Property, error) {
	return r.propertyForListing(ctx, obj)
}

// Media resolves the media field on Listing.
func (r *Resolver) ListingMedia(ctx context.Context, obj *domain.Listing, first *int) ([]*domain.ListingMedia, error) {
	media := obj.Media
	if media == nil {
		var err error
		media, err = r.propertyService.ListListingMedia(ctx, obj.ID)
		if err != nil {
			r.log.Error("failed to list media for listing", "listing_id", obj.ID, "error", err)
			return nil, err
		}
	}

	limit := len(media)
	if first != nil && *first > 0 && *first < limit {
		limit = *first
	}

	result := make([]*domain.ListingMedia, 0, limit)
	for i := 0; i < limit && i < len(media); i++ {
		m := media[i]
		result = append(result, &m)
	}

	return result, nil
}

// Thumbnails resolves the thumbnails field on ListingMedia by converting the map to an array.
func (r *Resolver) ListingMediaThumbnails(ctx context.Context, obj *domain.ListingMedia) ([]*domain.ThumbnailVariant, error) {
	if len(obj.Thumbnails) == 0 {
		return []*domain.ThumbnailVariant{}, nil
	}

	thumbnails := make([]*domain.ThumbnailVariant, 0, len(obj.Thumbnails))
	for name, thumb := range obj.Thumbnails {
		// Remove file extension from size name (e.g., "small.jpg" -> "small")
		size := strings.TrimSuffix(name, filepath.Ext(name))

		thumbnails = append(thumbnails, &domain.ThumbnailVariant{
			Size:      size,
			Key:       thumb.Key,
			URL:       thumb.URL,
			Width:     thumb.Width,
			Height:    thumb.Height,
			SizeBytes: thumb.SizeBytes,
			MimeType:  thumb.MimeType,
		})
	}

	return thumbnails, nil
}

// OwnerProfile resolves the profile of the listing owner.
func (r *Resolver) OwnerProfile(ctx context.Context, obj *domain.Listing) (*profiledomain.Profile, error) {
	if obj == nil || obj.OwnerID == uuid.Nil {
		return nil, nil
	}

	if l := loaders.For(ctx); l != nil && l.Profile != nil {
		profile, err := l.Profile.Load(ctx, obj.OwnerID.String())
		if err != nil {
			r.log.Error("failed to load profile for owner", "owner_id", obj.OwnerID, "error", err)
			return nil, err
		}
		return profile, nil
	}

	return nil, fmt.Errorf("profile loader unavailable")
}

func (r *Resolver) warmListingLoaders(ctx context.Context, listings []domain.Listing) {
	if len(listings) == 0 {
		return
	}

	l := loaders.For(ctx)
	if l == nil {
		return
	}

	if l.Property != nil {
		unique := make(map[uuid.UUID]struct{}, len(listings))
		for _, listing := range listings {
			if listing.PropertyID != uuid.Nil {
				unique[listing.PropertyID] = struct{}{}
			}
		}
		if len(unique) > 0 {
			ids := make([]uuid.UUID, 0, len(unique))
			for id := range unique {
				ids = append(ids, id)
			}
			if _, err := l.Property.LoadMany(ctx, ids); err != nil && r.log != nil {
				r.log.Warn("failed to preload properties for listings", "error", err)
			}
		}
	}

	if l.Profile != nil {
		unique := make(map[string]struct{}, len(listings))
		for _, listing := range listings {
			if listing.OwnerType == domain.OwnerIndividual && listing.OwnerID != uuid.Nil {
				unique[listing.OwnerID.String()] = struct{}{}
			}
		}
		if len(unique) > 0 {
			ids := make([]string, 0, len(unique))
			for id := range unique {
				ids = append(ids, id)
			}
			if _, err := l.Profile.LoadMany(ctx, ids); err != nil && r.log != nil {
				r.log.Warn("failed to preload profiles for listings", "error", err)
			}
		}
	}
}
