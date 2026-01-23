package service

import (
	"math"
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
	locationFilter *LocationFilter,
) []domain.RankedListing {
	ranked := make([]domain.RankedListing, 0, len(propertyResults))

	for _, pr := range propertyResults {
		// Calculate location score if filter and listing location are available
		var locationScore *float64
		if locationFilter != nil && pr.Location != nil {
			score := CalculateLocationScore(
				locationFilter.Latitude, locationFilter.Longitude, locationFilter.RadiusKm,
				pr.Location.Lat, pr.Location.Lng,
			)
			locationScore = &score
		}

		// Build ranking score
		score := domain.RankingScore{
			SemanticScore:  pr.Score, // From property search (0-1 range)
			PromotionBoost: 1.0,      // Default: no boost
			RecencyScore:   domain.CalculateRecencyScore(pr.Listing.CreatedAt),
			LocationScore:  locationScore,
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

// CalculateLocationScore calculates a 0-1 score based on distance from a center point.
// Uses linear decay: 1.0 at center, 0.0 at radius and beyond.
func CalculateLocationScore(centerLat, centerLng, radiusKm, targetLat, targetLng float64) float64 {
	distKm := haversineDistance(centerLat, centerLng, targetLat, targetLng)
	if distKm >= radiusKm {
		return 0.0
	}
	return 1.0 - (distKm / radiusKm)
}

// haversineDistance calculates the great-circle distance between two points in kilometers.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
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
