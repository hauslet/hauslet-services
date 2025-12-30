package schema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes" // Recommended for better JSONB support
	"gorm.io/gorm"
)

type ReviewTargetType string
type ReviewStatus string
type ModerationReason string

const (
	ReviewTargetListing    ReviewTargetType = "listing"
	ReviewTargetHost       ReviewTargetType = "host"
	ReviewTargetExperience ReviewTargetType = "experience"
)

const (
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusPublished ReviewStatus = "published"
	ReviewStatusHidden    ReviewStatus = "hidden"   // Hidden by admin
	ReviewStatusArchived  ReviewStatus = "archived" // Soft delete logic
	ReviewStatusStandoff  ReviewStatus = "standoff" // Waiting for other party
)

type Review struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	// Booking Context
	BookingID uuid.UUID `gorm:"type:uuid;not null;index:idx_review_unique,unique"`

	// Polymorphic Relations (Who is being reviewed?)
	// Note: We index TargetID/Type heavily for "Get all reviews for Listing X"
	TargetType ReviewTargetType `gorm:"type:varchar(20);not null;index:idx_review_unique,unique;index"`
	TargetID   uuid.UUID        `gorm:"type:uuid;not null;index:idx_review_unique,unique;index"`

	// The Author
	ReviewerID uuid.UUID `gorm:"type:uuid;not null;index:idx_review_unique,unique"`
	// Helper to filter reviews by users from specific countries
	ReviewerCountryCode string `gorm:"type:char(2);index"`

	// Content
	Rating   int    `gorm:"not null;check:rating >= 1 AND rating <= 5"` // DB constraint
	Title    string `gorm:"type:varchar(255)"`
	Body     string `gorm:"type:text;not null"`
	Language string `gorm:"type:varchar(10);default:'en'"` // e.g., 'en-US'

	// Sub-ratings (Cleanliness, Accuracy, etc.)
	// Using datatypes.JSON for better GORM integration
	SubRatings datatypes.JSON `gorm:"type:jsonb"`

	// State Management
	Status ReviewStatus `gorm:"type:varchar(20);not null;default:'standoff';index"`

	// Moderation
	IsReported       bool `gorm:"default:false;index"`
	ModerationReason *string

	// Response Mechanism (Has the host replied?)
	// Storing the ID allows efficient pre-fetching of the response without a separate query every time
	ResponseID *uuid.UUID `gorm:"type:uuid;index"`

	// Timestamps
	PublishedAt *time.Time `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// AfterCreate is triggered automatically after a new review is inserted.
func (r *Review) AfterCreate(tx *gorm.DB) (err error) {
	// 1. Define a variable to hold the counterparty's review (if it exists)
	var otherReview Review

	// 2. Search for a review with the same BookingID but a different ID
	// We use tx (the current transaction) to ensure data consistency.
	result := tx.Where("booking_id = ? AND id != ?", r.BookingID, r.ID).First(&otherReview)

	// 3. Check if the counter-review exists
	switch result.Error {
	case nil:
		// --- SCENARIO: MATCH FOUND (Both parties have reviewed) ---

		now := time.Now()

		// A. Update the OTHER review (User A's previous review) to Published
		if err := tx.Model(&otherReview).Updates(map[string]interface{}{
			"status":       ReviewStatusPublished,
			"published_at": now,
		}).Error; err != nil {
			return err
		}

		// B. Update the CURRENT review (User B's new review) to Published
		// Note: We modify the struct 'r' directly so the change is reflected in the caller's instance,
		// and we update the DB row.
		r.Status = ReviewStatusPublished
		r.PublishedAt = &now

		if err := tx.Model(r).Updates(map[string]interface{}{
			"status":       ReviewStatusPublished,
			"published_at": now,
		}).Error; err != nil {
			return err
		}
	case gorm.ErrRecordNotFound:
		// --- SCENARIO: NO MATCH (This is the first review) ---
		// Do nothing. The default status is already 'standoff' (or pending).
		// It will wait for the cron job or the second review.
	default:
		// Return any other actual database errors
		return result.Error
	}

	return nil
}

// Separate struct for Replies (Host responses)
type ReviewResponse struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	// The Unique Index here guarantees strictly ONE response per review
	ReviewID uuid.UUID `gorm:"type:uuid;not null;index:idx_one_response_per_review,unique"`

	AuthorID uuid.UUID `gorm:"type:uuid;not null"` // Usually the Host ID
	Body     string    `gorm:"type:text;not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListingStats acts as a "Read Model" for the Listing Page.
// It is updated asynchronously whenever a Review is published.
type ListingStats struct {
	// PK is the ListingID (One-to-One relationship with Listings table)
	ListingID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Aggregates
	AverageRating float64 `gorm:"type:decimal(3,2);not null;default:0.0"` // e.g., 4.87
	ReviewCount   int     `gorm:"not null;default:0"`

	// Breakdown of specific categories (Cleanliness, Value, etc.)
	// Structure: {"cleanliness": 4.8, "accuracy": 4.9, "check_in": 5.0}
	SubRatingAverages datatypes.JSON `gorm:"type:jsonb"`

	// Histogram data for the UI bars
	// Structure: {"5": 100, "4": 12, "3": 1, "2": 0, "1": 2}
	RatingDistribution datatypes.JSON `gorm:"type:jsonb"`

	// Metadata
	LastUpdatedAt time.Time `gorm:"index"` // Useful for debugging or cache expiry
}

// HostStats aggregates data across ALL of a host's listings.
// Used for the "Superhost" calculation and the Host Profile page.
type HostStats struct {
	// PK is the HostID (One-to-One relationship with Users table)
	HostID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Global Aggregates
	GlobalAverageRating float64 `gorm:"type:decimal(3,2);not null;default:0.0"`
	TotalReviewCount    int     `gorm:"not null;default:0"`

	// Superhost Logic usually requires looking at recent history
	ReviewCountLast365Days int `gorm:"not null;default:0"`

	LastUpdatedAt time.Time
}
