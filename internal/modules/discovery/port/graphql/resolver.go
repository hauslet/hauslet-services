package graphql

import (
	"context"
	"log/slog"
	"strings"

	"hauslet/internal/modules/discovery/domain"
	"hauslet/internal/modules/discovery/service"
	graphmodel "hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles GraphQL queries for the discovery module
type Resolver struct {
	discoverySvc service.DiscoveryService
	log          *slog.Logger
}

// NewResolver creates a new discovery GraphQL resolver
func NewResolver(discoverySvc service.DiscoveryService, log *slog.Logger) *Resolver {
	return &Resolver{
		discoverySvc: discoverySvc,
		log:          log,
	}
}

// Discover performs semantic search with promotion-aware ranking
func (r *Resolver) Discover(
	ctx context.Context,
	filter graphmodel.DiscoverySearchFilterInput,
	options *graphmodel.SearchOptionsInput,
) (*domain.SearchResult, error) {
	// Map GraphQL input to service types
	serviceFilter := mapToServiceFilter(filter)
	serviceOptions := mapToServiceOptions(options)

	// Execute search
	result, err := r.discoverySvc.SearchListings(ctx, serviceFilter, serviceOptions)
	if err != nil {
		r.log.Error("discover query failed", "error", err)
		return nil, err
	}

	return result, nil
}

// HomeFeed returns the curated home feed
func (r *Resolver) HomeFeed(
	ctx context.Context,
	options *graphmodel.FeedOptionsInput,
) ([]*domain.HomeFeedSection, error) {
	// Get user ID from context if authenticated
	var userID *uuid.UUID
	if uid, err := viewer.GetUserIDFromContext(ctx); err == nil {
		userID = &uid
	}

	serviceOptions := mapToFeedOptions(options)
	sections, err := r.discoverySvc.GetHomeFeed(ctx, userID, serviceOptions)
	if err != nil {
		r.log.Error("home feed query failed", "error", err)
		return nil, err
	}

	return toHomeFeedSectionPointers(sections), nil
}

// FeaturedListings returns currently featured listings
func (r *Resolver) FeaturedListings(
	ctx context.Context,
	limit *int,
) ([]*domain.RankedListing, error) {
	limitValue := 10 // default
	if limit != nil {
		limitValue = *limit
	}

	listings, err := r.discoverySvc.GetFeaturedListings(ctx, limitValue)
	if err != nil {
		r.log.Error("featured listings query failed", "error", err)
		return nil, err
	}

	return toRankedListingPointers(listings), nil
}

// DiscoverSimilar finds listings similar to the given listing with promotion-aware ranking
func (r *Resolver) DiscoverSimilar(
	ctx context.Context,
	listingID uuid.UUID,
	limit *int,
) ([]*domain.RankedListing, error) {
	limitValue := 10 // default
	if limit != nil {
		limitValue = *limit
	}

	listings, err := r.discoverySvc.FindSimilarListings(ctx, listingID, limitValue)
	if err != nil {
		r.log.Error("similar listings query failed", "error", err, "listingID", listingID)
		return nil, err
	}

	return toRankedListingPointers(listings), nil
}

// ===== MAPPING FUNCTIONS =====

// mapToServiceFilter converts GraphQL filter to service filter
func mapToServiceFilter(input graphmodel.DiscoverySearchFilterInput) service.SearchFilter {
	filter := service.SearchFilter{
		Query: input.Query,
	}

	if input.Location != nil {
		filter.Location = &service.LocationFilter{
			Latitude:  input.Location.Lat,
			Longitude: input.Location.Lng,
			RadiusKm:  input.Location.RadiusKm,
		}
	}

	if input.PriceRange != nil {
		filter.PriceRange = &service.PriceRangeFilter{
			Min:      intPtrToInt64Ptr(input.PriceRange.Min),
			Max:      intPtrToInt64Ptr(input.PriceRange.Max),
			Currency: input.PriceRange.Currency,
		}
	}

	if len(input.PropertyTypes) > 0 {
		filter.PropertyTypes = make([]string, len(input.PropertyTypes))
		for i, pt := range input.PropertyTypes {
			filter.PropertyTypes[i] = string(pt)
		}
	}

	if input.Bedrooms != nil {
		filter.Bedrooms = &service.IntRangeFilter{
			Min: input.Bedrooms.Min,
			Max: input.Bedrooms.Max,
		}
	}

	if input.Bathrooms != nil {
		filter.Bathrooms = &service.IntRangeFilter{
			Min: input.Bathrooms.Min,
			Max: input.Bathrooms.Max,
		}
	}

	if len(input.ListingTypes) > 0 {
		filter.ListingTypes = make([]string, len(input.ListingTypes))
		for i, lt := range input.ListingTypes {
			filter.ListingTypes[i] = string(lt)
		}
	}

	filter.City = input.City
	filter.State = input.State
	filter.Country = input.Country
	filter.Amenities = input.Amenities

	return filter
}

// mapToServiceOptions converts GraphQL options to service options
func mapToServiceOptions(input *graphmodel.SearchOptionsInput) service.SearchOptions {
	options := service.SearchOptions{
		Limit:           20, // default
		IncludePromoted: true,
	}

	if input != nil {
		if input.Limit != nil {
			options.Limit = *input.Limit
		}
		if input.IncludePromoted != nil {
			options.IncludePromoted = *input.IncludePromoted
		}
		if input.RankingConfig != nil {
			defaults := domain.DefaultRankingConfig()
			options.RankingConfig = &domain.RankingConfig{
				SemanticWeight:        getFloatOrDefault(input.RankingConfig.SemanticWeight, defaults.SemanticWeight),
				PromotionWeight:       getFloatOrDefault(input.RankingConfig.PromotionWeight, defaults.PromotionWeight),
				RecencyWeight:         getFloatOrDefault(input.RankingConfig.RecencyWeight, defaults.RecencyWeight),
				LocationWeight:        getFloatOrDefault(input.RankingConfig.LocationWeight, defaults.LocationWeight),
				TextMatchWeight:       getFloatOrDefault(input.RankingConfig.TextMatchWeight, defaults.TextMatchWeight),
				PersonalizationWeight: 0.0,
			}
		}
	}

	return options
}

// mapToFeedOptions converts GraphQL feed options to service options
func mapToFeedOptions(input *graphmodel.FeedOptionsInput) service.FeedOptions {
	options := service.FeedOptions{
		Limit: 10, // default
	}

	if input != nil {
		if input.Location != nil {
			options.Location = &service.LocationFilter{
				Latitude:  input.Location.Lat,
				Longitude: input.Location.Lng,
				RadiusKm:  input.Location.RadiusKm,
			}
		}
		options.City = sanitizeOptionalString(input.City)
		options.State = sanitizeOptionalString(input.State)
		if input.Limit != nil {
			options.Limit = *input.Limit
		}
		if len(input.SectionsToInclude) > 0 {
			options.SectionsToInclude = input.SectionsToInclude
		}
	}

	return options
}

func sanitizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// getFloatOrDefault returns the float value if not nil, otherwise returns the default
func getFloatOrDefault(value *float64, defaultValue float64) float64 {
	if value != nil {
		return *value
	}
	return defaultValue
}

func intPtrToInt64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func toHomeFeedSectionPointers(sections []domain.HomeFeedSection) []*domain.HomeFeedSection {
	if len(sections) == 0 {
		return []*domain.HomeFeedSection{}
	}
	pointers := make([]*domain.HomeFeedSection, len(sections))
	for i := range sections {
		pointers[i] = &sections[i]
	}
	return pointers
}

func toRankedListingPointers(listings []domain.RankedListing) []*domain.RankedListing {
	if len(listings) == 0 {
		return []*domain.RankedListing{}
	}
	pointers := make([]*domain.RankedListing, len(listings))
	for i := range listings {
		pointers[i] = &listings[i]
	}
	return pointers
}
