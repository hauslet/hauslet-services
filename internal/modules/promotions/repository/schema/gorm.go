package schema

import (
	"time"

	"github.com/google/uuid"
)

// ListingPromotion represents the GORM schema for listing promotions
type ListingPromotion struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ListingID uuid.UUID  `gorm:"type:uuid;not null;index:idx_listing_promo"`
	OwnerID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Type      string     `gorm:"type:varchar(50);not null;index"`
	Status    string     `gorm:"type:varchar(20);not null;index:idx_promo_status"`
	Amount    int64      `gorm:"not null"`
	Currency  string     `gorm:"type:varchar(3);not null"`
	Duration  int        `gorm:"not null"`
	StartedAt *time.Time `gorm:"index:idx_promo_active"`
	ExpiresAt *time.Time `gorm:"index:idx_promo_active"`

	PaymentID      *uuid.UUID `gorm:"type:uuid;index"`
	SubscriptionID *uuid.UUID `gorm:"type:uuid;index"`
	IsIncluded     bool       `gorm:"not null;default:false"`

	BoostMultiplier float64 `gorm:"not null;default:1.0"`
	Metadata        string  `gorm:"type:jsonb"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// TableName sets the table name for ListingPromotion
func (ListingPromotion) TableName() string {
	return "listing_promotions"
}

// AgentSubscription represents the GORM schema for agent subscriptions
type AgentSubscription struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_user_subscription"`
	UserEmail string     `gorm:"type:varchar(255);not null"`
	UserName  string     `gorm:"type:varchar(255);not null"`
	PlanType  string     `gorm:"type:varchar(50);not null"`
	Status    string     `gorm:"type:varchar(20);not null;index:idx_subscription_status"`

	BillingCycle    string     `gorm:"type:varchar(20);not null"`
	Amount          int64      `gorm:"not null"`
	Currency        string     `gorm:"type:varchar(3);not null"`
	NextBillingDate *time.Time `gorm:"index:idx_billing_due"`

	TrialEndsAt *time.Time
	StartedAt   time.Time `gorm:"not null"`
	CancelledAt *time.Time
	ExpiredAt   *time.Time

	MaxListings         int `gorm:"not null"`
	MaxPhotosPerListing int `gorm:"not null"`
	MaxVirtualTours     int `gorm:"not null"`

	IncludedFeaturedPerMonth        int `gorm:"not null"`
	IncludedPremiumPerMonth         int `gorm:"not null"`
	IncludedOpenHousesPerMonth      int `gorm:"not null;default:0"`
	IncludedPrivateShowingsPerMonth int `gorm:"not null;default:0"`

	Features string `gorm:"type:jsonb"`

	// Pending Plan Changes (Option A)
	PendingPlanType        *string    `gorm:"type:varchar(50)"`
	PendingPlanScheduledAt *time.Time

	// Payment Method (for recurring billing)
	PaymentMethodID *uuid.UUID `gorm:"type:uuid;index:idx_subscription_payment_method"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// TableName sets the table name for AgentSubscription
func (AgentSubscription) TableName() string {
	return "agent_subscriptions"
}

// UsageTracking represents the GORM schema for usage tracking
type UsageTracking struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SubscriptionID uuid.UUID `gorm:"type:uuid;not null;index:idx_usage_subscription"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index"`

	PeriodStart time.Time `gorm:"not null;index:idx_usage_period"`
	PeriodEnd   time.Time `gorm:"not null;index:idx_usage_period"`

	FeaturedUsed int `gorm:"not null;default:0"`
	PremiumUsed  int `gorm:"not null;default:0"`

	OpenHousesUsed      int `gorm:"not null;default:0"`
	PrivateShowingsUsed int `gorm:"not null;default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName sets the table name for UsageTracking
func (UsageTracking) TableName() string {
	return "usage_trackings"
}
