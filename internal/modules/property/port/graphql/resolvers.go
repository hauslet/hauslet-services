package graphql

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"strings"

	"hauslet/config"
	profiledomain "hauslet/internal/modules/profile/domain"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/service"
	"hauslet/internal/platform/xchange"
	"hauslet/internal/transport/graph/helpers"
	"hauslet/internal/transport/graph/loaders"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"
	localization "hauslet/internal/transport/middleware/localization"

	"github.com/google/uuid"
)

// Resolver handles property-specific GraphQL fields.
type Resolver struct {
	propertyService service.PropertyService
	log             *slog.Logger
	cdnHost         string
	fx              xchange.XChange
}

func NewResolver(propertyService service.PropertyService, cfg *config.StorageConfig, fx xchange.XChange, log *slog.Logger) *Resolver {
	return &Resolver{
		propertyService: propertyService,
		cdnHost:         cfg.R2.CDNHost,
		fx:              fx,
		log:             log,
	}
}

// ===========================
// QUERY RESOLVERS
// ===========================

// ListingByPublicId retrieves a listing by its public ID.
func (r *Resolver) ListingByPublicId(ctx context.Context, publicId string) (*domain.Listing, error) {
	listing, err := r.propertyService.GetListingByPublicID(ctx, publicId, false)
	if err != nil {
		r.log.Error("Failed to get listing by public ID %s: %v", publicId, err)
		return nil, err
	}
	if listing != nil && len(listing.Media) > 0 {
		listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
	}

	r.localizeListing(ctx, listing)
	return sanitizeListingForViewer(ctx, listing, viewer.FromContext(ctx)), nil
}

// Listing retrieves a listing by ID.
func (r *Resolver) Listing(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	if l := loaders.For(ctx); l != nil && l.Listing != nil {
		if listing, err := l.Listing.Load(ctx, id); err == nil {
			if listing != nil && len(listing.Media) > 0 {
				listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
			}
			r.localizeListing(ctx, listing)
			return sanitizeListingForViewer(ctx, listing, viewer.FromContext(ctx)), nil
		}
	}

	listing, err := r.propertyService.GetListingByID(ctx, id, false)
	if err != nil {
		r.log.Error("Failed to get listing by ID %s: %v", id, err)
		return nil, err
	}
	if listing != nil && len(listing.Media) > 0 {
		listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
	}
	r.localizeListing(ctx, listing)
	return sanitizeListingForViewer(ctx, listing, viewer.FromContext(ctx)), nil
}

// ListingBySlug retrieves a listing by its slug.
func (r *Resolver) ListingBySlug(ctx context.Context, slug string) (*domain.Listing, error) {
	listing, err := r.propertyService.GetListingBySlug(ctx, slug, false)
	if err != nil {
		r.log.Error("Failed to get listing by slug %s: %v", slug, err)
		return nil, err
	}
	if listing != nil && len(listing.Media) > 0 {
		listing.Media = helpers.BuildListingMediaURLs(listing.Media, r.cdnHost)
	}
	r.localizeListing(ctx, listing)
	return sanitizeListingForViewer(ctx, listing, viewer.FromContext(ctx)), nil
}

// Listings retrieves listings with cursor-based pagination.
func (r *Resolver) Listings(ctx context.Context, filter *model.ListingFilterInput, first *int, after *string) (*model.ListingConnection, error) {
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
		r.log.Error("Failed to list listings: %v", err)
		return nil, err
	}

	r.warmListingLoaders(ctx, listings)
	for i := range listings {
		if len(listings[i].Media) > 0 {
			listings[i].Media = helpers.BuildListingMediaURLs(listings[i].Media, r.cdnHost)
		}
		r.localizeListing(ctx, &listings[i])
	}

	return buildListingConnection(listings, total, offset, limit, ctx, viewer.FromContext(ctx)), nil
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
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to access myListings")
		return nil, fmt.Errorf("unauthenticated")
	}

	ownerID, err := uuid.Parse(v.UserID)
	if err != nil {
		r.log.Error("Invalid user ID in myListings: %s", v.UserID)
		return nil, fmt.Errorf("invalid user ID")
	}

	if filter == nil {
		filter = &model.ListingFilterInput{}
	}
	filter.OwnerID = &ownerID

	return r.Listings(ctx, filter, first, after)
}

