package domain

import (
	"time"

	"github.com/google/uuid"
)

// ListingStats represents aggregated review statistics for a listing
type ListingStats struct {
	// PK is the ListingID (One-to-One relationship with Listings table)
	ListingID uuid.UUID

	// Aggregates
	AverageRating float64
	ReviewCount   int

	// Breakdown of specific categories (Cleanliness, Value, etc.)
	SubRatingAverages *SubRatings

	// Histogram data for the UI bars
	RatingDistribution *RatingDistribution

	// Metadata
	LastUpdatedAt time.Time
}

// HostStats represents aggregated review statistics for a host
type HostStats struct {
	// PK is the HostID (One-to-One relationship with Users table)
	HostID uuid.UUID

	// Global Aggregates
	GlobalAverageRating float64
	TotalReviewCount    int

	// Superhost Logic usually requires looking at recent history
	ReviewCountLast365Days int

	LastUpdatedAt time.Time
}

// RatingDistribution represents the histogram of star ratings
type RatingDistribution struct {
	FiveStarCount  int
	FourStarCount  int
	ThreeStarCount int
	TwoStarCount   int
	OneStarCount   int
}

// TotalReviews returns the total number of reviews in the distribution
func (r *RatingDistribution) TotalReviews() int {
	if r == nil {
		return 0
	}
	return r.FiveStarCount + r.FourStarCount + r.ThreeStarCount + r.TwoStarCount + r.OneStarCount
}

// GetPercentage returns the percentage of reviews for a given star rating
func (r *RatingDistribution) GetPercentage(stars int) float64 {
	if r == nil {
		return 0.0
	}

	total := r.TotalReviews()
	if total == 0 {
		return 0.0
	}

	var count int
	switch stars {
	case 5:
		count = r.FiveStarCount
	case 4:
		count = r.FourStarCount
	case 3:
		count = r.ThreeStarCount
	case 2:
		count = r.TwoStarCount
	case 1:
		count = r.OneStarCount
	default:
		return 0.0
	}

	return (float64(count) / float64(total)) * 100.0
}

// HasReviews returns true if there are any reviews
func (l *ListingStats) HasReviews() bool {
	return l.ReviewCount > 0
}

// IsHighlyRated returns true if the listing has an average rating >= 4.5
func (l *ListingStats) IsHighlyRated() bool {
	return l.AverageRating >= 4.5
}

// IsEligibleForSuperhost checks if host meets basic superhost criteria
// Typically: >= 10 reviews in last year and >= 4.8 average rating
func (h *HostStats) IsEligibleForSuperhost() bool {
	return h.ReviewCountLast365Days >= 10 && h.GlobalAverageRating >= 4.8
}

// HasReviews returns true if the host has received any reviews
func (h *HostStats) HasReviews() bool {
	return h.TotalReviewCount > 0
}
