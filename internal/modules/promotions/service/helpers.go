package service

import (
	"fmt"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/promotions/domain"
)

// calculatePromotionPrice looks up the price for a promotion type and duration
func calculatePromotionPrice(cfg *config.PromotionYAMLConfig, promoType domain.PromotionType, durationDays int) (int64, error) {
	price, found := cfg.GetPromotionPrice(promoType.String(), durationDays)
	if !found {
		return 0, fmt.Errorf("no pricing found for promotion type %s with duration %d days", promoType, durationDays)
	}
	return price, nil
}

// calculateBoostMultiplier returns the search boost multiplier for a promotion type
func calculateBoostMultiplier(cfg *config.PromotionYAMLConfig, promoType domain.PromotionType) float64 {
	promo, exists := cfg.ListingPromotions[promoType.String()]
	if !exists {
		return 1.0 // No boost
	}
	return promo.Placement.SearchBoostMultiplier
}

// calculateProrationAmount calculates proration for subscription upgrade/downgrade
func calculateProrationAmount(oldAmount, newAmount int64, daysRemaining, totalDays int) int64 {
	if daysRemaining <= 0 || totalDays <= 0 {
		return newAmount
	}

	// Calculate unused portion of old subscription
	unusedAmount := (oldAmount * int64(daysRemaining)) / int64(totalDays)

	// Calculate prorated amount for new subscription
	proratedNewAmount := (newAmount * int64(daysRemaining)) / int64(totalDays)

	// Return the difference (could be positive for upgrade, negative for downgrade)
	return proratedNewAmount - unusedAmount
}

// loadPlanConfig loads subscription plan configuration
func loadPlanConfig(cfg *config.PromotionYAMLConfig, planType domain.PlanType, billingCycle domain.BillingCycle) (*PlanConfig, error) {
	plan, exists := cfg.SubscriptionPlans[planType.String()]
	if !exists || !plan.Enabled {
		return nil, fmt.Errorf("plan %s not found or disabled", planType)
	}

	limits, _ := cfg.GetPlanLimits(planType.String())
	features, _ := cfg.GetPlanFeatures(planType.String())
	price, found := cfg.GetSubscriptionPrice(planType.String(), billingCycle.String())
	if !found {
		return nil, fmt.Errorf("no pricing found for plan %s with billing cycle %s", planType, billingCycle)
	}

	return &PlanConfig{
		PlanType:                        planType,
		Amount:                          price,
		Currency:                        cfg.Currency.Code,
		MaxListings:                     limits.MaxListings,
		MaxPhotosPerListing:             limits.MaxPhotosPerListing,
		MaxVirtualTours:                 limits.MaxVirtualTours,
		IncludedFeaturedPerMonth:        limits.IncludedFeaturedPerMonth,
		IncludedPremiumPerMonth:         limits.IncludedPremiumPerMonth,
		IncludedOpenHousesPerMonth:      limits.IncludedOpenHousesPerMonth,
		IncludedPrivateShowingsPerMonth: limits.IncludedPrivateShowingsPerMonth,
		Features:                        features,
	}, nil
}

// PlanConfig holds the resolved configuration for a subscription plan
type PlanConfig struct {
	PlanType                        domain.PlanType
	Amount                          int64
	Currency                        string
	MaxListings                     int
	MaxPhotosPerListing             int
	MaxVirtualTours                 int
	IncludedFeaturedPerMonth        int
	IncludedPremiumPerMonth         int
	IncludedOpenHousesPerMonth      int
	IncludedPrivateShowingsPerMonth int
	Features                        map[string]bool
}

// validatePromotionDuration checks if the duration is valid for a promotion type
func validatePromotionDuration(cfg *config.PromotionYAMLConfig, promoType domain.PromotionType, durationDays int) error {
	promo, exists := cfg.ListingPromotions[promoType.String()]
	if !exists {
		return fmt.Errorf("promotion type %s not found", promoType)
	}

	// Check if this duration has pricing configured
	_, found := cfg.GetPromotionPrice(promoType.String(), durationDays)
	if !found {
		availableDurations := make([]string, 0, len(promo.Pricing))
		for duration := range promo.Pricing {
			availableDurations = append(availableDurations, duration)
		}
		return fmt.Errorf("invalid duration %d days for %s promotion. Available: %v", durationDays, promoType, availableDurations)
	}

	return nil
}