// ListingCompleteness calculates listing completeness.
func (r *Resolver) ListingCompleteness(ctx context.Context, listingID uuid.UUID) (*domain.ListingCompleteness, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to get listing completeness for %s", listingID)
		return nil, fmt.Errorf("unauthenticated")
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		r.log.Error("Invalid user ID in listingCompleteness: %s", v.UserID)
		return nil, fmt.Errorf("invalid user ID")
	}

	completeness, err := r.propertyService.GetListingCompleteness(ctx, listingID, requesterID)
	if err != nil {
		r.log.Error("Failed to get listing completeness for %s: %v", listingID, err)
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
		r.log.Error("Failed to get business listings for %s: %v", businessID, err)
		return nil, err
	}

	r.warmListingLoaders(ctx, listings)
	for i := range listings {
		if len(listings[i].Media) > 0 {
			listings[i].Media = helpers.BuildListingMediaURLs(listings[i].Media, r.cdnHost)
		}
		r.localizeListing(ctx, &listings[i])
	}

	return buildListingConnection(listings, total, offset, limit, ctx, viewer.FromContext(ctx)), nil
}

func (r *Resolver) localizeListing(ctx context.Context, listing *domain.Listing) {
	if listing == nil || r.fx == nil {
		return
	}

	target, ok := localization.PreferredCurrency(ctx)
	if !ok {
		return
	}
	target = xchange.NormalizeCurrency(target)

	source := string(listing.Currency)
	if source == "" {
		source = xchange.NormalizeCurrency("")
	}
	if strings.EqualFold(source, target) {
		return
	}

	rate, err := r.fx.GetExchangeRate(source, target)
	if err != nil {
		r.log.Warn("failed to convert listing=%s from=%s to=%s: %v", listing.ID, source, target, err)
		return
	}

	convert := func(value float64) float64 {
		return math.Round(value*rate*100) / 100
	}

	convertPtr := func(ptr *float64) {
		if ptr != nil {
			*ptr = convert(*ptr)
		}
	}

	convertCharges := func(charges *[]domain.ServiceCharge) {
		if charges == nil {
			return
		}
		for i := range *charges {
			(*charges)[i].Amount = convert((*charges)[i].Amount)
		}
	}

	if detail := listing.ShortletDetails; detail != nil {
		detail.NightlyRate = convert(detail.NightlyRate)
		convertPtr(detail.CautionFee)
		convertPtr(detail.CleaningFee)
		convertPtr(detail.ServiceFee)
		convertPtr(detail.ExtraGuestFee)
	}

	if detail := listing.RentalDetails; detail != nil {
		detail.RentalPrice = convert(detail.RentalPrice)
		convertPtr(detail.AgencyFee)
		convertPtr(detail.LegalFee)
		convertPtr(detail.RegistrationFee)
		convertPtr(detail.CautionFee)
		convertPtr(detail.ServiceCharge)
		convertCharges(detail.ServiceChargeBreakdown)
	}

	if detail := listing.SaleDetails; detail != nil {
		detail.SalePrice = convert(detail.SalePrice)
		convertPtr(detail.AgencyFee)
		convertPtr(detail.LegalFee)
		convertPtr(detail.SurveyFee)
		convertPtr(detail.TitleProcessingFee)
		convertPtr(detail.DevelopmentFee)
		convertPtr(detail.OtherFees)
		convertPtr(detail.ServiceCharge)
		convertCharges(detail.ServiceChargeBreakdown)
	}

	listing.Currency = domain.CurrencyCode(target)
}

