package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hauslet/internal/modules/discovery/domain"
	propertydomain "hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const defaultHomeSectionLimit = 10
const maxSectionLimit = 50
const maxShortletPreviewConcurrency = 5
const minSectionResults = 9 // Minimum results to consider a section "populated enough" (fills largest screen slide)

const (
	defaultShortletPreviewGuests    = 2
	defaultShortletPreviewNights    = 2
	shortletPreviewLookAheadDays    = 45
	shortletPreviewPayloadFieldName = "shortletPreview"
)

// GetHomeFeed returns a curated home feed with multiple sections.
// Sections are built concurrently for lower latency.
func (s *ServiceImpl) GetHomeFeed(ctx context.Context, userID *uuid.UUID, options FeedOptions) ([]domain.HomeFeedSection, error) {
	options = normalizeFeedOptions(options)

	type indexedSection struct {
		order   int
		section domain.HomeFeedSection
	}

	var (
		mu      sync.Mutex
		results []indexedSection
	)

	g, gCtx := errgroup.WithContext(ctx)

	// addSection launches a concurrent section build if the section should be included.
	addSection := func(order int, sectionType domain.FeedSectionType, build func(context.Context) (*domain.HomeFeedSection, error)) {
		if !shouldIncludeSection(options, sectionType) {
			return
		}
		g.Go(func() error {
			section, err := s.getOrBuildHomeFeedSection(gCtx, userID, options, sectionType, build)
			if err != nil {
				s.log.Warn("failed to build section", "type", sectionType, "error", err)
				return nil // Don't fail other sections
			}
			if section != nil {
				mu.Lock()
				results = append(results, indexedSection{order: order, section: *section})
				mu.Unlock()
			}
			return nil
		})
	}

	// 1. Featured section
	addSection(0, domain.FeedSectionFeatured, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildFeaturedSection(innerCtx, options)
	})

	// 2. Premium section
	addSection(1, domain.FeedSectionPremium, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildPremiumSection(innerCtx, options)
	})

	// 3. Recent section
	addSection(2, domain.FeedSectionRecent, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildRecentSection(innerCtx, options)
	})

	// 4. Near you section
	addSection(3, domain.FeedSectionNearYou, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildNearYouSection(innerCtx, options)
	})

	// 5-7. Area sections — only build the relevant one when listingType is set,
	// otherwise build all three for the mixed feed.
	if options.ListingType == nil || *options.ListingType == string(propertydomain.ListingRent) {
		addSection(4, domain.FeedSectionRentalsArea, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
			return s.buildListingTypeAreaSection(innerCtx, options, propertydomain.ListingRent, domain.FeedSectionRentalsArea, "Rentals")
		})
	}

	if options.ListingType == nil || *options.ListingType == string(propertydomain.ListingShortLet) {
		addSection(5, domain.FeedSectionShortletsArea, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
			return s.buildListingTypeAreaSection(innerCtx, options, propertydomain.ListingShortLet, domain.FeedSectionShortletsArea, "Shortlets")
		})
	}

	if options.ListingType == nil || *options.ListingType == string(propertydomain.ListingSale) {
		addSection(6, domain.FeedSectionForSaleArea, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
			return s.buildListingTypeAreaSection(innerCtx, options, propertydomain.ListingSale, domain.FeedSectionForSaleArea, "For Sale")
		})
	}

	// 8. Trending section
	addSection(7, domain.FeedSectionTrending, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildTrendingSection(innerCtx, options)
	})

	// 9. Top rated section
	addSection(8, domain.FeedSectionTopRated, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildTopRatedSection(innerCtx, options)
	})

	// 10. Budget friendly section
	addSection(9, domain.FeedSectionBudgetFriendly, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildBudgetFriendlySection(innerCtx, options)
	})

	// 11. Verified only section
	addSection(10, domain.FeedSectionVerifiedOnly, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
		return s.buildVerifiedOnlySection(innerCtx, options)
	})

	// 12. Large groups section (shortlet only)
	if options.ListingType == nil || *options.ListingType == string(propertydomain.ListingShortLet) {
		addSection(11, domain.FeedSectionLargeGroups, func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
			return s.buildLargeGroupsSection(innerCtx, options)
		})
	}

	// 13. Recommended section (future implementation)
	if userID != nil && shouldIncludeSection(options, domain.FeedSectionRecommended) {
		s.log.Debug("personalized recommendations not yet implemented", "userID", userID)
	}

	// Wait for all section builds to complete
	_ = g.Wait()

	// Sort by original display order
	sort.Slice(results, func(i, j int) bool {
		return results[i].order < results[j].order
	})

	sections := make([]domain.HomeFeedSection, len(results))
	for i, r := range results {
		sections[i] = r.section
	}

	return sections, nil
}

