package graphql

import (
	"context"
	"log/slog"

	"hauslet/internal/modules/discovery/domain"
	"hauslet/internal/modules/discovery/service"
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
	filter DiscoverySearchFilterInput,
	options *SearchOptionsInput,
) (*SearchResult, error) {
	// Map GraphQL input to service types
	serviceFilter := mapToServiceFilter(filter)
	serviceOptions := mapToServiceOptions(options)

	// Execute search
	result, err := r.discoverySvc.SearchListings(ctx, serviceFilter, serviceOptions)
	if err != nil {
		r.log.Error("discover query failed", "error", err)
		return nil, err
	}

	// Map result to GraphQL type
	return mapToGraphQLSearchResult(result), nil
}

// HomeFeed returns the curated home feed
func (r *Resolver) HomeFeed(
	ctx context.Context,
	options *FeedOptionsInput,
) ([]HomeFeedSection, error) {
	// Get user ID from context if authenticated
	var userID *uuid.UUID
	if v := viewer.FromContext(ctx); v != nil && v.UserID != "" {
		parsedID, err := uuid.Parse(v.UserID)
		if err != nil {
			if r.log != nil {
				r.log.Warn("invalid viewer user ID", "user_id", v.UserID, "error", err)
			}
		} else {
			userID = &parsedID
		}
	}

	serviceOptions := mapToFeedOptions(options)

	sections, err := r.discoverySvc.GetHomeFeed(ctx, userID, serviceOptions)
	if err != nil {
		r.log.Error("home feed query failed", "error", err)
		return nil, err
	}

	return mapToGraphQLHomeFeed(sections), nil
}

// FeaturedListings returns currently featured listings
func (r *Resolver) FeaturedListings(
	ctx context.Context,
	limit *int,
) ([]RankedListing, error) {
	limitValue := 10 // default
	if limit != nil {
		limitValue = *limit
	}

	listings, err := r.discoverySvc.GetFeaturedListings(ctx, limitValue)
	if err != nil {
		r.log.Error("featured listings query failed", "error", err)
		return nil, err
	}

	return mapToGraphQLRankedListings(listings), nil
}

// DiscoverSimilar finds listings similar to the given listing with promotion-aware ranking
func (r *Resolver) DiscoverSimilar(
	ctx context.Context,
	listingID uuid.UUID,
	limit *int,
) ([]RankedListing, error) {
	limitValue := 10 // default
	if limit != nil {
		limitValue = *limit
	}

	listings, err := r.discoverySvc.FindSimilarListings(ctx, listingID, limitValue)
	if err != nil {
		r.log.Error("similar listings query failed", "error", err, "listingID", listingID)
		return nil, err
	}

	return mapToGraphQLRankedListings(listings), nil
}

// ===== MAPPING FUNCTIONS =====

// mapToServiceFilter converts GraphQL filter to service filter
func mapToServiceFilter(input DiscoverySearchFilterInput) service.SearchFilter {
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
			Min:      input.PriceRange.Min,
			Max:      input.PriceRange.Max,
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
func mapToServiceOptions(input *SearchOptionsInput) service.SearchOptions {
	options := service.SearchOptions{
		Limit:          20, // default
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
			options.RankingConfig = &domain.RankingConfig{
				SemanticWeight:        getFloatOrDefault(input.RankingConfig.SemanticWeight, 0.4),
				PromotionWeight:       getFloatOrDefault(input.RankingConfig.PromotionWeight, 0.3),
				RecencyWeight:         getFloatOrDefault(input.RankingConfig.RecencyWeight, 0.2),
				LocationWeight:        getFloatOrDefault(input.RankingConfig.LocationWeight, 0.1),
				PersonalizationWeight: 0.0,
			}
		}
	}

	return options
}

// mapToFeedOptions converts GraphQL feed options to service options
func mapToFeedOptions(input *FeedOptionsInput) service.FeedOptions {
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
		if input.Limit != nil {
			options.Limit = *input.Limit
		}
		if len(input.SectionsToInclude) > 0 {
			options.SectionsToInclude = make([]domain.FeedSectionType, len(input.SectionsToInclude))
			for i, st := range input.SectionsToInclude {
				options.SectionsToInclude[i] = domain.FeedSectionType(st)
			}
		}
	}

	return options
}

// mapToGraphQLSearchResult maps domain SearchResult to GraphQL type
func mapToGraphQLSearchResult(result *domain.SearchResult) *SearchResult {
	processingTime := int(result.ProcessingTime)
	return &SearchResult{
		Listings:       mapToGraphQLRankedListings(result.Listings),
		TotalCount:     result.TotalCount,
		SearchID:       result.SearchID,
		ProcessingTime: &processingTime,
	}
}

