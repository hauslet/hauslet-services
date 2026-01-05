package schema

import (
	"hauslet/internal/modules/promotions/domain"
)

// MapListingPromotionToSchema converts domain ListingPromotion to schema
func MapListingPromotionToSchema(d *domain.ListingPromotion) (*ListingPromotion, error) {
	if d == nil {
		return nil, nil
	}

	metadata, err := domain.MarshalMetadata(d.Metadata)
	if err != nil {
		return nil, err
	}

	return &ListingPromotion{
		ID:              d.ID,
		ListingID:       d.ListingID,
		OwnerID:         d.OwnerID,
		Type:            d.Type.String(),
		Status:          d.Status.String(),
		Amount:          d.Amount,
		Currency:        d.Currency,
		Duration:        d.Duration,
		StartedAt:       d.StartedAt,
		ExpiresAt:       d.ExpiresAt,
		PaymentID:       d.PaymentID,
		SubscriptionID:  d.SubscriptionID,
		IsIncluded:      d.IsIncluded,
		BoostMultiplier: d.BoostMultiplier,
		Metadata:        metadata,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		DeletedAt:       d.DeletedAt,
	}, nil
}

// MapListingPromotionFromSchema converts schema ListingPromotion to domain
func MapListingPromotionFromSchema(s *ListingPromotion) (*domain.ListingPromotion, error) {
	if s == nil {
		return nil, nil
	}

	metadata, err := domain.UnmarshalMetadata(s.Metadata)
	if err != nil {
		return nil, err
	}

	return &domain.ListingPromotion{
		ID:              s.ID,
		ListingID:       s.ListingID,
		OwnerID:         s.OwnerID,
		Type:            domain.PromotionType(s.Type),
		Status:          domain.PromotionStatus(s.Status),
		Amount:          s.Amount,
		Currency:        s.Currency,
		Duration:        s.Duration,
		StartedAt:       s.StartedAt,
		ExpiresAt:       s.ExpiresAt,
		PaymentID:       s.PaymentID,
		SubscriptionID:  s.SubscriptionID,
		IsIncluded:      s.IsIncluded,
		BoostMultiplier: s.BoostMultiplier,
		Metadata:        metadata,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
		DeletedAt:       s.DeletedAt,
	}, nil
}

// MapAgentSubscriptionToSchema converts domain AgentSubscription to schema
func MapAgentSubscriptionToSchema(d *domain.AgentSubscription) (*AgentSubscription, error) {
	if d == nil {
		return nil, nil
	}

	features, err := domain.MarshalFeatures(d.Features)
	if err != nil {
		return nil, err
	}

	var pendingPlanType *string
	if d.PendingPlanType != nil {
		planTypeStr := d.PendingPlanType.String()
		pendingPlanType = &planTypeStr
	}

	return &AgentSubscription{
		ID:                              d.ID,
		UserID:                          d.UserID,
		UserEmail:                       d.UserEmail,
		UserName:                        d.UserName,
		PlanType:                        d.PlanType.String(),
		Status:                          d.Status.String(),
		BillingCycle:                    d.BillingCycle.String(),
		Amount:                          d.Amount,
		Currency:                        d.Currency,
		NextBillingDate:                 d.NextBillingDate,
		TrialEndsAt:                     d.TrialEndsAt,
		StartedAt:                       d.StartedAt,
		CancelledAt:                     d.CancelledAt,
		ExpiredAt:                       d.ExpiredAt,
		MaxListings:                     d.MaxListings,
		MaxPhotosPerListing:             d.MaxPhotosPerListing,
		MaxVirtualTours:                 d.MaxVirtualTours,
		IncludedFeaturedPerMonth:        d.IncludedFeaturedPerMonth,
		IncludedPremiumPerMonth:         d.IncludedPremiumPerMonth,
		IncludedOpenHousesPerMonth:      d.IncludedOpenHousesPerMonth,
		IncludedPrivateShowingsPerMonth: d.IncludedPrivateShowingsPerMonth,
		Features:                        features,
		PendingPlanType:                 pendingPlanType,
		PendingPlanScheduledAt:          d.PendingPlanScheduledAt,
		PaymentMethodID:                 d.PaymentMethodID,
		CreatedAt:                       d.CreatedAt,
		UpdatedAt:                       d.UpdatedAt,
		DeletedAt:                       d.DeletedAt,
	}, nil
}

