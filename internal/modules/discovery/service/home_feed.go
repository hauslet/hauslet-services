package service

import (
	"context"

	"hauslet/internal/modules/discovery/domain"

	"github.com/google/uuid"
)

// GetHomeFeed returns a curated home feed with multiple sections
func (s *ServiceImpl) GetHomeFeed(ctx context.Context, userID *uuid.UUID, options FeedOptions) ([]domain.HomeFeedSection, error) {
	sections := make([]domain.HomeFeedSection, 0)

	// Apply default limit if not specified
	if options.Limit == 0 {
		options.Limit = 10
	}

	// 1. Featured Section (promoted listings with highest boost)
	if shouldIncludeSection(options, domain.FeedSectionFeatured) {
		featuredSection, err := s.buildFeaturedSection(ctx, options.Limit)
		if err != nil {
			s.log.Warn("failed to build featured section", "error", err)
		} else if featuredSection != nil {
			sections = append(sections, *featuredSection)
		}
	}

	// 2. Premium Section (premium promoted listings)
	if shouldIncludeSection(options, domain.FeedSectionPremium) {
		premiumSection, err := s.buildPremiumSection(ctx, options.Limit)
		if err != nil {
			s.log.Warn("failed to build premium section", "error", err)
		} else if premiumSection != nil {
			sections = append(sections, *premiumSection)
		}
	}

	// 3. Recent Section (recently published listings)
	if shouldIncludeSection(options, domain.FeedSectionRecent) {
		recentSection, err := s.buildRecentSection(ctx, options.Limit*2) // Get more for diversity
		if err != nil {
			s.log.Warn("failed to build recent section", "error", err)
		} else if recentSection != nil {
			sections = append(sections, *recentSection)
		}
	}

	// 4. Recommended Section (personalized, if user authenticated) - Future implementation
	if userID != nil && shouldIncludeSection(options, domain.FeedSectionRecommended) {
		// TODO: Implement personalized recommendations
		s.log.Debug("personalized recommendations not yet implemented", "userID", userID)
	}

	return sections, nil
}

// buildFeaturedSection builds the featured listings section
func (s *ServiceImpl) buildFeaturedSection(ctx context.Context, limit int) (*domain.HomeFeedSection, error) {
	// Get featured listing IDs
	featuredIDs, err := s.promotionHooks.GetFeaturedListings(ctx, limit)
	if err != nil {
		return nil, err
	}

	if len(featuredIDs) == 0 {
		return nil, nil // No featured listings, skip section
	}

	// Fetch listings
	listings, err := s.propertyHooks.GetListingsByIDs(ctx, featuredIDs)
	if err != nil {
		return nil, err
	}

	if len(listings) == 0 {
		return nil, nil
	}

	// Get promotion info
	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, featuredIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info for featured listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	// Convert to scored and rank
	scoredListings := convertToScoredListings(listings, 1.0)
	rankedListings := s.rankListings(scoredListings, promotions, s.rankingConfig)

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionFeatured,
		Title:       "Featured Properties",
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// buildPremiumSection builds the premium listings section
func (s *ServiceImpl) buildPremiumSection(ctx context.Context, limit int) (*domain.HomeFeedSection, error) {
	// Get premium listing IDs
	premiumIDs, err := s.promotionHooks.GetPremiumListings(ctx, limit)
	if err != nil {
		return nil, err
	}

	if len(premiumIDs) == 0 {
		return nil, nil // No premium listings, skip section
	}

	// Fetch listings
	listings, err := s.propertyHooks.GetListingsByIDs(ctx, premiumIDs)
	if err != nil {
		return nil, err
	}

	if len(listings) == 0 {
		return nil, nil
	}

	// Get promotion info
	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, premiumIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info for premium listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	// Convert to scored and rank
	scoredListings := convertToScoredListings(listings, 1.0)
	rankedListings := s.rankListings(scoredListings, promotions, s.rankingConfig)

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionPremium,
		Title:       "Premium Listings",
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// buildRecentSection builds the recent listings section
func (s *ServiceImpl) buildRecentSection(ctx context.Context, limit int) (*domain.HomeFeedSection, error) {
	// Get recent listings
	recentListings, err := s.propertyHooks.GetRecentListings(ctx, limit)
	if err != nil {
		return nil, err
	}

	if len(recentListings) == 0 {
		return nil, nil // No recent listings, skip section
	}

	// Get promotion info for recent listings (to boost promoted ones)
	listingIDs := extractListingIDs(recentListings)
	promotions, err := s.promotionHooks.GetActivePromotionForListings(ctx, listingIDs)
	if err != nil {
		s.log.Warn("failed to fetch promotion info for recent listings", "error", err)
		promotions = make(map[uuid.UUID]*PromotionInfo)
	}

	// Convert to scored and rank
	scoredListings := convertToScoredListings(recentListings, 1.0)
	rankedListings := s.rankListings(scoredListings, promotions, s.rankingConfig)

	return &domain.HomeFeedSection{
		SectionType: domain.FeedSectionRecent,
		Title:       "Recently Added",
		Listings:    rankedListings,
		TotalCount:  len(rankedListings),
	}, nil
}

// shouldIncludeSection checks if a section should be included based on options
func shouldIncludeSection(options FeedOptions, sectionType domain.FeedSectionType) bool {
	// If no specific sections requested, include all
	if len(options.SectionsToInclude) == 0 {
		return true
	}

	// Check if this section type is in the requested list
	for _, requestedSection := range options.SectionsToInclude {
		if requestedSection == sectionType {
			return true
		}
	}

	return false
}
