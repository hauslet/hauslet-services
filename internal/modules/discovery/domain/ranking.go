package domain

import "time"

// RankingConfig holds configuration for the ranking algorithm
type RankingConfig struct {
	SemanticWeight        float64 // Weight for semantic similarity score
	PromotionWeight       float64 // Weight for promotion boost
	RecencyWeight         float64 // Weight for listing recency
	LocationWeight        float64 // Weight for location proximity
	PersonalizationWeight float64 // Weight for personalization (future)
}

// DefaultRankingConfig returns the default ranking configuration
func DefaultRankingConfig() RankingConfig {
	return RankingConfig{
		SemanticWeight:        0.8,
		PromotionWeight:       0.1,
		RecencyWeight:         0.1,
		LocationWeight:        0.1,
		PersonalizationWeight: 0.0,
	}
}

// Validate checks if the ranking configuration is valid
func (r RankingConfig) Validate() error {
	// Ensure all weights are non-negative
	if r.SemanticWeight < 0 || r.PromotionWeight < 0 ||
		r.RecencyWeight < 0 || r.LocationWeight < 0 ||
		r.PersonalizationWeight < 0 {
		return ErrInvalidRankingConfig
	}

	// Weights don't need to sum to 1.0 as we normalize the final score
	return nil
}

// RankingScore provides transparency into how the final score was calculated
type RankingScore struct {
	FinalScore        float64  // Final combined score
	SemanticScore     *float64 // Semantic similarity score (nil if not semantic search)
	PromotionBoost    float64  // Promotion boost multiplier (1.0 = no boost)
	LocationScore     *float64 // Location proximity score (nil if not location-based)
	RecencyScore      float64  // Recency score (decay over time)
	PersonalizedScore *float64 // Personalization score (nil if not personalized, future)
}

// CalculateFinalScore combines all ranking factors using the provided config
func (r RankingScore) CalculateFinalScore(config RankingConfig) float64 {
	score := 0.0

	// Add semantic score if available
	if r.SemanticScore != nil {
		score += *r.SemanticScore * config.SemanticWeight
	}

	// Add promotion boost
	score += r.PromotionBoost * config.PromotionWeight

	// Add recency score
	score += r.RecencyScore * config.RecencyWeight

	// Add location score if available
	if r.LocationScore != nil {
		score += *r.LocationScore * config.LocationWeight
	}

	// Add personalization score if available
	if r.PersonalizedScore != nil {
		score += *r.PersonalizedScore * config.PersonalizationWeight
	}

	return score
}

// CalculateRecencyScore calculates a recency score based on creation time
// Returns 1.0 for new listings, decaying to 0.1 for old listings
func CalculateRecencyScore(createdAt time.Time) float64 {
	daysSinceCreation := time.Since(createdAt).Hours() / 24

	// Tiered decay function
	if daysSinceCreation <= 7 {
		return 1.0 // New listings (within 7 days)
	} else if daysSinceCreation <= 30 {
		return 0.8 // Recent listings (7-30 days)
	} else if daysSinceCreation <= 90 {
		return 0.5 // Older listings (30-90 days)
	} else {
		return 0.1 // Very old listings (90+ days)
	}
}
