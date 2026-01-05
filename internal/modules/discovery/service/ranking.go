package service

import (
	"sort"

	"hauslet/internal/modules/discovery/domain"
	propertydomain "hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

// rankListings applies the ranking algorithm to property search results
func (s *ServiceImpl) rankListings(
	propertyResults []propertydomain.ScoredListing,
	promotions map[uuid.UUID]*PromotionInfo,
	config domain.RankingConfig,
) []domain.RankedListing {
	ranked := make([]domain.RankedListing, 0, len(propertyResults))

	for _, pr := range propertyResults {
		// Build ranking score
		score := domain.RankingScore{
			SemanticScore:  pr.Score, // From property search (0-1 range)
			PromotionBoost: 1.0,      // Default: no boost
			RecencyScore:   domain.CalculateRecencyScore(pr.Listing.CreatedAt),
			LocationScore:  nil, // TODO: Implement location scoring
		}

		var promoInfo *domain.PromotionBoostInfo

		// Apply promotion boost if listing is promoted
		if promo, exists := promotions[pr.Listing.ID]; exists {
			score.PromotionBoost = promo.BoostMultiplier
			promoInfo = &domain.PromotionBoostInfo{
				PromotionID:     promo.PromotionID,
				PromotionType:   promo.PromotionType,
				BoostMultiplier: promo.BoostMultiplier,
				ExpiresAt:       promo.ExpiresAt,
			}
		}

		// Calculate final score using the config
		score.FinalScore = score.CalculateFinalScore(config)

		ranked = append(ranked, domain.RankedListing{
			Listing:        pr.Listing,
			Score:          score,
			Ranking:        0, // Will be set after sorting
			PromotionBoost: promoInfo,
		})
	}

	// Sort by final score (descending)
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score.FinalScore > ranked[j].Score.FinalScore
	})

	// Assign ranking positions (1-indexed)
	for i := range ranked {
		ranked[i].Ranking = i + 1
	}

	return ranked
}

// extractListingIDsFromScored extracts listing IDs from scored listings
func extractListingIDsFromScored(scoredListings []propertydomain.ScoredListing) []uuid.UUID {
	ids := make([]uuid.UUID, len(scoredListings))
	for i, sl := range scoredListings {
		ids[i] = sl.Listing.ID
	}
	return ids
}

// extractListingIDs extracts listing IDs from listings
func extractListingIDs(listings []propertydomain.Listing) []uuid.UUID {
	ids := make([]uuid.UUID, len(listings))
	for i, l := range listings {
		ids[i] = l.ID
	}
	return ids
}

// convertToScoredListings converts regular listings to scored listings with a default score
func convertToScoredListings(listings []propertydomain.Listing, defaultScore float64) []propertydomain.ScoredListing {
	scored := make([]propertydomain.ScoredListing, len(listings))
	for i, listing := range listings {
		score := &defaultScore
		scored[i] = propertydomain.ScoredListing{
			Listing: listing,
			Score:   score,
		}
	}
	return scored
}
