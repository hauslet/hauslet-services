package hooks

import (
	"context"
	"strings"

	discoveryservice "hauslet/internal/modules/discovery/service"
	propertydomain "hauslet/internal/modules/property/domain"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// PropertyDiscoveryAdapter wraps PropertyService to implement PropertyDiscoveryHooks
type PropertyDiscoveryAdapter struct {
	propertySvc propertyservice.PropertyService
}

// NewPropertyDiscoveryAdapter creates a new PropertyDiscoveryAdapter
func NewPropertyDiscoveryAdapter(propertySvc propertyservice.PropertyService) discoveryservice.PropertyDiscoveryHooks {
	return &PropertyDiscoveryAdapter{
		propertySvc: propertySvc,
	}
}

// GetListingByID retrieves a single listing by ID
func (a *PropertyDiscoveryAdapter) GetListingByID(ctx context.Context, id uuid.UUID) (*propertydomain.Listing, error) {
	return a.propertySvc.GetListingByID(ctx, id, false)
}

// GetListingsByIDs batch fetches listings by IDs
func (a *PropertyDiscoveryAdapter) GetListingsByIDs(ctx context.Context, ids []uuid.UUID) ([]propertydomain.Listing, error) {
	return a.propertySvc.GetListingsByIDs(ctx, ids, true)
}

// SearchListingsWithEmbedding performs semantic search using embeddings
func (a *PropertyDiscoveryAdapter) SearchListingsWithEmbedding(ctx context.Context, filter discoveryservice.SearchFilter, limit int, excludedIDs []uuid.UUID) ([]propertydomain.ScoredListing, error) {
	// Convert discovery filter to property filter
	propertyFilter := a.mapToPropertyFilter(filter)

	if len(excludedIDs) > 0 {
		propertyFilter.ExcludedListingIDs = excludedIDs
	}

	return a.propertySvc.SearchListings(ctx, propertyFilter, limit)
}

// GetRecentListings fetches recently published listings
func (a *PropertyDiscoveryAdapter) GetRecentListings(ctx context.Context, limit int) ([]propertydomain.Listing, error) {
	// Create filter for active, published listings
	filter := propertyservice.ListingFilter{
		Statuses: []propertydomain.ListingStatus{propertydomain.StatusActive},
	}
	published := true
	filter.Published = &published
	filter.ExcludeSuspended = true

	// Use ListListings with limit
	listings, _, err := a.propertySvc.ListListings(ctx, filter, propertyservice.Pagination{
		Limit:  limit,
		Offset: 0,
	})
	return listings, err
}

// FindSimilarListings finds listings similar to the given listing
func (a *PropertyDiscoveryAdapter) FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int) ([]propertydomain.ScoredListing, error) {
	// Use minimum similarity of 0.7 (70% similar)
	return a.propertySvc.FindSimilarListings(ctx, listingID, limit, 0.7)
}

