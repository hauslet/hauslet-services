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

// CreateSubscriptionPayload contains the result of creating a subscription
type CreateSubscriptionPayload struct {
	Subscription *domain.AgentSubscription
	PaymentURL   *string
	PaymentID    *uuid.UUID
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
	MaxListings               int
	MaxPhotosPerListing       int
	MaxVideosPerListing       int
	FeaturedPromotionsPerMonth int
	PremiumPromotionsPerMonth  int
	OpenHousesPerMonth         int
	PrivateShowingsPerMonth    int
	Features                   []string
}
