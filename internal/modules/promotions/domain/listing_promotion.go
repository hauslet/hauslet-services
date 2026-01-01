package domain

import (
	"time"

	"github.com/google/uuid"
)

// ListingPromotion represents a promotional campaign for a listing
type ListingPromotion struct {
	ID        uuid.UUID `json:"id"`
	ListingID uuid.UUID `json:"listing_id"`
	OwnerID   uuid.UUID `json:"owner_id"`

	Type   PromotionType   `json:"type"`
	Status PromotionStatus `json:"status"`

	// Pricing
	Amount   int64  `json:"amount"`   // In minor units (kobo)
	Currency string `json:"currency"` // NGN
	Duration int    `json:"duration"` // Duration in days

	// Timestamps
	StartedAt *time.Time `json:"started_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Payment
	PaymentID *uuid.UUID `json:"payment_id,omitempty"`

	// Subscription Integration
	SubscriptionID *uuid.UUID `json:"subscription_id,omitempty"` // If from included quota
	IsIncluded     bool       `json:"is_included"`                // From subscription quota

	// Search boost configuration
	BoostMultiplier float64                `json:"boost_multiplier"` // Search ranking boost
	Metadata        map[string]interface{} `json:"metadata,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// IsActive returns true if the promotion is currently active
func (p *ListingPromotion) IsActive() bool {
	if p.Status != PromotionStatusActive {
		return false
	}
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().Before(*p.ExpiresAt)
}

// CanBeCancelled returns true if the promotion can be cancelled
func (p *ListingPromotion) CanBeCancelled() bool {
	return p.Status == PromotionStatusPending || p.Status == PromotionStatusActive
}

// Start activates the promotion and sets start/expiry times
func (p *ListingPromotion) Start() error {
	if p.Status != PromotionStatusPending {
		return ErrCannotCancelPromotion
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, p.Duration)

	p.Status = PromotionStatusActive
	p.StartedAt = &now
	p.ExpiresAt = &expiresAt
	p.UpdatedAt = now

	return nil
}

// Expire marks the promotion as expired
func (p *ListingPromotion) Expire() error {
	if p.Status != PromotionStatusActive {
		return ErrPromotionExpired
	}

	p.Status = PromotionStatusExpired
	p.UpdatedAt = time.Now()

	return nil
}

// Cancel marks the promotion as cancelled
func (p *ListingPromotion) Cancel() error {
	if !p.CanBeCancelled() {
		return ErrCannotCancelPromotion
	}

	p.Status = PromotionStatusCancelled
	p.UpdatedAt = time.Now()

	return nil
}

// CalculateSearchBoost returns the search ranking boost multiplier
func (p *ListingPromotion) CalculateSearchBoost() float64 {
	if !p.IsActive() {
		return 1.0 // No boost
	}
	return p.BoostMultiplier
}

// HasExpired returns true if the promotion has passed its expiry date
func (p *ListingPromotion) HasExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// DaysRemaining returns the number of days remaining in the promotion
func (p *ListingPromotion) DaysRemaining() int {
	if p.ExpiresAt == nil {
		return 0
	}
	duration := time.Until(*p.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}
