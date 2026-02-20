package seeders

import (
	"encoding/json"
	"fmt"
	"time"

	"hauslet/db/seeds/utils"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// listingStatsSeed mirrors the listing_stats table for seeding.
type listingStatsSeed struct {
	ListingID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	AverageRating      float64        `gorm:"type:decimal(3,2);not null;default:0.0"`
	ReviewCount        int            `gorm:"not null;default:0"`
	SubRatingAverages  datatypes.JSON `gorm:"type:jsonb"`
	RatingDistribution datatypes.JSON `gorm:"type:jsonb"`
	LastUpdatedAt      time.Time      `gorm:"index"`
}

func (listingStatsSeed) TableName() string {
	return "listing_stats"
}

// SeedListingStats seeds listing_stats directly for all active, published listings.
func SeedListingStats(ctx *SeedContext) error {
	var listingIDs []uuid.UUID
	if err := ctx.DB.Table("listings").
		Select("id").
		Where("status = ? AND published = ? AND deleted_at IS NULL", "active", true).
		Scan(&listingIDs).Error; err != nil {
		return fmt.Errorf("failed to fetch active listings: %w", err)
	}

	if len(listingIDs) == 0 {
		fmt.Println("    No active listings found — skipping listing stats seeding")
		return nil
	}

	now := time.Now()
	stats := make([]listingStatsSeed, 0, len(listingIDs))

	for _, id := range listingIDs {
		reviewCount := utils.RandomInt(3, 45)
		avgRating := randomRating()

		subRatings := buildSubRatingAverages(avgRating)
		distribution := buildRatingDistribution(reviewCount, avgRating)

		subRatingsJSON, _ := json.Marshal(subRatings)
		distributionJSON, _ := json.Marshal(distribution)

		stats = append(stats, listingStatsSeed{
			ListingID:          id,
			AverageRating:      avgRating,
			ReviewCount:        reviewCount,
			SubRatingAverages:  datatypes.JSON(subRatingsJSON),
			RatingDistribution: datatypes.JSON(distributionJSON),
			LastUpdatedAt:      now.Add(-time.Duration(utils.RandomInt(0, 30)) * 24 * time.Hour),
		})
	}

	fmt.Printf("    Creating listing stats for %d listings...\n", len(stats))
	if err := ctx.DB.CreateInBatches(stats, 100).Error; err != nil {
		return fmt.Errorf("failed to create listing stats: %w", err)
	}

	return nil
}

// randomRating generates a realistic average rating skewed toward 3.5–5.0.
func randomRating() float64 {
	// Most listings cluster between 3.8 and 5.0 (realistic distribution)
	roll := utils.RandomInt(1, 100)
	switch {
	case roll <= 5: // 5% get 2.0–3.4 (poor)
		return roundTo2(2.0 + float64(utils.RandomInt(0, 14))/10.0)
	case roll <= 20: // 15% get 3.5–3.9 (decent)
		return roundTo2(3.5 + float64(utils.RandomInt(0, 4))/10.0)
	case roll <= 50: // 30% get 4.0–4.4 (good)
		return roundTo2(4.0 + float64(utils.RandomInt(0, 4))/10.0)
	case roll <= 80: // 30% get 4.5–4.8 (great)
		return roundTo2(4.5 + float64(utils.RandomInt(0, 3))/10.0)
	default: // 20% get 4.9–5.0 (excellent)
		return roundTo2(4.9 + float64(utils.RandomInt(0, 1))/10.0)
	}
}

func roundTo2(v float64) float64 {
	return float64(int(v*100)) / 100.0
}

// buildSubRatingAverages generates per-category ratings that hover near the overall avg.
func buildSubRatingAverages(avgRating float64) map[string]float64 {
	categories := []string{
		"cleanliness", "accuracy", "check_in",
		"communication", "location", "value",
	}

	subRatings := make(map[string]float64, len(categories))
	for _, cat := range categories {
		// Vary by ±0.5 from the average, clamped to [1.0, 5.0]
		delta := float64(utils.RandomInt(-5, 5)) / 10.0
		rating := avgRating + delta
		if rating < 1.0 {
			rating = 1.0
		}
		if rating > 5.0 {
			rating = 5.0
		}
		subRatings[cat] = roundTo2(rating)
	}
	return subRatings
}

// buildRatingDistribution generates a histogram that roughly matches the avg rating.
func buildRatingDistribution(totalReviews int, avgRating float64) map[string]int {
	dist := map[string]int{"5": 0, "4": 0, "3": 0, "2": 0, "1": 0}

	// Distribute reviews to produce an average close to avgRating.
	// Higher avgRating → more 5-star reviews.
	for i := 0; i < totalReviews; i++ {
		star := weightedStarForAvg(avgRating)
		key := fmt.Sprintf("%d", star)
		dist[key]++
	}

	return dist
}

// weightedStarForAvg picks a star rating (1–5) weighted to produce the target average.
func weightedStarForAvg(avg float64) int {
	roll := utils.RandomInt(1, 100)

	switch {
	case avg >= 4.8:
		switch {
		case roll <= 80:
			return 5
		case roll <= 95:
			return 4
		default:
			return 3
		}
	case avg >= 4.3:
		switch {
		case roll <= 55:
			return 5
		case roll <= 85:
			return 4
		case roll <= 95:
			return 3
		default:
			return 2
		}
	case avg >= 3.8:
		switch {
		case roll <= 30:
			return 5
		case roll <= 60:
			return 4
		case roll <= 85:
			return 3
		case roll <= 95:
			return 2
		default:
			return 1
		}
	default: // avg < 3.8
		switch {
		case roll <= 15:
			return 5
		case roll <= 30:
			return 4
		case roll <= 55:
			return 3
		case roll <= 80:
			return 2
		default:
			return 1
		}
	}
}
