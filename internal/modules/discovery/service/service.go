package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"hauslet/internal/modules/discovery/domain"
	"hauslet/internal/modules/discovery/repository"
	platformredis "hauslet/internal/platform/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	analyticsHooks AnalyticsDiscoveryHooks
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

// SetAnalyticsHooks attaches optional analytics hooks for trending/top-rated sections.
func (s *ServiceImpl) SetAnalyticsHooks(analyticsHooks AnalyticsDiscoveryHooks) {
	s.analyticsHooks = analyticsHooks
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

// SyncDestinationsToRedis updates the autocomplete redis store
func (s *ServiceImpl) SyncDestinationsToRedis(ctx context.Context) error {
	redisKey := "discovery:destinations"
	tmpKey := redisKey + ":tmp"

	locations, err := s.propertyHooks.GetDistinctLocations(ctx)
	if err != nil {
		s.log.Error("failed to get distinct locations from property module", "error", err)
		return fmt.Errorf("could not fetch locations: %w", err)
	}

	if len(locations) == 0 {
		return nil
	}

	// We clear the tmp key first to prevent stale data.
	s.cache.Del(ctx, tmpKey)

	// Push distinct locations to Redis for autocomplete search.
	// We index each destination twice:
	// 1) city key:  c|<city_lower>|City|State|Country|Display
	// 2) state key: s|<state_lower>|City|State|Country|Display
	// Search can then prioritize city matches and fall back to state matches.
	members := make([]redis.Z, 0, len(locations)*2)
	for _, loc := range locations {
		displayText := fmt.Sprintf("%s, %s", loc.City, loc.State)

		cityEntry := fmt.Sprintf("c|%s|%s|%s|%s|%s", strings.ToLower(loc.City), loc.City, loc.State, loc.Country, displayText)
		members = append(members, redis.Z{
			Score:  0,
			Member: cityEntry,
		})

		if strings.TrimSpace(loc.State) != "" {
			stateEntry := fmt.Sprintf("s|%s|%s|%s|%s|%s", strings.ToLower(loc.State), loc.City, loc.State, loc.Country, displayText)
			members = append(members, redis.Z{
				Score:  0,
				Member: stateEntry,
			})
		}
	}

	cmd := s.cache.ZAdd(ctx, tmpKey, members...)
	if cmd.Err() != nil {
		s.log.Error("failed to sync locations to redis tmp key", "error", cmd.Err())
		return fmt.Errorf("failed to save locations to redis: %w", cmd.Err())
	}

	// Atomically swap the tmp key with the live key
	if err := s.cache.Rename(ctx, tmpKey, redisKey).Err(); err != nil {
		s.log.Error("failed to swap tmp key to live discovery locations", "error", err)
		return fmt.Errorf("failed to publish locations to redis: %w", err)
	}

	// Set TTL as a safety measure
	s.cache.Expire(ctx, redisKey, 24*time.Hour)

	s.log.Info("Successfully synced distinct destinations to redis", "count", len(locations))
	return nil
}

// SearchDestinations provides autocomplete suggestions for destinations using Redis Lexicographical Search
func (s *ServiceImpl) SearchDestinations(ctx context.Context, query string, limit int) ([]*domain.Destination, error) {
	trimmedQuery := strings.TrimSpace(query)
	if len(trimmedQuery) < 3 {
		return []*domain.Destination{}, nil // Require over 2 chars as per requirement
	}
	if limit <= 0 {
		limit = 10
	}

	redisKey := "discovery:destinations" // Redis key for destinations sorted set

	searchQuery := strings.ToLower(trimmedQuery)
	if cityPart, statePart, ok := parseCityStateQuery(searchQuery); ok {
		fallbackCount := max(limit*3, limit)

		cityMatches, err := s.searchDestinationEntries(ctx, redisKey, "c", cityPart, fallbackCount)
		if err != nil {
			s.log.Error("failed to query redis city destinations for city,state query", "error", err)
			return nil, fmt.Errorf("failed to search destinations: %w", err)
		}

		stateMatches, err := s.searchDestinationEntries(ctx, redisKey, "s", statePart, fallbackCount)
		if err != nil {
			s.log.Error("failed to query redis state destinations for city,state query", "error", err)
			return nil, fmt.Errorf("failed to search destinations: %w", err)
		}

		combined := mergeCityStateDestinationMatches(cityMatches, stateMatches, limit)
		if len(combined) > 0 {
			return combined, nil
		}
	}

	cityMatches, err := s.searchDestinationEntries(ctx, redisKey, "c", searchQuery, limit)
	if err != nil {
		s.log.Error("failed to query redis city destinations", "error", err)
		return nil, fmt.Errorf("failed to search destinations: %w", err)
	}

	fallbackCount := max(limit*3, limit)
	stateMatches, err := s.searchDestinationEntries(ctx, redisKey, "s", searchQuery, fallbackCount)
	if err != nil {
		s.log.Error("failed to query redis state destinations", "error", err)
		return nil, fmt.Errorf("failed to search destinations: %w", err)
	}

	return mergeDestinationMatches(cityMatches, stateMatches, limit), nil
}

func parseCityStateQuery(query string) (cityPart string, statePart string, ok bool) {
	if !strings.Contains(query, ",") {
		return "", "", false
	}

	parts := strings.SplitN(query, ",", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	cityPart = strings.TrimSpace(parts[0])
	statePart = strings.TrimSpace(parts[1])
	if cityPart == "" || statePart == "" {
		return "", "", false
	}

	return cityPart, statePart, true
}

func (s *ServiceImpl) searchDestinationEntries(
	ctx context.Context,
	redisKey string,
	keyType string,
	query string,
	limit int,
) ([]*domain.Destination, error) {
	if limit <= 0 {
		return []*domain.Destination{}, nil
	}

	start := "[" + keyType + "|" + query
	end := "[" + keyType + "|" + query + "\xff"

	redisOpt := redis.ZRangeArgs{
		Key:    redisKey,
		Start:  start,
		Stop:   end,
		ByLex:  true,
		Offset: 0,
		Count:  int64(limit),
	}

	resultCmd := s.cache.ZRangeArgs(ctx, redisOpt)
	if err := resultCmd.Err(); err != nil && err != redis.Nil {
		return nil, err
	}

	results := resultCmd.Val()
	destinations := make([]*domain.Destination, 0, len(results))
	for _, res := range results {
		if dest, ok := parseDestinationEntry(res); ok {
			destinations = append(destinations, dest)
		}
	}

	return destinations, nil
}

func parseDestinationEntry(entry string) (*domain.Destination, bool) {
	parts := strings.Split(entry, "|")

	// Format: keyType|search_key|City|State|Country|DisplayText
	if len(parts) >= 6 {
		city := parts[2]
		state := parts[3]
		country := parts[4]
		displayText := parts[5]
		return &domain.Destination{
			ID:          city + "|" + state + "|" + country,
			City:        city,
			State:       state,
			Country:     country,
			DisplayText: displayText,
		}, true
	}

	return nil, false
}

func mergeDestinationMatches(
	cityMatches []*domain.Destination,
	stateMatches []*domain.Destination,
	limit int,
) []*domain.Destination {
	if limit <= 0 {
		return []*domain.Destination{}
	}

	merged := make([]*domain.Destination, 0, limit)
	seen := make(map[string]struct{}, len(cityMatches)+len(stateMatches))

	appendUnique := func(items []*domain.Destination) {
		for _, item := range items {
			if item == nil {
				continue
			}
			if len(merged) >= limit {
				return
			}
			if _, exists := seen[item.ID]; exists {
				continue
			}
			seen[item.ID] = struct{}{}
			merged = append(merged, item)
		}
	}

	appendUnique(cityMatches)
	appendUnique(stateMatches)

	return merged
}

func mergeCityStateDestinationMatches(
	cityMatches []*domain.Destination,
	stateMatches []*domain.Destination,
	limit int,
) []*domain.Destination {
	if limit <= 0 {
		return []*domain.Destination{}
	}

	stateByID := make(map[string]*domain.Destination, len(stateMatches))
	for _, item := range stateMatches {
		if item == nil {
			continue
		}
		stateByID[item.ID] = item
	}

	both := make([]*domain.Destination, 0, len(cityMatches))
	cityOnly := make([]*domain.Destination, 0, len(cityMatches))
	for _, item := range cityMatches {
		if item == nil {
			continue
		}
		if _, ok := stateByID[item.ID]; ok {
			both = append(both, item)
			delete(stateByID, item.ID)
			continue
		}
		cityOnly = append(cityOnly, item)
	}

	stateOnly := make([]*domain.Destination, 0, len(stateByID))
	for _, item := range stateMatches {
		if item == nil {
			continue
		}
		if _, ok := stateByID[item.ID]; ok {
			stateOnly = append(stateOnly, item)
		}
	}

	firstPass := mergeDestinationMatches(both, cityOnly, limit)
	if len(firstPass) >= limit {
		return firstPass
	}

	return mergeDestinationMatches(firstPass, stateOnly, limit)
}
