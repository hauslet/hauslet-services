package service

import (
	"context"
	"fmt"
	"strings"

	"hauslet/internal/modules/discovery/domain"
	propertydomain "hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

const defaultHomeSectionLimit = 10

// GetHomeFeed returns a curated home feed with multiple sections.
func (s *ServiceImpl) GetHomeFeed(ctx context.Context, userID *uuid.UUID, options FeedOptions) ([]domain.HomeFeedSection, error) {
	options = normalizeFeedOptions(options)
	sections := make([]domain.HomeFeedSection, 0)

	// 1. Featured section
	if shouldIncludeSection(options, domain.FeedSectionFeatured) {
		featuredSection, err := s.buildFeaturedSection(ctx, options.Limit)
		if err != nil {
			s.log.Warn("failed to build featured section", "error", err)
		} else if featuredSection != nil {
			sections = append(sections, *featuredSection)
		}
	}

	// 2. Premium section
	if shouldIncludeSection(options, domain.FeedSectionPremium) {
		premiumSection, err := s.buildPremiumSection(ctx, options.Limit)
		if err != nil {
			s.log.Warn("failed to build premium section", "error", err)
		} else if premiumSection != nil {
			sections = append(sections, *premiumSection)
		}
	}

	// 3. Recent section
	if shouldIncludeSection(options, domain.FeedSectionRecent) {
		recentSection, err := s.buildRecentSection(ctx, options.Limit)
		if err != nil {
			s.log.Warn("failed to build recent section", "error", err)
		} else if recentSection != nil {
			sections = append(sections, *recentSection)
		}
	}

	// 4. Near you section
	if shouldIncludeSection(options, domain.FeedSectionNearYou) {
		nearYouSection, err := s.buildNearYouSection(ctx, options)
		if err != nil {
			s.log.Warn("failed to build near_you section", "error", err)
		} else if nearYouSection != nil {
			sections = append(sections, *nearYouSection)
		}
	}

	// 5. Rentals in <city>, <state>
	if shouldIncludeSection(options, domain.FeedSectionRentalsArea) {
		rentalsSection, err := s.buildListingTypeAreaSection(
			ctx,
			options,
			propertydomain.ListingRent,
			domain.FeedSectionRentalsArea,
			"Rentals",
		)
		if err != nil {
			s.log.Warn("failed to build rentals_in_area section", "error", err)
		} else if rentalsSection != nil {
			sections = append(sections, *rentalsSection)
		}
	}

	// 6. Shortlets in <city>, <state>
	if shouldIncludeSection(options, domain.FeedSectionShortletsArea) {
		shortletsSection, err := s.buildListingTypeAreaSection(
			ctx,
			options,
			propertydomain.ListingShortLet,
			domain.FeedSectionShortletsArea,
			"Shortlets",
		)
		if err != nil {
			s.log.Warn("failed to build shortlets_in_area section", "error", err)
		} else if shortletsSection != nil {
			sections = append(sections, *shortletsSection)
		}
	}

	// 7. For Sale in <city>, <state>
	if shouldIncludeSection(options, domain.FeedSectionForSaleArea) {
		forSaleSection, err := s.buildListingTypeAreaSection(
			ctx,
			options,
			propertydomain.ListingSale,
			domain.FeedSectionForSaleArea,
			"For Sale",
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
