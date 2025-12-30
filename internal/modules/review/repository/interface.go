package repository

import (
	"context"
	"time"

	"hauslet/internal/modules/review/repository/schema"

	"github.com/google/uuid"
)

// ReviewRepository handles the complex lifecycle of the review (Standoff -> Published).
type ReviewRepository interface {
	// --- Core CRUD ---
	Create(ctx context.Context, review *schema.Review) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Review, error)
	Update(ctx context.Context, review *schema.Review) error
	Delete(ctx context.Context, id uuid.UUID) error

	// --- Specialized Fetching ---
	List(ctx context.Context, filter ReviewFilter) ([]schema.Review, int64, error)
	GetByBookingAndReviewer(ctx context.Context, bookingID, reviewerID uuid.UUID) (*schema.Review, error)
	GetCounterpartReview(ctx context.Context, bookingID, currentReviewerID uuid.UUID) (*schema.Review, error)

	// --- Batch / Cron Operations ---
	PublishExpiredStandoffs(ctx context.Context, olderThan time.Time) (int64, error)

	// --- Moderation ---
	// MarkAsHidden hides a review (admin action) and logs the reason.
	MarkAsHidden(ctx context.Context, reviewID uuid.UUID, reason schema.ModerationReason) error
}

// ResponseRepository manages the Host's rebuttals.
type ResponseRepository interface {
	Create(ctx context.Context, response *schema.ReviewResponse) error
	GetByReviewID(ctx context.Context, reviewID uuid.UUID) (*schema.ReviewResponse, error)
	Update(ctx context.Context, response *schema.ReviewResponse) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// StatsRepository handles the heavy lifting for Listing pages.
// It prevents us from running "SELECT AVG(rating)" on every page load.
type StatsRepository interface {
	GetListingStats(ctx context.Context, listingID uuid.UUID) (*schema.ListingStats, error)
	RecalculateStats(ctx context.Context, listingID uuid.UUID) error
	GetHostStats(ctx context.Context, hostID uuid.UUID) (*schema.HostStats, error)
}

// --- Helper Structs for "Comprehensive" Interaction ---

// ReviewFilter defines how we query reviews for the UI.
type ReviewFilter struct {
	TargetID   uuid.UUID               // ListingID or HostID
	TargetType schema.ReviewTargetType // listing, host, experience

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy string // "newest", "highest_rated", "lowest_rated", "most_relevant"

	// Filters
	Rating      *int    // e.g., Show only 5-star reviews
	Language    *string // e.g., "en", "fr"
	OnlyVisible bool    // If true, excludes 'hidden' or 'archived' status
}

// ReviewDistribution represents the "histogram" often seen on Airbnb (e.g., 5 stars: 80%, 4 stars: 10%).
type ReviewDistribution struct {
	FiveStarCount  int
	FourStarCount  int
	ThreeStarCount int
	TwoStarCount   int
	OneStarCount   int
}
