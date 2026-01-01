package config

import "fmt"

// PromotionYAMLConfig defines the promotion and subscription system configuration
type PromotionYAMLConfig struct {
	Currency            PromotionCurrencyConfig            `yaml:"currency"`
	FreeTier            FreeTierConfig                     `yaml:"free_tier"`
	ListingPromotions   map[string]ListingPromotionConfig  `yaml:"listing_promotions"`
	SubscriptionPlans   map[string]SubscriptionPlanConfig  `yaml:"subscription_plans"`
	Addons              map[string]AddonConfig             `yaml:"addons"`
	Billing             BillingConfig                      `yaml:"billing"`
	Analytics           AnalyticsConfig                    `yaml:"analytics"`
	RateLimits          RateLimitsConfig                   `yaml:"rate_limits"`
	Notifications       PromotionNotificationsConfig       `yaml:"notifications"`
}

// PromotionCurrencyConfig defines currency settings
type PromotionCurrencyConfig struct {
	Code      string `yaml:"code"`
	MinorUnit int64  `yaml:"minor_unit"`
}

// FreeTierConfig defines free tier limits
type FreeTierConfig struct {
	MaxListings                int  `yaml:"max_listings"`
	MaxPhotosPerListing        int  `yaml:"max_photos_per_listing"`
	MaxVirtualTours            int  `yaml:"max_virtual_tours"`
	AnalyticsEnabled           bool `yaml:"analytics_enabled"`
	APIAccessEnabled           bool `yaml:"api_access_enabled"`
	BulkUploadEnabled          bool `yaml:"bulk_upload_enabled"`
	PrioritySupport            bool `yaml:"priority_support"`
	LeadManagementEnabled      bool `yaml:"lead_management_enabled"`
	ViewingEventsEnabled       bool `yaml:"viewing_events_enabled"`
	OpenHouseEventsEnabled     bool `yaml:"open_house_events_enabled"`
	PrivateShowingsEnabled     bool `yaml:"private_showings_enabled"`
}

// ListingPromotionConfig defines a promotion type configuration
type ListingPromotionConfig struct {
	Name        string                      `yaml:"name"`
	Description string                      `yaml:"description"`
	Enabled     bool                        `yaml:"enabled"`
	Pricing     map[string]int64            `yaml:"pricing"` // duration -> price in minor units
	Benefits    []string                    `yaml:"benefits"`
	Placement   PromotionPlacementConfig    `yaml:"placement"`
}

// PromotionPlacementConfig defines placement rules for promotions
type PromotionPlacementConfig struct {
	MaxConcurrentPerCategory int     `yaml:"max_concurrent_per_category"`
	RotationEnabled          bool    `yaml:"rotation_enabled"`
	SearchBoostMultiplier    float64 `yaml:"search_boost_multiplier"`
	MaxPhotosAllowed         int     `yaml:"max_photos_allowed,omitempty"`
	VirtualTourEnabled       bool    `yaml:"virtual_tour_enabled,omitempty"`
	AnalyticsEnabled         bool    `yaml:"analytics_enabled,omitempty"`
}

// SubscriptionPlanConfig defines a subscription plan
type SubscriptionPlanConfig struct {
	Name           string                       `yaml:"name"`
	Description    string                       `yaml:"description"`
	Enabled        bool                         `yaml:"enabled"`
	TargetAudience string                       `yaml:"target_audience"`
	Pricing        SubscriptionPricingConfig    `yaml:"pricing"`
	Limits         SubscriptionLimitsConfig     `yaml:"limits"`
	Features       map[string]bool              `yaml:"features"`
}

// SubscriptionPricingConfig defines pricing for a subscription plan
type SubscriptionPricingConfig struct {
	Monthly int64 `yaml:"monthly"` // in minor units
	Yearly  int64 `yaml:"yearly"`  // in minor units
}

// SubscriptionLimitsConfig defines usage limits for a subscription plan
type SubscriptionLimitsConfig struct {
	MaxListings                     int `yaml:"max_listings"`          // -1 = unlimited
	MaxPhotosPerListing             int `yaml:"max_photos_per_listing"`
	MaxVirtualTours                 int `yaml:"max_virtual_tours"`     // -1 = unlimited
	IncludedFeaturedPerMonth        int `yaml:"included_featured_per_month"`
	IncludedPremiumPerMonth         int `yaml:"included_premium_per_month"` // -1 = unlimited
	IncludedOpenHousesPerMonth      int `yaml:"included_open_houses_per_month"` // -1 = unlimited
	IncludedPrivateShowingsPerMonth int `yaml:"included_private_showings_per_month"` // -1 = unlimited
}

