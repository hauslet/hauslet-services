package seeders

import (
	"encoding/json"
	"fmt"
	"time"

	"hauslet/db/seeds/utils"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ReviewSeed struct {
	ID                  uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BookingID           uuid.UUID      `gorm:"type:uuid;not null;index:idx_review_unique,unique"`
	TargetType          string         `gorm:"type:varchar(20);not null;index:idx_review_unique,unique;index"`
	TargetID            uuid.UUID      `gorm:"type:uuid;not null;index:idx_review_unique,unique;index"`
	ReviewerID          uuid.UUID      `gorm:"type:uuid;not null;index:idx_review_unique,unique"`
	ReviewerCountryCode string         `gorm:"type:char(2);index"`
	Rating              int            `gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Title               string         `gorm:"type:varchar(255)"`
	Body                string         `gorm:"type:text;not null"`
	Language            string         `gorm:"type:varchar(10);default:'en'"`
	SubRatings          datatypes.JSON `gorm:"type:jsonb"`
	Status              string         `gorm:"type:varchar(20);not null;default:'standoff';index"`
	IsReported          bool           `gorm:"default:false;index"`
	PublishedAt         *time.Time     `gorm:"index"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (ReviewSeed) TableName() string {
	return "reviews"
}

// listingOwnerPair holds the listing ID and its owner ID
type listingOwnerPair struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
}

// SeedReview seeds reviews for existing published listings and users
func SeedReview(ctx *SeedContext) error {
	var listings []listingOwnerPair
	// Fetch active, published listings and their owner IDs
	if err := ctx.DB.Table("listings").
		Select("id, owner_id").
		Where("status = ? AND published = ? AND deleted_at IS NULL", "active", true).
		Scan(&listings).Error; err != nil {
		return fmt.Errorf("failed to fetch active listings: %w", err)
	}

	var userIDs []uuid.UUID
	// Fetch existing users
	if err := ctx.DB.Table("users").
		Select("id").
		Where("deleted_at IS NULL").
		Scan(&userIDs).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	if len(listings) == 0 {
		fmt.Println("    No active listings found — skipping reviews seeding")
		return nil
	}
	if len(userIDs) < 2 {
		fmt.Println("    Not enough users found — minimum 2 users required for reviews seeding")
		return nil
	}

	reviews := make([]ReviewSeed, 0)
	now := time.Now()

	for _, listing := range listings {
		reviewCount := utils.RandomInt(1, 4) // 1 to 4 reviews per listing

		for i := 0; i < reviewCount; i++ {
			// Find a user who is not the owner to act as guest
			var guestID uuid.UUID
			for {
				guestID = userIDs[utils.RandomInt(0, len(userIDs)-1)]
				if guestID != listing.OwnerID {
					break
				}
			}

			// Generate a fake booking ID to tie the reviews
			bookingID := uuid.New()
			publishedAt := now.Add(-time.Duration(utils.RandomInt(1, 60)) * 24 * time.Hour)

			star := weightedStarForAvgReview(4.5)

			subRatings := map[string]float64{
				"cleanliness":   float64(star),
				"accuracy":      float64(star),
				"check_in":      float64(star),
				"communication": float64(star),
				"location":      float64(star),
				"value":         float64(star),
			}
			subRatingsData, _ := json.Marshal(subRatings)

			// 1. Guest reviews the Listing
			reviews = append(reviews, ReviewSeed{
				ID:                  uuid.New(),
				BookingID:           bookingID,
				TargetType:          "listing",
				TargetID:            listing.ID,
				ReviewerID:          guestID,
				ReviewerCountryCode: randomCountry(),
				Rating:              star,
				Title:               generateReviewTitle(star),
				Body:                generateListingReviewBody(star),
				Language:            "en",
				SubRatings:          datatypes.JSON(subRatingsData),
				Status:              "published",
				PublishedAt:         &publishedAt,
				CreatedAt:           publishedAt.Add(-2 * time.Hour),
				UpdatedAt:           publishedAt.Add(-2 * time.Hour),
			})

			// 2. Guest reviews the Host (Listing Owner)
			reviews = append(reviews, ReviewSeed{
				ID:                  uuid.New(),
				BookingID:           bookingID,
				TargetType:          "host",
				TargetID:            listing.OwnerID,
				ReviewerID:          guestID,
				ReviewerCountryCode: randomCountry(),
				Rating:              star,
				Title:               "Great Host",
				Body:                generateHostReviewBody(star),
				Language:            "en",
				SubRatings:          datatypes.JSON(subRatingsData),
				Status:              "published",
				PublishedAt:         &publishedAt,
				CreatedAt:           publishedAt.Add(-2 * time.Hour),
				UpdatedAt:           publishedAt.Add(-2 * time.Hour),
			})

			// 3. Host reviews the Guest (TargetID represents User ID)
			reviews = append(reviews, ReviewSeed{
				ID:                  uuid.New(),
				BookingID:           bookingID,
				TargetType:          "host", // Host reviewing guest also falls under reviewing user
				TargetID:            guestID,
				ReviewerID:          listing.OwnerID,
				ReviewerCountryCode: randomCountry(),
				Rating:              5, // Hosts typically give 5 stars for decent guests
				Title:               "Excellent Guest",
				Body:                "Great communication and they left the place clean!",
				Language:            "en",
				SubRatings:          datatypes.JSON(subRatingsData),
				Status:              "published",
				PublishedAt:         &publishedAt,
				CreatedAt:           publishedAt.Add(-1 * time.Hour),
				UpdatedAt:           publishedAt.Add(-1 * time.Hour),
			})
		}
	}

	fmt.Printf("    Creating %d reviews...\n", len(reviews))
	if err := ctx.DB.CreateInBatches(reviews, 100).Error; err != nil {
		return fmt.Errorf("failed to create reviews: %w", err)
	}

	return nil
}

func weightedStarForAvgReview(avg float64) int {
	roll := utils.RandomInt(1, 100)
	switch {
	case avg >= 4.5:
		if roll <= 80 {
			return 5
		}
		if roll <= 95 {
			return 4
		}
		return 3
	default:
		if roll <= 50 {
			return 5
		}
		if roll <= 80 {
			return 4
		}
		return 3
	}
}

func randomCountry() string {
	countries := []string{"US", "GB", "NG", "DE", "FR", "CA"}
	return countries[utils.RandomInt(0, len(countries)-1)]
}

func generateReviewTitle(rating int) string {
	if rating >= 4 {
		titles := []string{"Excellent Stay!", "Highly Recommended", "Great location and value", "Wonderful experience"}
		return titles[utils.RandomInt(0, len(titles)-1)]
	} else if rating == 3 {
		return "It was okay, but could be better"
	}
	return "Disappointing experience"
}

func generateListingReviewBody(rating int) string {
	if rating >= 4 {
		return "We had a wonderful time. The place was clean, exactly as described, and the neighborhood is lovely. Would 100% come back!"
	} else if rating == 3 {
		return "The place was fine, but a few things could be improved. It served its purpose for the trip."
	}
	return "Would not recommend. Several issues with the stay and it wasn't as advertised."
}

func generateHostReviewBody(rating int) string {
	if rating >= 4 {
		return "The host was very responsive, friendly, and accommodating to our needs."
	}
	return "Host was hard to reach and unhelpful when we needed assistance."
}