// MapAgentSubscriptionFromSchema converts schema AgentSubscription to domain
func MapAgentSubscriptionFromSchema(s *AgentSubscription) (*domain.AgentSubscription, error) {
	if s == nil {
		return nil, nil
	}

	features, err := domain.UnmarshalFeatures(s.Features)
	if err != nil {
		return nil, err
	}

	var pendingPlanType *domain.PlanType
	if s.PendingPlanType != nil {
		planType := domain.PlanType(*s.PendingPlanType)
		pendingPlanType = &planType
	}

	return &domain.AgentSubscription{
		ID:                              s.ID,
		UserID:                          s.UserID,
		UserEmail:                       s.UserEmail,
		UserName:                        s.UserName,
		PlanType:                        domain.PlanType(s.PlanType),
		Status:                          domain.SubscriptionStatus(s.Status),
		BillingCycle:                    domain.BillingCycle(s.BillingCycle),
		Amount:                          s.Amount,
		Currency:                        s.Currency,
		NextBillingDate:                 s.NextBillingDate,
		TrialEndsAt:                     s.TrialEndsAt,
		StartedAt:                       s.StartedAt,
		CancelledAt:                     s.CancelledAt,
		ExpiredAt:                       s.ExpiredAt,
		MaxListings:                     s.MaxListings,
		MaxPhotosPerListing:             s.MaxPhotosPerListing,
		MaxVirtualTours:                 s.MaxVirtualTours,
		IncludedFeaturedPerMonth:        s.IncludedFeaturedPerMonth,
		IncludedPremiumPerMonth:         s.IncludedPremiumPerMonth,
		IncludedOpenHousesPerMonth:      s.IncludedOpenHousesPerMonth,
		IncludedPrivateShowingsPerMonth: s.IncludedPrivateShowingsPerMonth,
		Features:                        features,
		PendingPlanType:                 pendingPlanType,
		PendingPlanScheduledAt:          s.PendingPlanScheduledAt,
		PaymentMethodID:                 s.PaymentMethodID,
		CreatedAt:                       s.CreatedAt,
		UpdatedAt:                       s.UpdatedAt,
		DeletedAt:                       s.DeletedAt,
	}, nil
}

// MapUsageTrackingToSchema converts domain UsageTracking to schema
func MapUsageTrackingToSchema(d *domain.UsageTracking) *UsageTracking {
	if d == nil {
		return nil
	}

	return &UsageTracking{
		ID:                  d.ID,
		SubscriptionID:      d.SubscriptionID,
		UserID:              d.UserID,
		PeriodStart:         d.PeriodStart,
		PeriodEnd:           d.PeriodEnd,
		FeaturedUsed:        d.FeaturedUsed,
		PremiumUsed:         d.PremiumUsed,
		OpenHousesUsed:      d.OpenHousesUsed,
		PrivateShowingsUsed: d.PrivateShowingsUsed,
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}

// MapUsageTrackingFromSchema converts schema UsageTracking to domain
func MapUsageTrackingFromSchema(s *UsageTracking) *domain.UsageTracking {
	if s == nil {
		return nil
	}

	return &domain.UsageTracking{
		ID:                  s.ID,
		SubscriptionID:      s.SubscriptionID,
		UserID:              s.UserID,
		PeriodStart:         s.PeriodStart,
		PeriodEnd:           s.PeriodEnd,
		FeaturedUsed:        s.FeaturedUsed,
		PremiumUsed:         s.PremiumUsed,
		OpenHousesUsed:      s.OpenHousesUsed,
		PrivateShowingsUsed: s.PrivateShowingsUsed,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

// Batch mapping functions for slices

// MapListingPromotionsFromSchema converts multiple schemas to domain
func MapListingPromotionsFromSchema(schemas []*ListingPromotion) ([]*domain.ListingPromotion, error) {
	if schemas == nil {
		return nil, nil
	}

	result := make([]*domain.ListingPromotion, len(schemas))
	for i, s := range schemas {
		d, err := MapListingPromotionFromSchema(s)
		if err != nil {
			return nil, err
		}
		result[i] = d
	}
	return result, nil
}

// MapAgentSubscriptionsFromSchema converts multiple schemas to domain
func MapAgentSubscriptionsFromSchema(schemas []*AgentSubscription) ([]*domain.AgentSubscription, error) {
	if schemas == nil {
		return nil, nil
	}

	result := make([]*domain.AgentSubscription, len(schemas))
	for i, s := range schemas {
		d, err := MapAgentSubscriptionFromSchema(s)
		if err != nil {
			return nil, err
		}
		result[i] = d
	}
	return result, nil
}

// MapUsageTrackingsFromSchema converts multiple schemas to domain
func MapUsageTrackingsFromSchema(schemas []*UsageTracking) []*domain.UsageTracking {
	if schemas == nil {
		return nil
	}

	result := make([]*domain.UsageTracking, len(schemas))
	for i, s := range schemas {
		result[i] = MapUsageTrackingFromSchema(s)
	}
	return result
}