// AddonConfig defines an add-on product
type AddonConfig struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Pricing     map[string]int64       `yaml:"pricing"` // e.g., "one_time" -> price
	Includes    map[string]int         `yaml:"includes,omitempty"` // e.g., "open_houses" -> 5
}

// BillingConfig defines billing and usage settings
type BillingConfig struct {
	TrialPeriodDays int    `yaml:"trial_period_days"`
	GracePeriodDays int    `yaml:"grace_period_days"`
	AutoRenewDefault bool  `yaml:"auto_renew_default"`
	ProrationEnabled bool  `yaml:"proration_enabled"`
	UsageResetDay    int   `yaml:"usage_reset_day"`
	UsageResetTime   string `yaml:"usage_reset_time"`
}

// AnalyticsConfig defines analytics tracking settings
type AnalyticsConfig struct {
	TrackImpressions bool     `yaml:"track_impressions"`
	TrackClicks      bool     `yaml:"track_clicks"`
	TrackLeads       bool     `yaml:"track_leads"`
	TrackMessages    bool     `yaml:"track_messages"`
	RetentionDays    int      `yaml:"retention_days"`
	FunnelEvents     []string `yaml:"funnel_events"`
}

// RateLimitsConfig defines rate limiting settings
type RateLimitsConfig struct {
	FreeTier                          FreeTierRateLimitConfig `yaml:"free_tier"`
	PaidTier                          PaidTierRateLimitConfig `yaml:"paid_tier"`
	MaxActivePromotionsPerListing     int                     `yaml:"max_active_promotions_per_listing"`
	MinPromotionDurationHours         int                     `yaml:"min_promotion_duration_hours"`
	CooldownBetweenPromotionsHours    int                     `yaml:"cooldown_between_promotions_hours"`
}

// FreeTierRateLimitConfig defines rate limits for free tier
type FreeTierRateLimitConfig struct {
	LeadsPerDay        int `yaml:"leads_per_day"`
	PromotionsPerMonth int `yaml:"promotions_per_month"`
}

// PaidTierRateLimitConfig defines rate limits for paid tiers
type PaidTierRateLimitConfig struct {
	LeadsPerDay        int `yaml:"leads_per_day"`        // -1 = unlimited
	PromotionsPerMonth int `yaml:"promotions_per_month"` // -1 = unlimited
}

// PromotionNotificationsConfig defines notification settings
type PromotionNotificationsConfig struct {
	SendPromotionStarted          bool `yaml:"send_promotion_started"`
	SendPromotionExpiring         bool `yaml:"send_promotion_expiring"`
	SendPromotionExpired          bool `yaml:"send_promotion_expired"`
	SendSubscriptionRenewal       bool `yaml:"send_subscription_renewal"`
	SendUsageWarnings             bool `yaml:"send_usage_warnings"`
	SendAnalyticsReports          bool `yaml:"send_analytics_reports"`
	UsageWarningThresholdPercent  int  `yaml:"usage_warning_threshold_percent"`
}

// GetPromotionPrice returns the price for a promotion type and duration
func (p *PromotionYAMLConfig) GetPromotionPrice(promoType string, durationDays int) (int64, bool) {
	promo, exists := p.ListingPromotions[promoType]
	if !exists || !promo.Enabled {
		return 0, false
	}

	// Try exact duration match
	durationKey := formatDurationKey(durationDays)
	if price, ok := promo.Pricing[durationKey]; ok {
		return price, true
	}

	return 0, false
}

// GetSubscriptionPrice returns the price for a subscription plan and billing cycle
func (p *PromotionYAMLConfig) GetSubscriptionPrice(planType, billingCycle string) (int64, bool) {
	plan, exists := p.SubscriptionPlans[planType]
	if !exists || !plan.Enabled {
		return 0, false
	}

	switch billingCycle {
	case "monthly":
		return plan.Pricing.Monthly, true
	case "yearly":
		return plan.Pricing.Yearly, true
	default:
		return 0, false
	}
}

// GetPlanLimits returns the limits for a subscription plan
func (p *PromotionYAMLConfig) GetPlanLimits(planType string) (*SubscriptionLimitsConfig, bool) {
	plan, exists := p.SubscriptionPlans[planType]
	if !exists || !plan.Enabled {
		return nil, false
	}
	return &plan.Limits, true
}

// GetPlanFeatures returns the features map for a subscription plan
func (p *PromotionYAMLConfig) GetPlanFeatures(planType string) (map[string]bool, bool) {
	plan, exists := p.SubscriptionPlans[planType]
	if !exists || !plan.Enabled {
		return nil, false
	}
	return plan.Features, true
}

// formatDurationKey formats duration in days to a YAML key (e.g., 7 -> "7_days")
func formatDurationKey(days int) string {
	return fmt.Sprintf("%d_days", days)
}