// mapToGraphQLHomeFeed maps domain HomeFeedSection slice to GraphQL type
func mapToGraphQLHomeFeed(sections []domain.HomeFeedSection) []HomeFeedSection {
	graphqlSections := make([]HomeFeedSection, len(sections))
	for i, section := range sections {
		graphqlSections[i] = HomeFeedSection{
			SectionType: FeedSectionType(section.SectionType),
			Title:       section.Title,
			Listings:    mapToGraphQLRankedListings(section.Listings),
			TotalCount:  section.TotalCount,
		}
	}
	return graphqlSections
}

// mapToGraphQLRankedListings maps domain RankedListing slice to GraphQL type
func mapToGraphQLRankedListings(listings []domain.RankedListing) []RankedListing {
	graphqlListings := make([]RankedListing, len(listings))
	for i, listing := range listings {
		graphqlListings[i] = mapToGraphQLRankedListing(listing)
	}
	return graphqlListings
}

// mapToGraphQLRankedListing maps a single domain RankedListing to GraphQL type
func mapToGraphQLRankedListing(listing domain.RankedListing) RankedListing {
	var promotionBoost *PromotionBoostInfo
	if listing.PromotionBoost != nil {
		promotionBoost = &PromotionBoostInfo{
			PromotionID:     listing.PromotionBoost.PromotionID,
			PromotionType:   listing.PromotionBoost.PromotionType,
			BoostMultiplier: listing.PromotionBoost.BoostMultiplier,
			ExpiresAt:       listing.PromotionBoost.ExpiresAt,
		}
	}

	return RankedListing{
		Listing: &listing.Listing,
		Score: RankingScore{
			FinalScore:        listing.Score.FinalScore,
			SemanticScore:     listing.Score.SemanticScore,
			PromotionBoost:    listing.Score.PromotionBoost,
			RecencyScore:      listing.Score.RecencyScore,
			LocationScore:     listing.Score.LocationScore,
			PersonalizedScore: listing.Score.PersonalizedScore,
		},
		Ranking:        listing.Ranking,
		PromotionBoost: promotionBoost,
	}
}

// getFloatOrDefault returns the float value if not nil, otherwise returns the default
func getFloatOrDefault(value *float64, defaultValue float64) float64 {
	if value != nil {
		return *value
	}
	return defaultValue
}

// ===== GraphQL TYPE DEFINITIONS (will be generated by gqlgen) =====

// These types will be generated by gqlgen based on the schema

type DiscoverySearchFilterInput struct {
	Query         *string
	Location      *LocationFilterInput
	PriceRange    *PriceRangeFilterInput
	PropertyTypes []string
	Bedrooms      *IntRangeFilterInput
	Bathrooms     *IntRangeFilterInput
	ListingTypes  []string
	City          *string
	State         *string
	Country       *string
	Amenities     []string
}

type LocationFilterInput struct {
	Lat      float64
	Lng      float64
	RadiusKm float64
}

type PriceRangeFilterInput struct {
	Min      *int64
	Max      *int64
	Currency string
}

type IntRangeFilterInput struct {
	Min *int
	Max *int
}

type SearchOptionsInput struct {
	Limit          *int
	IncludePromoted *bool
	RankingConfig  *RankingConfigInput
}

type RankingConfigInput struct {
	SemanticWeight  *float64
	PromotionWeight *float64
	RecencyWeight   *float64
	LocationWeight  *float64
}

type FeedOptionsInput struct {
	Location          *LocationFilterInput
	Limit             *int
	SectionsToInclude []FeedSectionType
}

type SearchResult struct {
	Listings       []RankedListing
	TotalCount     int
	SearchID       uuid.UUID
	ProcessingTime *int
}

type HomeFeedSection struct {
	SectionType FeedSectionType
	Title       string
	Listings    []RankedListing
	TotalCount  int
}

type RankedListing struct {
	Listing        interface{} // Will be *propertydomain.Listing
	Score          RankingScore
	Ranking        int
	PromotionBoost *PromotionBoostInfo
}

type RankingScore struct {
	FinalScore        float64
	SemanticScore     *float64
	PromotionBoost    float64
	RecencyScore      float64
	LocationScore     *float64
	PersonalizedScore *float64
}

type PromotionBoostInfo struct {
	PromotionID     uuid.UUID
	PromotionType   string
	BoostMultiplier float64
	ExpiresAt       interface{} // time.Time
}

type FeedSectionType string

const (
	FeedSectionTypeFeatured    FeedSectionType = "featured"
	FeedSectionTypePremium     FeedSectionType = "premium"
	FeedSectionTypeRecent      FeedSectionType = "recent"
	FeedSectionTypeRecommended FeedSectionType = "recommended"
	FeedSectionTypeNearYou     FeedSectionType = "near_you"
)
