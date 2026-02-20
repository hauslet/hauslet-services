package seeders

import (
	"fmt"
	"time"

	"hauslet/db/seeds/utils"
	promoSchema "hauslet/internal/modules/promotions/repository/schema"

	"github.com/google/uuid"
)

// SeedPromotions seeds featured and premium listing promotions for active listings.
func SeedPromotions(ctx *SeedContext) error {
	// Fetch all active, published listing IDs with their owner IDs
	type listingRow struct {
		ID      uuid.UUID `gorm:"column:id"`
		OwnerID uuid.UUID `gorm:"column:owner_id"`
	}

	var activeListings []listingRow
	if err := ctx.DB.Table("listings").
		Select("id, owner_id").
		Where("status = ? AND published = ? AND deleted_at IS NULL", "active", true).
		Scan(&activeListings).Error; err != nil {
		return fmt.Errorf("failed to fetch active listings: %w", err)
	}

	if len(activeListings) == 0 {
		fmt.Println("    No active listings found — skipping promotions seeding")
		return nil
	}

	// Shuffle to randomize which listings get promotions
	utils.Shuffle(activeListings)

	// Allocate ~25% featured and ~20% premium (no overlap)
	featuredCount := max(2, len(activeListings)*25/100)
	premiumCount := max(2, len(activeListings)*20/100)

	// Don't exceed available listings
	totalPromos := featuredCount + premiumCount
	if totalPromos > len(activeListings) {
		featuredCount = len(activeListings) / 2
		premiumCount = len(activeListings) - featuredCount
	}

	now := time.Now()
	promotions := make([]promoSchema.ListingPromotion, 0, featuredCount+premiumCount)

	for i := 0; i < featuredCount; i++ {
		listing := activeListings[i]
		durationDays := utils.RandomChoice([]int{7, 14, 21, 30})
		startedAt := now.Add(-time.Duration(utils.RandomInt(0, durationDays/2)) * 24 * time.Hour)
		expiresAt := startedAt.Add(time.Duration(durationDays) * 24 * time.Hour)

		promotions = append(promotions, promoSchema.ListingPromotion{
			ID:              uuid.New(),
			ListingID:       listing.ID,
			OwnerID:         listing.OwnerID,
			Type:            "featured",
			Status:          "active",
			Amount:          utils.RandomInt64(5000, 25000),
			Currency:        "NGN",
			Duration:        durationDays,
			StartedAt:       &startedAt,
			ExpiresAt:       &expiresAt,
			BoostMultiplier: 1.5,
			IsIncluded:      utils.RandomBoolWithProbability(0.3),
			Metadata:        "{}",
			CreatedAt:       startedAt,
			UpdatedAt:       now,
		})
	}

	for i := featuredCount; i < featuredCount+premiumCount; i++ {
		listing := activeListings[i]
		durationDays := utils.RandomChoice([]int{14, 21, 30})
		startedAt := now.Add(-time.Duration(utils.RandomInt(0, durationDays/2)) * 24 * time.Hour)
		expiresAt := startedAt.Add(time.Duration(durationDays) * 24 * time.Hour)

		promotions = append(promotions, promoSchema.ListingPromotion{
			ID:              uuid.New(),
			ListingID:       listing.ID,
			OwnerID:         listing.OwnerID,
			Type:            "premium",
			Status:          "active",
			Amount:          utils.RandomInt64(15000, 50000),
			Currency:        "NGN",
			Duration:        durationDays,
			StartedAt:       &startedAt,
			ExpiresAt:       &expiresAt,
			BoostMultiplier: 2.0,
			IsIncluded:      utils.RandomBoolWithProbability(0.2),
			Metadata:        "{}",
			CreatedAt:       startedAt,
			UpdatedAt:       now,
		})
	}

	fmt.Printf("    Creating %d featured + %d premium promotions...\n", featuredCount, premiumCount)
	if err := ctx.DB.CreateInBatches(promotions, 50).Error; err != nil {
		return fmt.Errorf("failed to create promotions: %w", err)
	}

	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