type homeFeedSectionCacheValue struct {
	Found   bool                   `json:"found"`
	Section domain.HomeFeedSection `json:"section,omitempty"`
}

func (s *ServiceImpl) getOrBuildHomeFeedSection(
	ctx context.Context,
	userID *uuid.UUID,
	options FeedOptions,
	sectionType domain.FeedSectionType,
	build func(context.Context) (*domain.HomeFeedSection, error),
) (*domain.HomeFeedSection, error) {
	cacheKey := homeFeedSectionCacheKey(sectionType, options, userID)
	var cached homeFeedSectionCacheValue
	if ok, err := s.getCachedValue(ctx, cacheKey, &cached); err == nil && ok {
		if !cached.Found {
			return nil, nil
		}
		section := cached.Section
		return &section, nil
	} else if err != nil && s.log != nil {
		s.log.Warn("home feed section cache read failed", "section", sectionType, "error", err)
	}

	computed, err, _ := s.homeFeedGroup.Do(cacheKey, func() (any, error) {
		var innerCached homeFeedSectionCacheValue
		if ok, err := s.getCachedValue(ctx, cacheKey, &innerCached); err == nil && ok {
			return innerCached, nil
		} else if err != nil && s.log != nil {
			s.log.Warn("home feed section cache read failed", "section", sectionType, "error", err)
		}

		section, err := build(ctx)
		if err != nil {
			return nil, err
		}

		payload := homeFeedSectionCacheValue{Found: section != nil}
		if section != nil {
			payload.Section = *section
		}

		s.setCachedValue(ctx, cacheKey, homeFeedSectionCacheTTL, payload)
		return payload, nil
	})
	if err != nil {
		return nil, err
	}

	payload, ok := computed.(homeFeedSectionCacheValue)
	if !ok {
		return nil, fmt.Errorf("unexpected home feed section cache value type %T", computed)
	}

	if !payload.Found {
		return nil, nil
	}

	section := payload.Section
	return &section, nil
}