// getCurrentBillingPeriod calculates the current billing period start and end
func getCurrentBillingPeriod(cfg *config.PromotionYAMLConfig) (time.Time, time.Time) {
	now := time.Now()
	resetDay := cfg.Billing.UsageResetDay
	if resetDay == 0 {
		resetDay = 1 // Default to 1st of month
	}

	// Calculate period start
	year, month, _ := now.Date()
	periodStart := time.Date(year, month, resetDay, 0, 0, 0, 0, now.Location())

	// If we're before the reset day this month, period started last month
	if now.Day() < resetDay {
		periodStart = periodStart.AddDate(0, -1, 0)
	}

	// Period end is the day before next reset
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	return periodStart, periodEnd
}

// calculateNextBillingDate calculates the next billing date based on billing cycle
func calculateNextBillingDate(currentDate time.Time, billingCycle domain.BillingCycle) time.Time {
	switch billingCycle {
	case domain.BillingCycleMonthly:
		return currentDate.AddDate(0, 1, 0)
	case domain.BillingCycleYearly:
		return currentDate.AddDate(1, 0, 0)
	default:
		return currentDate.AddDate(0, 1, 0) // Default to monthly
	}
}

// calculateTrialEndDate calculates when the trial period ends
func calculateTrialEndDate(cfg *config.PromotionYAMLConfig, startDate time.Time) time.Time {
	trialDays := cfg.Billing.TrialPeriodDays
	if trialDays == 0 {
		trialDays = 14 // Default to 14 days
	}
	return startDate.AddDate(0, 0, trialDays)
}

// getDaysInBillingCycle returns the number of days in a billing cycle
func getDaysInBillingCycle(billingCycle domain.BillingCycle, startDate time.Time) int {
	switch billingCycle {
	case domain.BillingCycleMonthly:
		// Calculate days in the month
		nextMonth := startDate.AddDate(0, 1, 0)
		return int(nextMonth.Sub(startDate).Hours() / 24)
	case domain.BillingCycleYearly:
		// Calculate days in the year
		nextYear := startDate.AddDate(1, 0, 0)
		return int(nextYear.Sub(startDate).Hours() / 24)
	default:
		return 30 // Default estimate
	}
}

// isFeatureEnabled checks if a feature is enabled in the feature map
func isFeatureEnabled(features map[string]bool, featureName string) bool {
	if features == nil {
		return false
	}
	enabled, exists := features[featureName]
	return exists && enabled
}

// getFreeTierLimit returns the limit for free tier users
func getFreeTierLimit(cfg *config.PromotionYAMLConfig, limitType string) int {
	switch limitType {
	case "max_listings":
		return cfg.FreeTier.MaxListings
	case "max_photos_per_listing":
		return cfg.FreeTier.MaxPhotosPerListing
	case "max_virtual_tours":
		return cfg.FreeTier.MaxVirtualTours
	default:
		return 0
	}
}

// getFreeTierFeature returns whether a feature is enabled for free tier
func getFreeTierFeature(cfg *config.PromotionYAMLConfig, featureName string) bool {
	switch featureName {
	case "analytics_enabled":
		return cfg.FreeTier.AnalyticsEnabled
	case "api_access_enabled":
		return cfg.FreeTier.APIAccessEnabled
	case "bulk_upload_enabled":
		return cfg.FreeTier.BulkUploadEnabled
	case "priority_support":
		return cfg.FreeTier.PrioritySupport
	case "lead_management_enabled":
		return cfg.FreeTier.LeadManagementEnabled
	case "viewing_events_enabled":
		return cfg.FreeTier.ViewingEventsEnabled
	case "open_house_events_enabled":
		return cfg.FreeTier.OpenHouseEventsEnabled
	case "private_showings_enabled":
		return cfg.FreeTier.PrivateShowingsEnabled
	default:
		return false
	}
}
