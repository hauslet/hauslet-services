package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/leads/domain"
	"hauslet/internal/modules/leads/repository"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxLeadsPerEmailPerDay is the maximum number of leads allowed per email address per day
	MaxLeadsPerEmailPerDay = 3

	// MaxLeadsPerIPPerDay is the maximum number of leads allowed per IP address per day
	MaxLeadsPerIPPerDay = 10

	// MaxLeadsPerListingPerDay is the maximum number of leads allowed per listing per email per day
	MaxLeadsPerListingPerDay = 5

	// RateLimitWindow is the time window for rate limiting (24 hours)
	RateLimitWindow = 24 * time.Hour
)

// RateLimiter handles database-based rate limiting for lead submissions
type RateLimiter struct {
	repo repository.LeadRepository
}

// NewRateLimiter creates a new instance of RateLimiter
func NewRateLimiter(repo repository.LeadRepository) *RateLimiter {
	return &RateLimiter{repo: repo}
}

// CheckRateLimit checks if a lead submission is within rate limits
// Returns an error if any rate limit is exceeded
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, email, ipAddress string, listingID uuid.UUID) error {
	since := time.Now().Add(-RateLimitWindow)

	// Check email rate limit
	emailCount, err := rl.repo.CountLeadsByEmail(ctx, email, since)
	if err != nil {
		return fmt.Errorf("failed to check email rate limit: %w", err)
	}
	if emailCount >= MaxLeadsPerEmailPerDay {
		return domain.ErrEmailRateLimitReached
	}

	// Check IP rate limit (if IP address is provided)
	if ipAddress != "" {
		ipCount, err := rl.repo.CountLeadsByIP(ctx, ipAddress, since)
		if err != nil {
			return fmt.Errorf("failed to check IP rate limit: %w", err)
		}
		if ipCount >= MaxLeadsPerIPPerDay {
			return domain.ErrIPRateLimitReached
		}
	}

	// Check per-listing rate limit (email + listing combination)
	listingCount, err := rl.repo.CountLeadsByListing(ctx, listingID, email, since)
	if err != nil {
		return fmt.Errorf("failed to check listing rate limit: %w", err)
	}
	if listingCount >= MaxLeadsPerListingPerDay {
		return domain.ErrListingRateLimitReached
	}

	return nil
}

// GetRateLimitStatus returns the current rate limit status for an email/IP combination
// Returns the number of leads submitted and the limits for each category
func (rl *RateLimiter) GetRateLimitStatus(ctx context.Context, email, ipAddress string, listingID uuid.UUID) (*RateLimitStatus, error) {
	since := time.Now().Add(-RateLimitWindow)

	status := &RateLimitStatus{
		EmailLimit:   MaxLeadsPerEmailPerDay,
		IPLimit:      MaxLeadsPerIPPerDay,
		ListingLimit: MaxLeadsPerListingPerDay,
	}

	// Get email count
	emailCount, err := rl.repo.CountLeadsByEmail(ctx, email, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get email count: %w", err)
	}
	status.EmailCount = int(emailCount)
	status.EmailRemaining = MaxLeadsPerEmailPerDay - int(emailCount)
	if status.EmailRemaining < 0 {
		status.EmailRemaining = 0
	}

	// Get IP count (if IP address is provided)
	if ipAddress != "" {
		ipCount, err := rl.repo.CountLeadsByIP(ctx, ipAddress, since)
		if err != nil {
			return nil, fmt.Errorf("failed to get IP count: %w", err)
		}
		status.IPCount = int(ipCount)
		status.IPRemaining = MaxLeadsPerIPPerDay - int(ipCount)
		if status.IPRemaining < 0 {
			status.IPRemaining = 0
		}
	}

	// Get listing count
	listingCount, err := rl.repo.CountLeadsByListing(ctx, listingID, email, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get listing count: %w", err)
	}
	status.ListingCount = int(listingCount)
	status.ListingRemaining = MaxLeadsPerListingPerDay - int(listingCount)
	if status.ListingRemaining < 0 {
		status.ListingRemaining = 0
	}

	return status, nil
}

// RateLimitStatus represents the current rate limit status
type RateLimitStatus struct {
	EmailCount      int
	EmailLimit      int
	EmailRemaining  int
	IPCount         int
	IPLimit         int
	IPRemaining     int
	ListingCount    int
	ListingLimit    int
	ListingRemaining int
}