// MyIndividualListings retrieves individual listings owned by the authenticated user.
func (r *Resolver) MyIndividualListings(ctx context.Context, filter *model.ListingFilterInput, first *int, after *string) (*model.ListingConnection, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to access myIndividualListings")
		return nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		r.log.Error("Invalid user ID in myIndividualListings: %s", v.UserID)
		return nil, fmt.Errorf("invalid user ID")
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
		r.log.Error("Failed to get individual listings for user %s: %v", v.UserID, err)
		return nil, err
	}

	r.warmListingLoaders(ctx, listings)
	for i := range listings {
		if len(listings[i].Media) > 0 {
			listings[i].Media = helpers.BuildListingMediaURLs(listings[i].Media, r.cdnHost)
		}
		r.localizeListing(ctx, &listings[i])
	}

	return buildListingConnection(listings, total, offset, limit, ctx, v), nil
}

// ListingsNearPoint finds listings near a geographic point.
func (r *Resolver) ListingsNearPoint(ctx context.Context, lat float64, lng float64, radiusMeters float64, filter *model.ListingFilterInput, limit *int) ([]*model.ListingWithDistance, error) {
	r.log.Warn("listingsNearPoint not supported")
	return nil, fmt.Errorf("listingsNearPoint not supported")
}

// SearchListings performs full-text search on listings.
// SECURITY NOTE: This endpoint ONLY returns published listings. The published filter
// has been removed from the GraphQL API and is enforced at the service layer to prevent
// exposure of unpublished listings. Draft listings are additionally filtered by sanitizeListingForViewer.
func (r *Resolver) SearchListings(ctx context.Context, filter *model.ListingFilterInput, limit *int) ([]*model.ScoredListing, error) {
	searchLimit := 20
	if limit != nil && *limit > 0 {
		searchLimit = min(*limit, 50)
	}

	serviceFilter := mapListingFilterToService(filter)

	results, err := r.propertyService.SearchListings(ctx, serviceFilter, searchLimit)
	if err != nil {
		r.log.Error("failed to search listings: %v", err)
		return nil, err
	}

	v := viewer.FromContext(ctx)
	if len(results) > 0 {
		listings := make([]domain.Listing, 0, len(results))
		for _, res := range results {
			listings = append(listings, res.Listing)
		}
		r.warmListingLoaders(ctx, listings)
	}
	scored := make([]*model.ScoredListing, 0, len(results))
	for _, res := range results {
		l := res.Listing
		if len(l.Media) > 0 {
			l.Media = helpers.BuildListingMediaURLs(l.Media, r.cdnHost)
		}
		r.localizeListing(ctx, &l)
		sanitized := sanitizeListingForViewer(ctx, &l, v)
		if sanitized == nil {
			continue
		}
		scored = append(scored, &model.ScoredListing{
			Listing: sanitized,
			Score:   res.Score,
			Ranking: res.Ranking,
		})
	}

	return scored, nil
}

// SimilarListings finds similar listings using vector similarity.
func (r *Resolver) SimilarListings(ctx context.Context, listingID uuid.UUID, limit *int, minSimilarity *float64) ([]*model.ScoredListing, error) {
	searchLimit := 10
	if limit != nil && *limit > 0 {
		searchLimit = min(*limit, 50)
	}

	minSim := 0.7
	if minSimilarity != nil {
		minSim = *minSimilarity
	}

	// Call service method
	results, err := r.propertyService.FindSimilarListings(ctx, listingID, searchLimit, minSim)
	if err != nil {
		r.log.Error("failed to find similar listings for %s: %v", listingID, err)
		return nil, err
	}

	// Build media URLs and sanitize for viewer
	v := viewer.FromContext(ctx)
	if len(results) > 0 {
		listings := make([]domain.Listing, 0, len(results))
		for _, res := range results {
			listings = append(listings, res.Listing)
		}
		r.warmListingLoaders(ctx, listings)
	}
	scored := make([]*model.ScoredListing, 0, len(results))
	for _, res := range results {
		l := res.Listing
		if len(l.Media) > 0 {
			l.Media = helpers.BuildListingMediaURLs(l.Media, r.cdnHost)
		}
		r.localizeListing(ctx, &l)
		sanitized := sanitizeListingForViewer(ctx, &l, v)
		if sanitized == nil {
			continue
		}
		scored = append(scored, &model.ScoredListing{
			Listing: sanitized,
			Score:   res.Score,
			Ranking: res.Ranking,
		})
	}

	return scored, nil
}

