package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"hauslet/internal/modules/promotions/repository/schema"
)

// ListingPromotionRepository defines the interface for listing promotion data access
type ListingPromotionRepository interface {
	// Create creates a new promotion
	Create(ctx context.Context, promo *schema.ListingPromotion) error

	// GetByID retrieves a promotion by ID
	GetByID(ctx context.Context, id uuid.UUID) (*schema.ListingPromotion, error)

	// GetActiveByListing retrieves the active promotion for a listing
	GetActiveByListing(ctx context.Context, listingID uuid.UUID) (*schema.ListingPromotion, error)

	// ListByOwner lists promotions owned by a user
	ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*schema.ListingPromotion, error)

	// ListExpiring lists promotions expiring within specified hours
	ListExpiring(ctx context.Context, withinHours int) ([]*schema.ListingPromotion, error)

	// UpdateStatus updates the status of a promotion
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

	// Update updates a promotion
	Update(ctx context.Context, promo *schema.ListingPromotion) error

	// ListActivePromotions lists active promotions of a specific type
	ListActivePromotions(ctx context.Context, promoType string, limit int) ([]*schema.ListingPromotion, error)
}

// AgentSubscriptionRepository defines the interface for agent subscription data access
type AgentSubscriptionRepository interface {
	// Create creates a new subscription
	Create(ctx context.Context, sub *schema.AgentSubscription) error

	// GetByID retrieves a subscription by ID
	GetByID(ctx context.Context, id uuid.UUID) (*schema.AgentSubscription, error)

	// GetActiveByUser retrieves the active subscription for a user
	GetActiveByUser(ctx context.Context, userID uuid.UUID) (*schema.AgentSubscription, error)

	// Update updates a subscription
	Update(ctx context.Context, sub *schema.AgentSubscription) error

	// UpdateStatus updates the status of a subscription
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

	// ListDueForBilling lists subscriptions that are due for billing
	ListDueForBilling(ctx context.Context) ([]*schema.AgentSubscription, error)

	// ListByUser lists all subscriptions for a user (including inactive)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*schema.AgentSubscription, error)
}

// UsageTrackingRepository defines the interface for usage tracking data access
type UsageTrackingRepository interface {
	// Create creates a new usage tracking record
	Create(ctx context.Context, usage *schema.UsageTracking) error

	// GetCurrentPeriod retrieves the current period usage for a subscription
	GetCurrentPeriod(ctx context.Context, subscriptionID uuid.UUID) (*schema.UsageTracking, error)

	// Update updates a usage tracking record
	Update(ctx context.Context, usage *schema.UsageTracking) error

	// CreateNewPeriod creates a new usage period
	CreateNewPeriod(ctx context.Context, subscriptionID, userID uuid.UUID, start, end time.Time) (*schema.UsageTracking, error)

	// IncrementFeatured atomically increments featured usage
	IncrementFeatured(ctx context.Context, id uuid.UUID) error

	// IncrementPremium atomically increments premium usage
	IncrementPremium(ctx context.Context, id uuid.UUID) error

	// IncrementOpenHouse atomically increments open house usage
	IncrementOpenHouse(ctx context.Context, id uuid.UUID) error

	// IncrementPrivateShowing atomically increments private showing usage
	IncrementPrivateShowing(ctx context.Context, id uuid.UUID) error
}
