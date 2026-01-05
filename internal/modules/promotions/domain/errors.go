package domain

import "errors"

var (
	// Promotion errors
	ErrPromotionNotFound        = errors.New("promotion not found")
	ErrDuplicateActivePromotion = errors.New("listing already has active promotion")
	ErrInvalidPromotionType     = errors.New("invalid promotion type")
	ErrPromotionExpired         = errors.New("promotion has expired")
	ErrCannotCancelPromotion    = errors.New("cannot cancel promotion")
	ErrInvalidPromotionDuration = errors.New("invalid promotion duration")

	// Subscription errors
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrQuotaExceeded         = errors.New("subscription quota exceeded")
	ErrListingLimitReached   = errors.New("listing limit reached")
	ErrPhotoLimitReached     = errors.New("photo limit reached")
	ErrFeatureNotAvailable   = errors.New("feature not available in your plan")
	ErrInvalidBillingCycle   = errors.New("invalid billing cycle")
	ErrInvalidPlanType       = errors.New("invalid plan type")
	ErrSubscriptionNotActive = errors.New("subscription is not active")
	ErrCannotUpgrade         = errors.New("cannot upgrade subscription")
	ErrCannotDowngrade       = errors.New("cannot downgrade subscription")

	// Usage errors
	ErrUsageNotFound           = errors.New("usage tracking not found")
	ErrInvalidUsagePeriod      = errors.New("invalid usage period")
	ErrViewingEventsQuotaExceeded = errors.New("viewing events quota exceeded")
	ErrOpenHouseQuotaExceeded     = errors.New("open house quota exceeded")
	ErrPrivateShowingQuotaExceeded = errors.New("private showing quota exceeded")

	// Payment errors
	ErrPaymentRequired = errors.New("payment required for this operation")
	ErrInvalidPayment  = errors.New("invalid payment information")
)