// ===========================
// MUTATION RESOLVERS
// ===========================

// CreateListing creates a new listing with its property in a transaction.
func (r *Resolver) CreateListing(ctx context.Context, input model.CreateListingInput) (*domain.Listing, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to create listing")
		return nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		r.log.Error("Invalid user ID in createListing: %s", v.UserID)
		return nil, fmt.Errorf("invalid user ID")
	}

	if input.Property == nil {
		r.log.Warn("CreateListing called without property payload by user %s", v.UserID)
		return nil, fmt.Errorf("property payload is required")
	}

	// Determine ownerID based on ownerType
	var ownerID uuid.UUID
	if input.OwnerType == domain.OwnerBusiness {
		// For business listings, businessID must be provided
		if input.BusinessID == nil {
			r.log.Warn("CreateListing called with business ownerType but no businessID by user %s", v.UserID)
			return nil, fmt.Errorf("businessID is required when ownerType is business")
		}
		ownerID = *input.BusinessID
		r.log.Info(" Creating business listing for business %s by user %s", ownerID, v.UserID)
	} else {
		// For individual listings, use userID
		ownerID = userID
		r.log.Info(" Creating individual listing for user %s", v.UserID)
	}

	// Map inputs to domain models
	property := mapCreateListingPropertyInput(input.Property, ownerID)
	listing := mapCreateListingInput(input, ownerID)

	createdProperty, createdListing, err := r.propertyService.CreatePropertyWithListing(ctx, *property, listing)
	if err != nil {
		r.log.Error("Failed to create property with listing for user %s: %v", v.UserID, err)
		return nil, err
	}

	r.log.Info(" Property %s and listing %s created successfully by user %s", createdProperty.ID, createdListing.ID, v.UserID)
	return sanitizeListingForViewer(ctx, createdListing, v), nil
}

// UpdateListing updates an existing listing.
func (r *Resolver) UpdateListing(ctx context.Context, id uuid.UUID, input model.UpdateListingInput) (*domain.Listing, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to update listing %s", id)
		return nil, fmt.Errorf("unauthenticated")
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		r.log.Error("Invalid user ID in updateListing: %s", v.UserID)
		return nil, fmt.Errorf("invalid user ID")
	}

	propUpdates := mapUpdateListingPropertyInput(input.Property)
	listingUpdates := mapListingUpdateInput(&input)
	updated, err := r.propertyService.UpdateListingWithProperty(ctx, id, listingUpdates, propUpdates, requesterID, v.Role)
	if err != nil {
		r.log.Error("Failed to update listing %s: %v", id, err)
		return nil, err
	}

	r.log.Info(" Listing %s updated successfully by user %s", id, v.UserID)
	return sanitizeListingForViewer(ctx, updated, v), nil
}

