package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/discovery/domain"
	propertydomain "hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

const defaultHomeSectionLimit = 10

const (
	defaultShortletPreviewGuests    = 2
	defaultShortletPreviewNights    = 2
	shortletPreviewLookAheadDays    = 45
	shortletPreviewPayloadFieldName = "shortletPreview"
)

// GetHomeFeed returns a curated home feed with multiple sections.
func (s *ServiceImpl) GetHomeFeed(ctx context.Context, userID *uuid.UUID, options FeedOptions) ([]domain.HomeFeedSection, error) {
	options = normalizeFeedOptions(options)
	sections := make([]domain.HomeFeedSection, 0)

	// 1. Featured section
	if shouldIncludeSection(options, domain.FeedSectionFeatured) {
		featuredSection, err := s.getOrBuildHomeFeedSection(
			ctx,
			userID,
			options,
			domain.FeedSectionFeatured,
			func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
				return s.buildFeaturedSection(innerCtx, options.Limit)
			},
		)
		if err != nil {
			s.log.Warn("failed to build featured section", "error", err)
		} else if featuredSection != nil {
			sections = append(sections, *featuredSection)
		}
	}

	// 2. Premium section
	if shouldIncludeSection(options, domain.FeedSectionPremium) {
		premiumSection, err := s.getOrBuildHomeFeedSection(
			ctx,
			userID,
			options,
			domain.FeedSectionPremium,
			func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
				return s.buildPremiumSection(innerCtx, options.Limit)
			},
		)
		if err != nil {
			s.log.Warn("failed to build premium section", "error", err)
		} else if premiumSection != nil {
			sections = append(sections, *premiumSection)
		}
	}

	// 3. Recent section
	if shouldIncludeSection(options, domain.FeedSectionRecent) {
		recentSection, err := s.getOrBuildHomeFeedSection(
			ctx,
			userID,
			options,
			domain.FeedSectionRecent,
			func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
				return s.buildRecentSection(innerCtx, options.Limit)
			},
		)
		if err != nil {
			s.log.Warn("failed to build recent section", "error", err)
		} else if recentSection != nil {
			sections = append(sections, *recentSection)
		}
	}

	// 4. Near you section
	if shouldIncludeSection(options, domain.FeedSectionNearYou) {
		nearYouSection, err := s.getOrBuildHomeFeedSection(
			ctx,
			userID,
			options,
			domain.FeedSectionNearYou,
			func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
				return s.buildNearYouSection(innerCtx, options)
			},
		)
		if err != nil {
			s.log.Warn("failed to build near_you section", "error", err)
		} else if nearYouSection != nil {
			sections = append(sections, *nearYouSection)
		}
	}

	// 5. Rentals in <city>, <state>
	if shouldIncludeSection(options, domain.FeedSectionRentalsArea) {
		rentalsSection, err := s.getOrBuildHomeFeedSection(
			ctx,
			userID,
			options,
			domain.FeedSectionRentalsArea,
			func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
				return s.buildListingTypeAreaSection(
					innerCtx,
					options,
					propertydomain.ListingRent,
					domain.FeedSectionRentalsArea,
					"Rentals",
				)
			},
		)
		if err != nil {
			s.log.Warn("failed to build rentals_in_area section", "error", err)
		} else if rentalsSection != nil {
			sections = append(sections, *rentalsSection)
		}
	}

	// 6. Shortlets in <city>, <state>
	if shouldIncludeSection(options, domain.FeedSectionShortletsArea) {
		shortletsSection, err := s.getOrBuildHomeFeedSection(
			ctx,
			userID,
			options,
			domain.FeedSectionShortletsArea,
			func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
				return s.buildListingTypeAreaSection(
					innerCtx,
					options,
					propertydomain.ListingShortLet,
					domain.FeedSectionShortletsArea,
					"Shortlets",
				)
			},
		)
		if err != nil {
			s.log.Warn("failed to build shortlets_in_area section", "error", err)
		} else if shortletsSection != nil {
			sections = append(sections, *shortletsSection)
		}
	}

	// 7. For Sale in <city>, <state>
	if shouldIncludeSection(options, domain.FeedSectionForSaleArea) {
		forSaleSection, err := s.getOrBuildHomeFeedSection(
			ctx,
			userID,
			options,
			domain.FeedSectionForSaleArea,
			func(innerCtx context.Context) (*domain.HomeFeedSection, error) {
				return s.buildListingTypeAreaSection(
					innerCtx,
					options,
					propertydomain.ListingSale,
					domain.FeedSectionForSaleArea,
					"For Sale",
				)
			},
		)
		if err != nil {
			s.log.Warn("failed to build for_sale_in_area section", "error", err)
		} else if forSaleSection != nil {
			sections = append(sections, *forSaleSection)
		}
	}

	// 8. Recommended section (future implementation)
	if userID != nil && shouldIncludeSection(options, domain.FeedSectionRecommended) {
		s.log.Debug("personalized recommendations not yet implemented", "userID", userID)
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
func (s *ServiceImpl) buildFeaturedSection(ctx context.Context, limit int) (*domain.HomeFeedSection, error) {
	featuredIDs, err := s.promotionHooks.GetFeaturedListings(ctx, limit)
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
	if len(listings) == 0 {
		return nil, nil
	}

	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, featuredIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info for featured listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	rankedListings := s.rankListings(convertToScoredListings(listings, 1.0), promotions, s.rankingConfig, nil)
	s.enrichShortletPreviewData(ctx, rankedListings)

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionFeatured,
		Title:       "Featured Properties",
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// buildPremiumSection builds the premium listings section.
func (s *ServiceImpl) buildPremiumSection(ctx context.Context, limit int) (*domain.HomeFeedSection, error) {
	premiumIDs, err := s.promotionHooks.GetPremiumListings(ctx, limit)
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
	if len(listings) == 0 {
		return nil, nil
	}

	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, premiumIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info for premium listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	rankedListings := s.rankListings(convertToScoredListings(listings, 1.0), promotions, s.rankingConfig, nil)
	s.enrichShortletPreviewData(ctx, rankedListings)

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionPremium,
		Title:       "Premium Listings",
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// buildRecentSection builds the recent listings section.
func (s *ServiceImpl) buildRecentSection(ctx context.Context, limit int) (*domain.HomeFeedSection, error) {
	recentListings, err := s.propertyHooks.GetRecentListings(ctx, limit)
	if err != nil {
		return nil, err
	}
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
	s.enrichShortletPreviewData(ctx, rankedListings)

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionRecent,
		Title:       "Recently Added",
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(domain.FeedSectionRecent, SearchFilter{}, limit),
	}, nil
}

// buildNearYouSection builds the near_you section from location and/or area filters.
func (s *ServiceImpl) buildNearYouSection(ctx context.Context, options FeedOptions) (*domain.HomeFeedSection, error) {
	filter, ok := buildNearYouFilter(options)
	if !ok {
		return nil, nil
	}

	rankedListings, err := s.searchAndRankSection(ctx, filter, options.Limit, options.Location)
	if err != nil {
		return nil, err
	}

	title := "Near You"
	if label := formatAreaLabel(options.City, options.State); label != "" {
		title = fmt.Sprintf("Near You in %s", label)
	}

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionNearYou,
		Title:       title,
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(domain.FeedSectionNearYou, filter, options.Limit),
	}, nil
}

func (s *ServiceImpl) buildListingTypeAreaSection(
	ctx context.Context,
	options FeedOptions,
	listingType propertydomain.ListingType,
	sectionType domain.FeedSectionType,
	titlePrefix string,
) (*domain.HomeFeedSection, error) {
	label := formatAreaLabel(options.City, options.State)
	if label == "" {
		return nil, nil
	}

	filter := SearchFilter{
		ListingTypes: []string{string(listingType)},
		City:         options.City,
		State:        options.State,
	}

	rankedListings, err := s.searchAndRankSection(ctx, filter, options.Limit, nil)
	if err != nil {
		return nil, err
	}
	if len(rankedListings) == 0 {
		return nil, nil
	}

	return &domain.HomeFeedSection{
		SectionType: sectionType,
		Title:       fmt.Sprintf("%s in %s", titlePrefix, label),
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
		SearchData:  buildDiscoverSearchData(sectionType, filter, options.Limit),
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

	for i := range rankedListings {
		listing := rankedListings[i].Listing
		if listing.ListingType != propertydomain.ListingShortLet || listing.ShortletDetails == nil {
			continue
		}

		payload, err := s.buildShortletPreviewPayload(ctx, listing)
		if err != nil {
			s.log.Warn("failed to build shortlet preview data", "listingID", listing.ID, "error", err)
			continue
		}
		if payload == nil {
			continue
		}

		if rankedListings[i].Data == nil {
			rankedListings[i].Data = make(map[string]any)
		}
		rankedListings[i].Data[shortletPreviewPayloadFieldName] = payload
	}
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