// buildFeaturedSection builds the featured listings section.
// When options.ListingType is set, only listings matching that type are included.
func (s *ServiceImpl) buildFeaturedSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	// Fetch more than needed when filtering by type, since we post-filter.
	fetchLimit := options.Limit
	if options.ListingType != nil {
		fetchLimit = options.Limit * 3
	}

	featuredIDs, err := s.promotionHooks.GetFeaturedListings(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}
	if len(featuredIDs) == 0 {
		return nil, nil
	}

	listings, err := s.propertyHooks.GetListingsByIDs(ctx, featuredIDs)
	if err != nil {
		return nil, err
	}

	listings = filterListingsByType(listings, options.ListingType)
	if len(listings) == 0 {
		return nil, nil
	}

	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, extractListingIDsFromSlice(listings))
	if err != nil {
		s.log.Warn("failed to fetch promotion info for featured listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	rankedListings := s.rankListings(convertToScoredListings(listings, 1.0), promotions, s.rankingConfig, nil)
	if len(rankedListings) > options.Limit {
		rankedListings = rankedListings[:options.Limit]
	}
	s.enrichShortletPreviewData(ctx, rankedListings)

	title := "Featured Properties"
	if options.ListingType != nil {
		title = fmt.Sprintf("Featured %s", listingTypeLabel(*options.ListingType))
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionFeatured,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// buildPremiumSection builds the premium listings section.
// When options.ListingType is set, only listings matching that type are included.
func (s *ServiceImpl) buildPremiumSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	fetchLimit := options.Limit
	if options.ListingType != nil {
		fetchLimit = options.Limit * 3
	}

	premiumIDs, err := s.promotionHooks.GetPremiumListings(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}
	if len(premiumIDs) == 0 {
		return nil, nil
	}

	listings, err := s.propertyHooks.GetListingsByIDs(ctx, premiumIDs)
	if err != nil {
		return nil, err
	}

	listings = filterListingsByType(listings, options.ListingType)
	if len(listings) == 0 {
		return nil, nil
	}

	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, extractListingIDsFromSlice(listings))
	if err != nil {
		s.log.Warn("failed to fetch promotion info for premium listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	rankedListings := s.rankListings(convertToScoredListings(listings, 1.0), promotions, s.rankingConfig, nil)
	if len(rankedListings) > options.Limit {
		rankedListings = rankedListings[:options.Limit]
	}
	s.enrichShortletPreviewData(ctx, rankedListings)

	title := "Premium Listings"
	if options.ListingType != nil {
		title = fmt.Sprintf("Premium %s", listingTypeLabel(*options.ListingType))
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionPremium,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// buildRecentSection builds the recent listings section.
// When options.ListingType is set, only listings matching that type are included.
func (s *ServiceImpl) buildRecentSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	fetchLimit := options.Limit
	if options.ListingType != nil {
		fetchLimit = options.Limit * 3
	}

	recentListings, err := s.propertyHooks.GetRecentListings(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}

	recentListings = filterListingsByType(recentListings, options.ListingType)
	if len(recentListings) == 0 {
		return nil, nil
	}

	listingIDs := extractListingIDs(recentListings)
	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, listingIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info for recent listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	rankedListings := s.rankListings(convertToScoredListings(recentListings, 1.0), promotions, s.rankingConfig, nil)
	if len(rankedListings) > options.Limit {
		rankedListings = rankedListings[:options.Limit]
	}
	s.enrichShortletPreviewData(ctx, rankedListings)

	title := "Recently Added"
	if options.ListingType != nil {
		title = fmt.Sprintf("Recently Added %s", listingTypeLabel(*options.ListingType))
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionRecent,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(domain.FeedSectionRecent, SearchFilter{}, options.Limit),
	}, nil
}

// buildNearYouSection builds the near_you section from location and/or area filters.
// When options.ListingType is set, the search is scoped to that listing type.
func (s *ServiceImpl) buildNearYouSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	filter, ok := buildNearYouFilter(options)
	if !ok {
		return nil, nil
	}

	// Scope by listing type if specified.
	if options.ListingType != nil {
		filter.ListingTypes = []string{*options.ListingType}
	}

	rankedListings, err := s.searchAndRankSection(ctx, filter, options.Limit, options.Location)
	if err != nil {
		return nil, err
	}

	title := "Near You"
	if label := formatAreaLabel(options.City, options.State); label != "" {
		title = fmt.Sprintf("Near You in %s", label)
	}
	if options.ListingType != nil {
		typeLabel := listingTypeLabel(*options.ListingType)
		if areaLabel := formatAreaLabel(options.City, options.State); areaLabel != "" {
			title = fmt.Sprintf("%s Near You in %s", typeLabel, areaLabel)
		} else {
			title = fmt.Sprintf("%s Near You", typeLabel)
		}
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionNearYou,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(domain.FeedSectionNearYou, filter, options.Limit),
	}, nil
}

// buildListingTypeAreaSection builds an area section for a specific listing type
// with geo fallback widening: City+State → State → National.
// If the most specific scope doesn't have enough results (minSectionResults),
// it widens progressively and adapts the title accordingly.
func (s *ServiceImpl) buildListingTypeAreaSection(
	ctx context.Context,
	options FeedOptions,
	listingType propertydomain.ListingType,
	sectionType domain.FeedSectionType,
	titlePrefix string,
) (*domain.HomeFeedSection, error) {
	levels := buildFallbackLevels(options.City, options.State, titlePrefix)
	if len(levels) == 0 {
		// No area context at all — build a national fallback directly.
		levels = []fallbackLevel{
			{city: nil, state: nil, title: fmt.Sprintf("Popular %s", titlePrefix)},
		}
	}

	for _, level := range levels {
		filter := SearchFilter{
			ListingTypes: []string{string(listingType)},
			City:         level.city,
			State:        level.state,
		}

		rankedListings, err := s.searchAndRankSection(ctx, filter, options.Limit, nil)
		if err != nil {
			return nil, err
		}

		if len(rankedListings) >= minSectionResults {
			return &domain.HomeFeedSection{
				SectionType: sectionType,
				Title:       level.title,
				Listings:    rankedListings,
				TotalCount:  len(rankedListings),
				SearchData:  buildDiscoverSearchData(sectionType, filter, options.Limit),
			}, nil
		}

		// If this level returned any results but not enough, keep them
		// as a fallback in case all wider levels also lack data.
		if len(rankedListings) > 0 {
			// Check if this is the last level — if so, return what we have.
			if isLastFallbackLevel(levels, level) {
				return &domain.HomeFeedSection{
					SectionType: sectionType,
					Title:       level.title,
					Listings:    rankedListings,
					TotalCount:  len(rankedListings),
					SearchData:  buildDiscoverSearchData(sectionType, filter, options.Limit),
				}, nil
			}
		}
	}

	// No results at any level.
	return nil, nil
}

// ===== NEW SECTION BUILDERS =====

// buildTrendingSection builds the "Trending" section from interaction_aggregates.
func (s *ServiceImpl) buildTrendingSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	if s.analyticsHooks == nil {
		return nil, nil
	}

	// Fetch more when filtering by type since we post-filter.
	fetchLimit := options.Limit
	if options.ListingType != nil {
		fetchLimit = options.Limit * 3
	}

	trendingIDs, err := s.analyticsHooks.GetTrendingListingIDs(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}
	if len(trendingIDs) == 0 {
		return nil, nil
	}

	listings, err := s.propertyHooks.GetListingsByIDs(ctx, trendingIDs)
	if err != nil {
		return nil, err
	}

	listings = filterListingsByType(listings, options.ListingType)
	if len(listings) == 0 {
		return nil, nil
	}

	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, extractListingIDsFromSlice(listings))
	if err != nil {
		s.log.Warn("failed to fetch promotions for trending", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	scoredListings := convertToScoredListings(listings, 1.0)
	rankedListings := s.rankListings(scoredListings, promotions, s.rankingConfig, nil)
	if len(rankedListings) > options.Limit {
		rankedListings = rankedListings[:options.Limit]
	}
	s.enrichShortletPreviewData(ctx, rankedListings)

	title := "Trending Now"
	if options.ListingType != nil {
		title = fmt.Sprintf("Trending %s", listingTypeLabel(*options.ListingType))
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionTrending,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// buildTopRatedSection builds the "Top Rated" section from listing_stats.
func (s *ServiceImpl) buildTopRatedSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	if s.analyticsHooks == nil {
		return nil, nil
	}

	fetchLimit := options.Limit
	if options.ListingType != nil {
		fetchLimit = options.Limit * 3
	}

	topRatedIDs, err := s.analyticsHooks.GetTopRatedListingIDs(ctx, fetchLimit)
	if err != nil {
		return nil, err
	}
	if len(topRatedIDs) == 0 {
		return nil, nil
	}

	listings, err := s.propertyHooks.GetListingsByIDs(ctx, topRatedIDs)
	if err != nil {
		return nil, err
	}

	listings = filterListingsByType(listings, options.ListingType)
	if len(listings) == 0 {
		return nil, nil
	}

	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, extractListingIDsFromSlice(listings))
	if err != nil {
		s.log.Warn("failed to fetch promotions for top rated", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	scoredListings := convertToScoredListings(listings, 1.0)
	rankedListings := s.rankListings(scoredListings, promotions, s.rankingConfig, nil)
	if len(rankedListings) > options.Limit {
		rankedListings = rankedListings[:options.Limit]
	}
	s.enrichShortletPreviewData(ctx, rankedListings)

	title := "Top Rated"
	if options.ListingType != nil {
		title = fmt.Sprintf("Top Rated %s", listingTypeLabel(*options.ListingType))
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionTopRated,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// budgetThresholds defines the max price (in base currency units, e.g. Naira)
// for a listing to be considered "budget-friendly" per listing type.
// These are static thresholds — intentionally simple.
var budgetThresholds = map[propertydomain.ListingType]float64{
	propertydomain.ListingShortLet: 30000,    // ₦30k/night
	propertydomain.ListingRent:     200000,   // ₦200k/month
	propertydomain.ListingSale:     30000000, // ₦30M
}

// buildBudgetFriendlySection builds the "Budget-Friendly" section using a price ceiling.
func (s *ServiceImpl) buildBudgetFriendlySection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	// Determine which listing type(s) to query.
	listingTypes := []string{
		string(propertydomain.ListingShortLet),
		string(propertydomain.ListingRent),
		string(propertydomain.ListingSale),
	}
	if options.ListingType != nil {
		listingTypes = []string{*options.ListingType}
	}

	// Use the budget threshold for the target type (or the lowest if mixed).
	maxPrice := budgetThresholds[propertydomain.ListingShortLet] // default
	if options.ListingType != nil {
		if threshold, ok := budgetThresholds[propertydomain.ListingType(*options.ListingType)]; ok {
			maxPrice = threshold
		}
	}

	maxPriceCents := int64(maxPrice * 100) // Convert to cents

	filter := SearchFilter{
		ListingTypes: listingTypes,
		City:         options.City,
		State:        options.State,
		PriceRange: &PriceRangeFilter{
			Max: &maxPriceCents,
		},
	}

	rankedListings, err := s.searchAndRankSection(ctx, filter, options.Limit, nil)
	if err != nil {
		return nil, err
	}
	if len(rankedListings) == 0 {
		return nil, nil
	}

	title := "Budget-Friendly"
	if options.ListingType != nil {
		title = fmt.Sprintf("Budget-Friendly %s", listingTypeLabel(*options.ListingType))
	}
	if area := formatAreaLabel(options.City, options.State); area != "" {
		title = fmt.Sprintf("%s in %s", title, area)
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionBudgetFriendly,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(domain.FeedSectionBudgetFriendly, filter, options.Limit),
	}, nil
}

// buildVerifiedOnlySection builds the "Verified" section using the is_verified filter.
func (s *ServiceImpl) buildVerifiedOnlySection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	verified := true

	listingTypes := []string{
		string(propertydomain.ListingShortLet),
		string(propertydomain.ListingRent),
		string(propertydomain.ListingSale),
	}
	if options.ListingType != nil {
		listingTypes = []string{*options.ListingType}
	}

	filter := SearchFilter{
		ListingTypes: listingTypes,
		City:         options.City,
		State:        options.State,
		IsVerified:   &verified,
	}

	rankedListings, err := s.searchAndRankSection(ctx, filter, options.Limit, nil)
	if err != nil {
		return nil, err
	}
	if len(rankedListings) == 0 {
		return nil, nil
	}

	title := "Verified Properties"
	if options.ListingType != nil {
		title = fmt.Sprintf("Verified %s", listingTypeLabel(*options.ListingType))
	}
	if area := formatAreaLabel(options.City, options.State); area != "" {
		title = fmt.Sprintf("%s in %s", title, area)
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionVerifiedOnly,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(domain.FeedSectionVerifiedOnly, filter, options.Limit),
	}, nil
}

// largeGroupMinGuests is the minimum max_guests threshold for "large groups".
const largeGroupMinGuests = 6

// buildLargeGroupsSection builds the "Large Groups" section for shortlets
// with high guest capacity.
func (s *ServiceImpl) buildLargeGroupsSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	guestCount := largeGroupMinGuests

	filter := SearchFilter{
		ListingTypes: []string{string(propertydomain.ListingShortLet)},
		City:         options.City,
		State:        options.State,
		GuestCount:   &guestCount,
	}

	rankedListings, err := s.searchAndRankSection(ctx, filter, options.Limit, nil)
	if err != nil {
		return nil, err
	}
	if len(rankedListings) == 0 {
		return nil, nil
	}

	title := "Perfect for Large Groups"
	if area := formatAreaLabel(options.City, options.State); area != "" {
		title = fmt.Sprintf("%s in %s", title, area)
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionLargeGroups,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(domain.FeedSectionLargeGroups, filter, options.Limit),
	}, nil
}

func (s *ServiceImpl) searchAndRankSection(
	ctx context.Context,
	filter SearchFilter,
	limit int,
	locationFilter *LocationFilter,
) ([]domain.RankedListing, error) {
	propertyResults, err := s.propertyHooks.SearchListingsWithEmbedding(ctx, filter, limit, nil)
	if err != nil {
		return nil, err
	}
	if len(propertyResults) == 0 {
		return []domain.RankedListing{}, nil
	}

	listingIDs := extractListingIDsFromScored(propertyResults)
	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, listingIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info for section listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	rankedListings := s.rankListings(propertyResults, promotions, s.rankingConfig, locationFilter)
	if len(rankedListings) > limit {
		rankedListings = rankedListings[:limit]
	}
	s.enrichShortletPreviewData(ctx, rankedListings)

	return rankedListings, nil
}

func normalizeFeedOptions(options FeedOptions) FeedOptions {
	if options.Limit <= 0 {
		options.Limit = defaultHomeSectionLimit
	}
	if options.Limit > maxSectionLimit {
		options.Limit = maxSectionLimit
	}

	options.City = sanitizeTextPtr(options.City)
	options.State = sanitizeTextPtr(options.State)

	if hasAreaContext(options) && len(options.SectionsToInclude) > 0 {
		options.SectionsToInclude = ensureSectionIncluded(options.SectionsToInclude, domain.FeedSectionNearYou)
	}

	return options
}

func sanitizeTextPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func hasAreaContext(options FeedOptions) bool {
	return options.City != nil || options.State != nil
}

func ensureSectionIncluded(
	sections []domain.FeedSectionType,
	sectionType domain.FeedSectionType,
) []domain.FeedSectionType {
	for _, section := range sections {
		if section == sectionType {
			return sections
		}
	}
	return append(sections, sectionType)
}

func buildNearYouFilter(options FeedOptions) (SearchFilter, bool) {
	filter := SearchFilter{
		Location: options.Location,
		City:     options.City,
		State:    options.State,
	}

	if filter.Location == nil && filter.City == nil && filter.State == nil {
		return SearchFilter{}, false
	}

	return filter, true
}

func formatAreaLabel(city, state *string) string {
	switch {
	case city != nil && state != nil:
		return fmt.Sprintf("%s, %s", *city, *state)
	case city != nil:
		return *city
	case state != nil:
		return *state
	default:
		return ""
	}
}

// ===== LISTING TYPE HELPERS =====

// listingTypeLabel returns a human-readable plural label for a listing type.
func listingTypeLabel(lt string) string {
	switch propertydomain.ListingType(lt) {
	case propertydomain.ListingShortLet:
		return "Shortlets"
	case propertydomain.ListingRent:
		return "Rentals"
	case propertydomain.ListingSale:
		return "For Sale"
	default:
		return strings.Title(lt) //nolint:staticcheck // acceptable for display labels
	}
}

// filterListingsByType returns only listings matching the given listing type.
// If listingType is nil, all listings are returned unchanged.
func filterListingsByType(listings []propertydomain.Listing, listingType *string) []propertydomain.Listing {
	if listingType == nil {
		return listings
	}

	lt := propertydomain.ListingType(*listingType)
	filtered := make([]propertydomain.Listing, 0, len(listings))
	for _, l := range listings {
		if l.ListingType == lt {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

// extractListingIDsFromSlice extracts listing IDs from a slice of listings.
func extractListingIDsFromSlice(listings []propertydomain.Listing) []uuid.UUID {
	ids := make([]uuid.UUID, len(listings))
	for i, l := range listings {
		ids[i] = l.ID
	}
	return ids
}

// ===== GEO FALLBACK HELPERS =====

type fallbackLevel struct {
	city  *string
	state *string
	title string
}

// buildFallbackLevels generates the fallback ladder for geo widening.
// City+State → State-only → National (no geo filter).
func buildFallbackLevels(city, state *string, titlePrefix string) []fallbackLevel {
	var levels []fallbackLevel

	// Level 1: City + State (most specific)
	if city != nil {
		label := formatAreaLabel(city, state)
		levels = append(levels, fallbackLevel{
			city:  city,
			state: state,
			title: fmt.Sprintf("%s in %s", titlePrefix, label),
		})
	}

	// Level 2: State only
	if state != nil {
		// Only add if it's different from level 1 (i.e., city was also set).
		if city != nil {
			levels = append(levels, fallbackLevel{
				city:  nil,
				state: state,
				title: fmt.Sprintf("%s in %s", titlePrefix, *state),
			})
		} else {
			// Only state was provided — this is already level 1.
			levels = append(levels, fallbackLevel{
				city:  nil,
				state: state,
				title: fmt.Sprintf("%s in %s", titlePrefix, *state),
			})
		}
	}

	// Level 3: National fallback (no geo filter)
	levels = append(levels, fallbackLevel{
		city:  nil,
		state: nil,
		title: fmt.Sprintf("Popular %s", titlePrefix),
	})

	return levels
}

// isLastFallbackLevel checks if the given level is the last in the ladder.
func isLastFallbackLevel(levels []fallbackLevel, current fallbackLevel) bool {
	if len(levels) == 0 {
		return true
	}
	last := levels[len(levels)-1]
	return ptrStringEqual(current.city, last.city) &&
		ptrStringEqual(current.state, last.state)
}

func ptrStringEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func buildDiscoverSearchData(
	sectionType domain.FeedSectionType,
	filter SearchFilter,
	limit int,
) map[string]any {
	filterData := make(map[string]any)

	if filter.Query != nil {
		filterData["query"] = *filter.Query
	}
	if filter.Location != nil {
		filterData["location"] = map[string]any{
			"lat":      filter.Location.Latitude,
			"lng":      filter.Location.Longitude,
			"radiusKm": filter.Location.RadiusKm,
		}
	}
	if len(filter.ListingTypes) > 0 {
		filterData["listingTypes"] = append([]string(nil), filter.ListingTypes...)
	}
	if filter.City != nil {
		filterData["city"] = *filter.City
	}
	if filter.State != nil {
		filterData["state"] = *filter.State
	}

	return map[string]any{
		"sectionType": sectionType.String(),
		"filter":      filterData,
		"options": map[string]any{
			"limit":           limit,
			"includePromoted": true,
		},
	}
}

type shortletPreviewCacheValue struct {
	Found                 bool    `json:"found"`
	GuestCountUsed        int     `json:"guest_count_used,omitempty"`
	NightsUsed            int     `json:"nights_used,omitempty"`
	NextAvailableCheckIn  string  `json:"next_available_check_in,omitempty"`
	NextAvailableCheckOut string  `json:"next_available_check_out,omitempty"`
	TotalPrice            float64 `json:"total_price,omitempty"`
	Currency              string  `json:"currency,omitempty"`
}

func (v shortletPreviewCacheValue) toPayload() map[string]any {
	if !v.Found {
		return nil
	}
	return map[string]any{
		"isPreview":             true,
		"guestCountUsed":        v.GuestCountUsed,
		"nightsUsed":            v.NightsUsed,
		"nextAvailableCheckIn":  v.NextAvailableCheckIn,
		"nextAvailableCheckOut": v.NextAvailableCheckOut,
		"totalPrice":            v.TotalPrice,
		"currency":              v.Currency,
	}
}

func (s *ServiceImpl) enrichShortletPreviewData(ctx context.Context, rankedListings []domain.RankedListing) {
	if s.bookingHooks == nil || len(rankedListings) == 0 {
		return
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxShortletPreviewConcurrency)

	for i := range rankedListings {
		i := i // capture loop variable
		listing := rankedListings[i].Listing
		if listing.ListingType != propertydomain.ListingShortLet || listing.ShortletDetails == nil {
			continue
		}

		g.Go(func() error {
			payload, err := s.buildShortletPreviewPayload(gCtx, listing)
			if err != nil {
				s.log.Warn("failed to build shortlet preview data", "listingID", listing.ID, "error", err)
				return nil // Don't fail other previews
			}
			if payload == nil {
				return nil
			}

			// Safe: each goroutine writes to a unique index
			if rankedListings[i].Data == nil {
				rankedListings[i].Data = make(map[string]any)
			}
			rankedListings[i].Data[shortletPreviewPayloadFieldName] = payload
			return nil
		})
	}

	_ = g.Wait()
}

func (s *ServiceImpl) buildShortletPreviewPayload(
	ctx context.Context,
	listing propertydomain.Listing,
) (map[string]any, error) {
	if listing.ShortletDetails == nil {
		return nil, nil
	}

	guestCount := deriveShortletPreviewGuestCount(listing.ShortletDetails)
	nights := deriveShortletPreviewNights(listing.ShortletDetails)
	startDate := beginningOfUTCDate(time.Now())
	cacheKey := shortletPreviewCacheKey(listing.ID, guestCount, nights, startDate)

	var cached shortletPreviewCacheValue
	if ok, err := s.getCachedValue(ctx, cacheKey, &cached); err == nil && ok {
		return cached.toPayload(), nil
	} else if err != nil && s.log != nil {
		s.log.Warn("shortlet preview cache read failed", "listingID", listing.ID, "error", err)
	}

	computed, err, _ := s.previewGroup.Do(cacheKey, func() (any, error) {
		var innerCached shortletPreviewCacheValue
		if ok, err := s.getCachedValue(ctx, cacheKey, &innerCached); err == nil && ok {
			return innerCached, nil
		} else if err != nil && s.log != nil {
			s.log.Warn("shortlet preview cache read failed", "listingID", listing.ID, "error", err)
		}

		result := shortletPreviewCacheValue{Found: false}
		for dayOffset := 0; dayOffset <= shortletPreviewLookAheadDays; dayOffset++ {
			checkIn := startDate.AddDate(0, 0, dayOffset)
			checkOut := checkIn.AddDate(0, 0, nights)

			quote, err := s.bookingHooks.QuoteBooking(ctx, listing.ID, checkIn, checkOut, guestCount)
			if err != nil {
				return nil, err
			}
			if quote == nil || !quote.Available {
				continue
			}

			result = shortletPreviewCacheValue{
				Found:                 true,
				GuestCountUsed:        guestCount,
				NightsUsed:            nights,
				NextAvailableCheckIn:  quote.CheckIn.Format(time.RFC3339),
				NextAvailableCheckOut: quote.CheckOut.Format(time.RFC3339),
				TotalPrice:            quote.TotalPrice,
				Currency:              quote.Currency,
			}
			break
		}
		s.setCachedValue(ctx, cacheKey, shortletPreviewCacheTTL, result)
		return result, nil
	})
	if err != nil {
		return nil, err
	}

	result, ok := computed.(shortletPreviewCacheValue)
	if !ok {
		return nil, fmt.Errorf("unexpected shortlet preview cache value type %T", computed)
	}

	return result.toPayload(), nil
}

func deriveShortletPreviewGuestCount(details *propertydomain.ShortletDetail) int {
	if details == nil {
		return defaultShortletPreviewGuests
	}

	if details.BaseGuestCount != nil && *details.BaseGuestCount > 0 {
		return *details.BaseGuestCount
	}

	if details.MaxGuests > 0 {
		if details.MaxGuests < defaultShortletPreviewGuests {
			return details.MaxGuests
		}
		return defaultShortletPreviewGuests
	}

	return 1
}

func deriveShortletPreviewNights(details *propertydomain.ShortletDetail) int {
	nights := defaultShortletPreviewNights
	if details == nil {
		return nights
	}

	if details.StayLimits.MinNights > nights {
		nights = details.StayLimits.MinNights
	}

	if details.StayLimits.MaxNights != nil && *details.StayLimits.MaxNights > 0 && nights > *details.StayLimits.MaxNights {
		nights = *details.StayLimits.MaxNights
	}

	if nights < 1 {
		nights = 1
	}

	return nights
}

func beginningOfUTCDate(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

// shouldIncludeSection checks if a section should be included based on options.
func shouldIncludeSection(options FeedOptions, sectionType domain.FeedSectionType) bool {
	if len(options.SectionsToInclude) == 0 {
		return true
	}

	for _, requestedSection := range options.SectionsToInclude {
		if requestedSection == sectionType {
			return true
		}
	}

	return false
}
