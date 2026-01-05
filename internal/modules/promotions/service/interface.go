package service

import (
	"context"

	"hauslet/internal/modules/promotions/domain"

	"github.com/google/uuid"
)

// CreatePromotionResult contains the result of creating a paid promotion
type CreatePromotionResult struct {
	Promotion  *domain.ListingPromotion
	PaymentURL string // URL for user to complete payment
	PaymentID  uuid.UUID
}

// CreateSubscriptionResult contains the result of creating a subscription
type CreateSubscriptionResult struct {
	Subscription *domain.AgentSubscription
	PaymentURL   string // URL for user to complete payment (empty for trial)
	PaymentID    *uuid.UUID
}

// PromotionService handles listing promotion operations
type PromotionService interface {
	// CreatePromotion creates a new promotion and initiates payment
	CreatePromotion(ctx context.Context, input CreatePromotionInput) (*CreatePromotionResult, error)

	// CreateIncludedPromotion creates a promotion from subscription quota (no payment)
	CreateIncludedPromotion(ctx context.Context, input CreateIncludedPromotionInput) (*domain.ListingPromotion, error)

	// StartPromotion activates a promotion (called after payment confirmation)
	StartPromotion(ctx context.Context, promoID uuid.UUID) error

	// CancelPromotion cancels a pending or active promotion
	CancelPromotion(ctx context.Context, promoID uuid.UUID) error

	// GetPromotion retrieves a promotion by ID
	GetPromotion(ctx context.Context, promoID uuid.UUID) (*domain.ListingPromotion, error)

	// ListUserPromotions lists promotions for a user
	ListUserPromotions(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.ListingPromotion, error)

	// GetActivePromotion gets the active promotion for a listing
	GetActivePromotion(ctx context.Context, listingID uuid.UUID) (*domain.ListingPromotion, error)

	// ExpirePromotions expires all promotions that have passed their expiry date (cron job)
	ExpirePromotions(ctx context.Context) error

	// GetFeaturedListings gets currently featured listings
	GetFeaturedListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error)

	// GetPremiumListings gets currently premium listings
	GetPremiumListings(ctx context.Context, limit int) ([]*domain.ListingPromotion, error)
}

// SubscriptionService handles agent subscription operations
type SubscriptionService interface {
	// CreateSubscription creates a new subscription and initiates payment (or starts trial)
	CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (*CreateSubscriptionResult, error)

	// UpgradeSubscription upgrades to a higher tier
	UpgradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan domain.PlanType) error

	// DowngradeSubscription downgrades to a lower tier
	DowngradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlan domain.PlanType) error

	// CancelSubscription cancels a subscription
	CancelSubscription(ctx context.Context, subscriptionID uuid.UUID) error

	// RenewSubscription renews a subscription after payment
	RenewSubscription(ctx context.Context, subscriptionID, paymentID uuid.UUID) error

	// HandlePaymentSuccess handles successful payment confirmation (webhooks, 3D Secure completion)
	// Completes pending upgrades when async payments succeed
	HandlePaymentSuccess(ctx context.Context, paymentID uuid.UUID) error

	// GetSubscription retrieves a subscription by ID
	GetSubscription(ctx context.Context, subscriptionID uuid.UUID) (*domain.AgentSubscription, error)

	// GetUserSubscription retrieves the active subscription for a user
	GetUserSubscription(ctx context.Context, userID uuid.UUID) (*domain.AgentSubscription, error)

	// ProcessBilling processes billing for due subscriptions (cron job)
	ProcessBilling(ctx context.Context) error

	// CanAddListing checks if user can add another listing
	CanAddListing(ctx context.Context, userID uuid.UUID) (bool, error)

	// CanAddPhotos checks if user can add photos to a listing
	CanAddPhotos(ctx context.Context, userID, listingID uuid.UUID, photoCount int) (bool, error)

	// CanUseFeature checks if user has access to a feature
	CanUseFeature(ctx context.Context, userID uuid.UUID, feature string) (bool, error)

	// CanUseIncludedPromotion checks if user can use an included promotion
	CanUseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType domain.PromotionType) (bool, error)

	// UseIncludedPromotion marks an included promotion as used
	UseIncludedPromotion(ctx context.Context, userID uuid.UUID, promoType domain.PromotionType) error

	// CanCreateOpenHouse checks if user can create an open house
	CanCreateOpenHouse(ctx context.Context, userID uuid.UUID) (bool, int, error)

	// CanCreatePrivateShowing checks if user can create a private showing
	CanCreatePrivateShowing(ctx context.Context, userID uuid.UUID) (bool, int, error)

	// UseOpenHouse marks an open house slot as used
	UseOpenHouse(ctx context.Context, userID uuid.UUID) error

	// UsePrivateShowing marks a private showing slot as used
	UsePrivateShowing(ctx context.Context, userID uuid.UUID) error

	// GetFeatureLimit gets the limit for a specific feature
	GetFeatureLimit(ctx context.Context, userID uuid.UUID, feature string) (int, error)
}

// UsageService tracks monthly quota usage
type UsageService interface {
	// GetCurrentUsage retrieves the current usage period for a subscription
	GetCurrentUsage(ctx context.Context, subscriptionID uuid.UUID) (*domain.UsageTracking, error)

	// GetOrCreateCurrentUsage gets or creates the current usage period
	GetOrCreateCurrentUsage(ctx context.Context, subscriptionID, userID uuid.UUID) (*domain.UsageTracking, error)

	// IncrementUsage increments usage for a specific type
	IncrementUsage(ctx context.Context, subscriptionID uuid.UUID, usageType domain.UsageType) error

	// ResetUsage creates a new usage period (called on billing cycle)
	ResetUsage(ctx context.Context, subscriptionID, userID uuid.UUID) error
}

// Input structs

// CreatePromotionInput contains the data needed to create a paid promotion
type CreatePromotionInput struct {
	ListingID  uuid.UUID
	OwnerID    uuid.UUID
	OwnerEmail string
	OwnerName  string
	Type       domain.PromotionType
	Duration   int // in days
}

// CreateIncludedPromotionInput contains the data needed to create an included promotion
type CreateIncludedPromotionInput struct {
	ListingID      uuid.UUID
	OwnerID        uuid.UUID
	Type           domain.PromotionType
	Duration       int // in days
	SubscriptionID uuid.UUID
}

// CreateSubscriptionInput contains the data needed to create a subscription
type CreateSubscriptionInput struct {
	UserID          uuid.UUID
	UserEmail       string
	UserName        string
	PlanType        domain.PlanType
	BillingCycle    domain.BillingCycle
	StartTrial      bool       // If true, starts trial period; if false, creates payment
	PaymentMethodID *uuid.UUID // Optional: link saved payment method for recurring billing
}