// DeleteListing deletes a listing.
func (r *Resolver) DeleteListing(ctx context.Context, id uuid.UUID, hard *bool) (bool, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to delete listing %s", id)
		return false, fmt.Errorf("unauthenticated")
	}

	existing, err := r.propertyService.GetListingByID(ctx, id, false)
	if err != nil {
		r.log.Error("Failed to get listing %s for deletion: %v", id, err)
		return false, err
	}

	if existing.OwnerType != domain.OwnerBusiness && existing.OwnerID.String() != v.UserID && !isAdminRole(v.Role) {
		r.log.Warn("User %s attempted to delete listing %s owned by %s", v.UserID, id, existing.OwnerID)
		return false, fmt.Errorf("forbidden: not the owner")
	}

	isHard := false
	if hard != nil {
		isHard = *hard
	}

	if err := r.propertyService.DeleteListing(ctx, id, isHard); err != nil {
		r.log.Error("Failed to delete listing %s: %v", id, err)
		return false, err
	}

	r.log.Info(" Listing %s deleted (hard: %v) by user %s", id, isHard, v.UserID)
	return true, nil
}

// PublishListing publishes a listing.
func (r *Resolver) PublishListing(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to publish listing %s", id)
		return nil, fmt.Errorf("unauthenticated")
	}

	existing, err := r.propertyService.GetListingByID(ctx, id, false)
	if err != nil {
		r.log.Error("Failed to get listing %s for publishing: %v", id, err)
		return nil, err
	}

	if existing.OwnerType != domain.OwnerBusiness && existing.OwnerID.String() != v.UserID && !isAdminRole(v.Role) {
		r.log.Warn("User %s attempted to publish listing %s owned by %s", v.UserID, id, existing.OwnerID)
		return nil, fmt.Errorf("forbidden: not the owner")
	}

	if err := r.propertyService.PublishListingRequest(ctx, id); err != nil {
		r.log.Error("Failed to publish listing %s: %v", id, err)
		return nil, err
	}
	updatedListing, err := r.propertyService.GetListingByID(ctx, id, false)
	if err != nil {
		r.log.Error("Failed to get listing %s for publishing: %v", id, err)
		return nil, err
	}

	r.log.Info(" Listing %s published request successful by user %s", id, v.UserID)
	return sanitizeListingForViewer(ctx, updatedListing, v), nil
}

// UnpublishListing unpublishes a listing.
func (r *Resolver) UnpublishListing(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("Unauthenticated attempt to unpublish listing %s", id)
		return nil, fmt.Errorf("unauthenticated")
	}

	existing, err := r.propertyService.GetListingByID(ctx, id, false)
	if err != nil {
		r.log.Error("Failed to get listing %s for unpublishing: %v", id, err)
		return nil, err
	}

	if existing.OwnerType != domain.OwnerBusiness && existing.OwnerID.String() != v.UserID && !isAdminRole(v.Role) {
		r.log.Warn("User %s attempted to unpublish listing %s owned by %s", v.UserID, id, existing.OwnerID)
		return nil, fmt.Errorf("forbidden: not the owner")
	}

	r.log.Warn("UnpublishListing not supported")
	return nil, fmt.Errorf("unpublish listing not supported")
}

// ===========================
// FIELD RESOLVERS
// ===========================

// propertyForListing fetches the property linked to a listing, honoring viewer visibility.
func (r *Resolver) propertyForListing(ctx context.Context, listing *domain.Listing) (*domain.Property, error) {
	if listing == nil {
		return nil, domain.ErrPropertyNotFound
	}

	v := viewer.FromContext(ctx)
	if l := loaders.For(ctx); l != nil && l.Property != nil {
		if p, err := l.Property.Load(ctx, listing.PropertyID); err == nil {
			return sanitizePropertyForViewer(p, v), nil
		}
	}

	property, err := r.propertyService.GetPropertyByID(ctx, listing.PropertyID)
	if err != nil {
		r.log.Error("Failed to get property %s: %v", listing.PropertyID, err)
		return nil, err
	}
	return sanitizePropertyForViewer(property, v), nil
}

// Address resolves property.address via the listing relationship.
func (r *Resolver) ListingAddress(ctx context.Context, obj *domain.Listing) (string, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.Address, nil
}

func (r *Resolver) ListingState(ctx context.Context, obj *domain.Listing) (string, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.State, nil
}