// mapToPropertyFilter converts discovery SearchFilter to property ListingFilter
func (a *PropertyDiscoveryAdapter) mapToPropertyFilter(filter discoveryservice.SearchFilter) propertyservice.ListingFilter {
	propertyFilter := propertyservice.ListingFilter{
		Statuses: []propertydomain.ListingStatus{propertydomain.StatusActive},
	}
	propertyFilter.ExcludeSuspended = true

	// Set published to true
	published := true
	propertyFilter.Published = &published

	// Map query (used for semantic search)
	if filter.Query != nil {
		propertyFilter.Query = filter.Query
	}

	// Map location
	if filter.Location != nil {
		propertyFilter.Latitude = &filter.Location.Latitude
		propertyFilter.Longitude = &filter.Location.Longitude
		// Convert km to meters
		radiusMeters := filter.Location.RadiusKm * 1000
		propertyFilter.RadiusMeters = &radiusMeters
	}

	// Map price range (Note: property module uses type-specific price filters)
	if filter.PriceRange != nil {
		minPrice, maxPrice := normalizePriceRange(filter.PriceRange.Min, filter.PriceRange.Max)
		propertyFilter.MinPrice = minPrice
		propertyFilter.MaxPrice = maxPrice

		if currency := normalizeCurrency(filter.PriceRange.Currency); currency != nil {
			propertyFilter.Currency = currency
		}
	}

	// Map property types
	if len(filter.PropertyTypes) > 0 {
		propertyTypes := make([]propertydomain.PropertyType, len(filter.PropertyTypes))
		for i, pt := range filter.PropertyTypes {
			propertyTypes[i] = propertydomain.PropertyType(pt)
		}
		propertyFilter.PropertyTypes = propertyTypes
	}

	// Map bedrooms
	if filter.Bedrooms != nil {
		propertyFilter.MinBedrooms = filter.Bedrooms.Min
		propertyFilter.MaxBedrooms = filter.Bedrooms.Max
	}

	// Map bathrooms
	if filter.Bathrooms != nil {
		propertyFilter.MinBathrooms = filter.Bathrooms.Min
		propertyFilter.MaxBathrooms = filter.Bathrooms.Max
	}

	// Map listing types
	if len(filter.ListingTypes) > 0 {
		listingTypes := make([]propertydomain.ListingType, len(filter.ListingTypes))
		for i, lt := range filter.ListingTypes {
			listingTypes[i] = propertydomain.ListingType(lt)
		}
		propertyFilter.ListingTypes = listingTypes
	}

	// Map location filters
	if filter.City != nil {
		propertyFilter.City = filter.City
	}
	if filter.State != nil {
		propertyFilter.State = filter.State
	}
	if filter.Country != nil {
		countryCode := propertydomain.CountryCode(*filter.Country)
		propertyFilter.Country = &countryCode
	}

	// Map amenities (via PropertyExtension)
	if len(filter.Amenities) > 0 {
		if propertyFilter.PropertyExtension == nil {
			propertyFilter.PropertyExtension = &propertyservice.PropertyFilterExtension{}
		}
		propertyFilter.PropertyExtension.Amenities = filter.Amenities
	}

	// Map Guest Count (mapped to Shortlet MinMaxGuests for now)
	// Note: This assumes guest count filter is primary for shortlets.
	// For general capacity, we might want to check bedrooms too, but MinMaxGuests is specific to shortlets.
	if filter.GuestCount != nil {
		if propertyFilter.ShortletFilter == nil {
			propertyFilter.ShortletFilter = &propertyservice.ShortletFilter{}
		}
		// We want listings where MaxGuests >= RequestedGuestCount
		minMax := *filter.GuestCount
		propertyFilter.ShortletFilter.MinMaxGuests = &minMax
	}

	// Map Furnishing
	if len(filter.Furnishing) > 0 {
		propertyFilter.Furnishings = make([]propertydomain.FurnishingType, len(filter.Furnishing))
		for i, v := range filter.Furnishing {
			propertyFilter.Furnishings[i] = propertydomain.FurnishingType(v)
		}
	}

	// Map Accommodation Types (Specific to shortlets)
	if len(filter.AccommodationTypes) > 0 {
		if propertyFilter.ShortletFilter == nil {
			propertyFilter.ShortletFilter = &propertyservice.ShortletFilter{}
		}
		propertyFilter.ShortletFilter.AccommodationTypes = make([]propertydomain.AccommodationType, len(filter.AccommodationTypes))
		for i, v := range filter.AccommodationTypes {
			propertyFilter.ShortletFilter.AccommodationTypes[i] = propertydomain.AccommodationType(v)
		}
	}

	// Map IsVerified
	if filter.IsVerified != nil {
		propertyFilter.IsVerified = filter.IsVerified
	}

	return propertyFilter
}

func normalizePriceRange(minCents *int64, maxCents *int64) (*float64, *float64) {
	var minPrice *float64
	var maxPrice *float64

	if minCents != nil && *minCents >= 0 {
		value := float64(*minCents) / 100
		minPrice = &value
	}
	if maxCents != nil && *maxCents >= 0 {
		value := float64(*maxCents) / 100
		maxPrice = &value
	}

	if minPrice != nil && maxPrice != nil && *minPrice > *maxPrice {
		minPrice, maxPrice = maxPrice, minPrice
	}

	return minPrice, maxPrice
}

func normalizeCurrency(code string) *propertydomain.CurrencyCode {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return nil
	}

	currency := propertydomain.CurrencyCode(strings.ToUpper(trimmed))
	switch currency {
	case propertydomain.CurrencyNGN,
		propertydomain.CurrencyGHS,
		propertydomain.CurrencyUSD,
		propertydomain.CurrencyEUR:
		return &currency
	default:
		return nil
	}
}
