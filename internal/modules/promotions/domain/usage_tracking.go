package domain

import (
	"time"

	"github.com/google/uuid"
)

// UsageTracking tracks monthly usage of included promotions and viewing events
type UsageTracking struct {
	ID             uuid.UUID `json:"id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	UserID         uuid.UUID `json:"user_id"`

	// Period tracking
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`

	// Promotion usage counts
	FeaturedUsed int `json:"featured_used"`
	PremiumUsed  int `json:"premium_used"`

	// Viewing events usage counts
	OpenHousesUsed      int `json:"open_houses_used"`
	PrivateShowingsUsed int `json:"private_showings_used"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CanUseFeatured checks if user can use another featured promotion
func (u *UsageTracking) CanUseFeatured(limit int) bool {
	if limit == -1 {
		return true // Unlimited
	}
	return u.FeaturedUsed < limit
}

// CanUsePremium checks if user can use another premium promotion
func (u *UsageTracking) CanUsePremium(limit int) bool {
	if limit == -1 {
		return true // Unlimited
	}
	return u.PremiumUsed < limit
}

// CanCreateOpenHouse checks if user can create another open house
func (u *UsageTracking) CanCreateOpenHouse(limit int) bool {
	if limit == -1 {
		return true // Unlimited
	}
	return u.OpenHousesUsed < limit
}

// CanCreatePrivateShowing checks if user can create another private showing
func (u *UsageTracking) CanCreatePrivateShowing(limit int) bool {
	if limit == -1 {
		return true // Unlimited
	}
	return u.PrivateShowingsUsed < limit
}

// IncrementFeatured increments the featured promotion usage counter
func (u *UsageTracking) IncrementFeatured() error {
	u.FeaturedUsed++
	u.UpdatedAt = time.Now()
	return nil
}

// IncrementPremium increments the premium promotion usage counter
func (u *UsageTracking) IncrementPremium() error {
	u.PremiumUsed++
	u.UpdatedAt = time.Now()
	return nil
}

// IncrementOpenHouse increments the open house usage counter
func (u *UsageTracking) IncrementOpenHouse() error {
	u.OpenHousesUsed++
	u.UpdatedAt = time.Now()
	return nil
}

// IncrementPrivateShowing increments the private showing usage counter
func (u *UsageTracking) IncrementPrivateShowing() error {
	u.PrivateShowingsUsed++
	u.UpdatedAt = time.Now()
	return nil
}

// IsCurrentPeriod checks if the tracking period is current
func (u *UsageTracking) IsCurrentPeriod() bool {
	now := time.Now()
	return now.After(u.PeriodStart) && now.Before(u.PeriodEnd)
}

// Reset resets all usage counters for a new period
func (u *UsageTracking) Reset(periodStart, periodEnd time.Time) {
	u.PeriodStart = periodStart
	u.PeriodEnd = periodEnd
	u.FeaturedUsed = 0
	u.PremiumUsed = 0
	u.OpenHousesUsed = 0
	u.PrivateShowingsUsed = 0
	u.UpdatedAt = time.Now()
}

// GetRemainingFeatured returns the number of remaining featured promotions
func (u *UsageTracking) GetRemainingFeatured(limit int) int {
	if limit == -1 {
		return -1 // Unlimited
	}
	remaining := limit - u.FeaturedUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetRemainingPremium returns the number of remaining premium promotions
func (u *UsageTracking) GetRemainingPremium(limit int) int {
	if limit == -1 {
		return -1 // Unlimited
	}
	remaining := limit - u.PremiumUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetRemainingOpenHouses returns the number of remaining open houses
func (u *UsageTracking) GetRemainingOpenHouses(limit int) int {
	if limit == -1 {
		return -1 // Unlimited
	}
	remaining := limit - u.OpenHousesUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetRemainingPrivateShowings returns the number of remaining private showings
func (u *UsageTracking) GetRemainingPrivateShowings(limit int) int {
	if limit == -1 {
		return -1 // Unlimited
	}
	remaining := limit - u.PrivateShowingsUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}
