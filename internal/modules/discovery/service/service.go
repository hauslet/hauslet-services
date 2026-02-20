package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hauslet/internal/modules/discovery/domain"
	"hauslet/internal/modules/discovery/repository"
	platformredis "hauslet/internal/platform/redis"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const maxSearchLimit = 100

// ServiceImpl implements the DiscoveryService interface
type ServiceImpl struct {
	repo           repository.DiscoveryRepository
	propertyHooks  PropertyDiscoveryHooks
	promotionHooks PromotionDiscoveryHooks
	calendarHooks  CalendarDiscoveryHooks
	bookingHooks   BookingDiscoveryHooks
	cache          platformredis.RedisClient
	rankingConfig  domain.RankingConfig
	homeFeedGroup  singleflight.Group
	previewGroup   singleflight.Group
	log            *slog.Logger
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(
	repo repository.DiscoveryRepository,
	propertyHooks PropertyDiscoveryHooks,
	promotionHooks PromotionDiscoveryHooks,
	calendarHooks CalendarDiscoveryHooks,
	cache platformredis.RedisClient,
	log *slog.Logger,
) *ServiceImpl {
	return &ServiceImpl{
		repo:           repo,
		propertyHooks:  propertyHooks,
		promotionHooks: promotionHooks,
		calendarHooks:  calendarHooks,
		bookingHooks:   nil,
		cache:          cache,
		rankingConfig:  domain.DefaultRankingConfig(),
		log:            log,
	}
}

// SetBookingHooks attaches optional booking hooks used for shortlet preview enrichment.
func (s *ServiceImpl) SetBookingHooks(bookingHooks BookingDiscoveryHooks) {
	s.bookingHooks = bookingHooks
}

// SearchListings performs semantic search with promotion-aware ranking
func (s *ServiceImpl) SearchListings(ctx context.Context, filter SearchFilter, options SearchOptions) (*domain.SearchResult, error) {
	startTime := time.Now()
	searchID := uuid.New()

	// Apply default options
	if options.Limit == 0 {
		options.Limit = 20 // Default limit
	}
	if options.Limit > maxSearchLimit {
		options.Limit = maxSearchLimit
	}

	// Use default ranking config if not provided
	rankingConfig := &s.rankingConfig
	if options.RankingConfig != nil {
		rankingConfig = options.RankingConfig
	}

	var excludedIDs []uuid.UUID

	// 0. If date range is provided, find busy listings to exclude
	if filter.CheckIn != nil && filter.CheckOut != nil && s.calendarHooks != nil {
		var err error
		excludedIDs, err = s.calendarHooks.GetUnavailableListingIDs(ctx, *filter.CheckIn, *filter.CheckOut)
		if err != nil {
			s.log.Error("failed to get unavailable listing IDs", "error", err)
			// User explicitly provided dates — returning potentially unavailable
			// listings is worse UX than an error.
			return nil, fmt.Errorf("failed to check listing availability: %w", err)
		}
	}

	// 1. Get base results from property semantic search
	// Fetch 2x the limit to ensure we have enough results after ranking
	propertyResults, err := s.propertyHooks.SearchListingsWithEmbedding(ctx, filter, options.Limit*2, excludedIDs)
	if err != nil {
		s.log.Error("failed to search listings", "error", err)
		return nil, err
	}

	// If no results, return early
	if len(propertyResults) == 0 {
		return &domain.SearchResult{
			Listings:       []domain.RankedListing{},
			TotalCount:     0,
			SearchID:       searchID,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// 2. Get promotions for listings (if enabled)
	var promotions map[uuid.UUID]*PromotionInfo
	if options.IncludePromoted {
		listingIDs := extractListingIDsFromScored(propertyResults)
		promotions, err = s.promotionHooks.GetActivePromotionForListings(ctx, listingIDs)
		if err != nil {
			s.log.Warn("failed to fetch promotions, continuing without", "error", err)
			promotions = make(map[uuid.UUID]*PromotionInfo)
		}
	} else {
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	// 3. Calculate rankings
	rankedListings := s.rankListings(propertyResults, promotions, *rankingConfig, filter.Location)

	// 4. Apply limit after ranking
	// TotalCount reflects the number of candidate results found before truncation
	totalCount := len(propertyResults)
	if len(rankedListings) > options.Limit {
		rankedListings = rankedListings[:options.Limit]
	}

	return &domain.SearchResult{
		Listings:       rankedListings,
		TotalCount:     totalCount,
		SearchID:       searchID,
		ProcessingTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// GetFeaturedListings returns currently featured (promoted) listings
func (s *ServiceImpl) GetFeaturedListings(ctx context.Context, limit int) ([]domain.RankedListing, error) {
	if limit == 0 {
		limit = 10 // Default limit
	}

	// 1. Get featured listing IDs from promotions
	featuredIDs, err := s.promotionHooks.GetFeaturedListings(ctx, limit)
	if err != nil {
		s.log.Error("failed to get featured listing IDs", "error", err)
		return nil, err
	}

	if len(featuredIDs) == 0 {
		return []domain.RankedListing{}, nil
	}

	// 2. Fetch the actual listings
	listings, err := s.propertyHooks.GetListingsByIDs(ctx, featuredIDs)
	if err != nil {
		s.log.Error("failed to fetch featured listings", "error", err)
		return nil, err
	}

	// 3. Get promotion info for these listings
	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, featuredIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info, continuing without", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	// 4. Convert to scored listings (with default score of 1.0 for semantic)
	scoredListings := convertToScoredListings(listings, 1.0)

	// 5. Rank the listings
	rankedListings := s.rankListings(scoredListings, promotions, s.rankingConfig, nil)

	return rankedListings, nil
}

// FindSimilarListings finds listings similar to the given listing
func (s *ServiceImpl) FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int) ([]domain.RankedListing, error) {
	if limit == 0 {
		limit = 10 // Default limit
	}

	// 1. Get similar listings from property service
	similarListings, err := s.propertyHooks.FindSimilarListings(ctx, listingID, limit*2)
	if err != nil {
		s.log.Error("failed to find similar listings", "error", err)
		return nil, err
	}

	if len(similarListings) == 0 {
		return []domain.RankedListing{}, nil
	}

	// 2. Get promotions for similar listings
	listingIDs := extractListingIDsFromScored(similarListings)
	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, listingIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotions for similar listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	// 3. Rank the similar listings
	rankedListings := s.rankListings(similarListings, promotions, s.rankingConfig, nil)

	// 4. Apply limit
	if len(rankedListings) > limit {
		rankedListings = rankedListings[:limit]
	}

	return rankedListings, nil
}