func (r *Resolver) ListingPostalCode(ctx context.Context, obj *domain.Listing) (string, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.PostalCode, nil
}

func (r *Resolver) ListingCountry(ctx context.Context, obj *domain.Listing) (domain.CountryCode, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.Country, nil
}

func (r *Resolver) ListingLocation(ctx context.Context, obj *domain.Listing) (*domain.Location, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.Location, nil
}

func (r *Resolver) ListingPropertyClass(ctx context.Context, obj *domain.Listing) (domain.PropertyClass, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.PropertyClass, nil
}

func (r *Resolver) ListingPropertyType(ctx context.Context, obj *domain.Listing) (domain.PropertyType, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.PropertyType, nil
}

func (r *Resolver) ListingFurnishingType(ctx context.Context, obj *domain.Listing) (domain.FurnishingType, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.FurnishingType, nil
}

func (r *Resolver) ListingPropertyCondition(ctx context.Context, obj *domain.Listing) (domain.PropertyCondition, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return "", err
	}
	return property.PropertyCondition, nil
}

func (r *Resolver) ListingBedrooms(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.Bedrooms, nil
}

func (r *Resolver) ListingBathrooms(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.Bathrooms, nil
}

func (r *Resolver) ListingToilets(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.Toilets, nil
}

func (r *Resolver) ListingHalfBathrooms(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.HalfBathrooms, nil
}

func (r *Resolver) ListingFloors(ctx context.Context, obj *domain.Listing) (*int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.Floors, nil
}

func (r *Resolver) ListingUnits(ctx context.Context, obj *domain.Listing) (int, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return 0, err
	}
	return property.Units, nil
}

func (r *Resolver) ListingSquareMeters(ctx context.Context, obj *domain.Listing) (float64, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return 0, err
	}
	return property.SquareMeters, nil
}

func (r *Resolver) ListingFloorArea(ctx context.Context, obj *domain.Listing) (*float64, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.FloorArea, nil
}

func (r *Resolver) ListingAmenities(ctx context.Context, obj *domain.Listing) ([]domain.AmenityGroup, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
		return nil, err
	}
	return property.Amenities, nil
}

func (r *Resolver) ListingFeaturesCommercial(ctx context.Context, obj *domain.Listing) ([]domain.AmenityGroup, error) {
	property, err := r.propertyForListing(ctx, obj)
	if err != nil {
		r.log.Error("Failed to get property for listing %s: %v", obj.ID, err)
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
			r.log.Error("Failed to list media for listing %s: %v", obj.ID, err)
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

// ServiceCharges resolves service charges for RentalDetail.
func (r *Resolver) RentalDetailServiceCharges(ctx context.Context, obj *domain.RentalDetail) ([]*domain.ServiceCharge, error) {
	if obj.ServiceChargeBreakdown == nil {
		return []*domain.ServiceCharge{}, nil
	}

	charges := make([]*domain.ServiceCharge, len(*obj.ServiceChargeBreakdown))
	for i, charge := range *obj.ServiceChargeBreakdown {
		c := charge
		charges[i] = &c
	}
	return charges, nil
}

// ServiceCharges resolves service charges for SaleDetail.
func (r *Resolver) SaleDetailServiceCharges(ctx context.Context, obj *domain.SaleDetail) ([]*domain.ServiceCharge, error) {
	if obj.ServiceChargeBreakdown == nil {
		return []*domain.ServiceCharge{}, nil
	}

	charges := make([]*domain.ServiceCharge, len(*obj.ServiceChargeBreakdown))
	for i, charge := range *obj.ServiceChargeBreakdown {
		c := charge
		charges[i] = &c
	}
	return charges, nil
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
			r.log.Error("Failed to load profile for owner %s: %v", obj.OwnerID, err)
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
				r.log.Warn("failed to preload properties for listings: %v", err)
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
				r.log.Warn("failed to preload profiles for listings: %v", err)
			}
		}
	}
}
