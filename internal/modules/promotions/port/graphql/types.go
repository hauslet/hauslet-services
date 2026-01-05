package graphql

import (
	"hauslet/internal/modules/promotions/domain"

	"github.com/google/uuid"
)

// CreatePromotionPayload contains the result of creating a paid promotion
type CreatePromotionPayload struct {
	Promotion  *domain.ListingPromotion
	PaymentURL string
	PaymentID  uuid.UUID
}

// CreatePromotionInput mirrors the GraphQL input for paid promotions.
type CreatePromotionInput struct {
	ListingID uuid.UUID            `json:"listingID"`
	Type      domain.PromotionType `json:"type"`
	Duration  int                  `json:"duration"`
}

// CreateIncludedPromotionInput mirrors the GraphQL input for included promotions.
type CreateIncludedPromotionInput struct {
	ListingID uuid.UUID            `json:"listingID"`
	Type      domain.PromotionType `json:"type"`
	Duration  int                  `json:"duration"`
}

// CreateSubscriptionPayload contains the result of creating a subscription
type CreateSubscriptionPayload struct {
	Subscription *domain.AgentSubscription
	PaymentURL   *string
	PaymentID    *uuid.UUID
}

// CreateSubscriptionInput mirrors the GraphQL input for subscriptions.
type CreateSubscriptionInput struct {
	PlanType        domain.PlanType     `json:"planType"`
	BillingCycle    domain.BillingCycle `json:"billingCycle"`
	StartTrial      bool                `json:"startTrial"`
	PaymentMethodID *uuid.UUID          `json:"paymentMethodID"`
}

// FeatureLimitCheckResult contains the result of checking a feature limit
type FeatureLimitCheckResult struct {
	Allowed   bool
	Limit     int
	Used      int
	Remaining int
}

// PlanLimits contains the limits for a subscription plan
type PlanLimits struct {
	MaxListings                int
	MaxPhotosPerListing        int
	MaxVideosPerListing        int
	FeaturedPromotionsPerMonth int
	PremiumPromotionsPerMonth  int
	OpenHousesPerMonth         int
	PrivateShowingsPerMonth    int
	Features                   []string
}
