package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ServiceConfig holds all YAML-based service configurations
type ServiceConfig struct {
	Calendar  CalendarYAMLConfig  `yaml:"calendar"`
	Queue     QueueYAMLConfig     `yaml:"queue"`
	Features  FeatureYAMLConfig   `yaml:"features"`
	Platform  PlatformYAMLConfig  `yaml:"platform"`
	Promotion PromotionYAMLConfig `yaml:"promotion"`
	RateLimit RateLimitYAMLConfig `yaml:"ratelimit"`
}

// CalendarYAMLConfig defines calendar service settings
type CalendarYAMLConfig struct {
	MaintenanceHour     int  `yaml:"maintenance_hour"`
	WeeklyCheckDay      int  `yaml:"weekly_check_day"`
	CleanupOldDays      int  `yaml:"cleanup_old_days"`
	SchedulerEnabled    bool `yaml:"scheduler_enabled"`
	ImmediateGeneration bool `yaml:"immediate_generation"`
	RetryDelayMinutes   int  `yaml:"retry_delay_minutes"`
}

// QueueYAMLConfig defines queue/Cloud Tasks settings
type QueueYAMLConfig struct {
	Subjects map[string]string `yaml:"subjects"`
}

// FeatureYAMLConfig defines feature flags
type FeatureYAMLConfig struct {
	EmailQueueEnabled   bool `yaml:"email_queue_enabled"`
	OAuthEnabled        bool `yaml:"oauth_enabled"`
	RateLimitingEnabled bool `yaml:"rate_limiting_enabled"`
}

// PlatformYAMLConfig defines platform-wide knobs (fees, payouts, etc.).
type PlatformYAMLConfig struct {
	Currency       PlatformCurrencyConfig       `yaml:"currency"`
	Fees           PlatformFeesConfig           `yaml:"fees"`
	Taxes          PlatformTaxConfig            `yaml:"taxes"`
	Payouts        PlatformPayoutConfig         `yaml:"payouts"`
	Refunds        PlatformRefundConfig         `yaml:"refunds"`
	AutoAccept     PlatformAutoAcceptConfig     `yaml:"auto_accept"`
	Reviews        PlatformReviewConfig         `yaml:"reviews"`
	Notifications  PlatformNotificationConfig   `yaml:"notifications"`
	Reconciliation PlatformReconciliationConfig `yaml:"reconciliation"`
	Messaging      PlatformMessagingConfig      `yaml:"messaging"`
}

type PlatformCurrencyConfig struct {
	Code      string `yaml:"code"`
	MinorUnit int64  `yaml:"minor_unit"`
}

type PlatformFeesConfig struct {
	HostCommissionPercent    float64 `yaml:"host_commission_percent"`
	GuestServicePercent      float64 `yaml:"guest_service_percent"`
	PayoutProcessingPercent  float64 `yaml:"payout_processing_percent"`
	MinimumServiceFeeMinor   int64   `yaml:"minimum_service_fee_minor"`
	MaximumServiceFeePercent float64 `yaml:"maximum_service_fee_percent"`
}

type PlatformTaxConfig struct {
	VATPercent float64 `yaml:"vat_percent"`
}

// EscrowReleaseEvent defines when escrow funds become available for payout
type EscrowReleaseEvent string

const (
	// EscrowReleaseCheckinConfirmed releases funds N hours after check-in
	EscrowReleaseCheckinConfirmed EscrowReleaseEvent = "checkin_confirmed"
	// EscrowReleaseCheckoutConfirmed releases funds N hours after check-out
	EscrowReleaseCheckoutConfirmed EscrowReleaseEvent = "checkout_confirmed"
)

// String returns the string representation of EscrowReleaseEvent
func (e EscrowReleaseEvent) String() string {
	return string(e)
}

// IsValid checks if the EscrowReleaseEvent is a valid value
func (e EscrowReleaseEvent) IsValid() bool {
	switch e {
	case EscrowReleaseCheckinConfirmed, EscrowReleaseCheckoutConfirmed:
		return true
	default:
		return false
	}
}

type PlatformPayoutConfig struct {
	EscrowReleaseHours   int                `yaml:"escrow_release_hours"`
	EscrowReleaseEvent   EscrowReleaseEvent `yaml:"escrow_release_event"`
	DisbursementProvider string             `yaml:"disbursement_provider"`
	BatchIntervalMinutes int                `yaml:"batch_interval_minutes"`
	MaxRetryAttempts     int                `yaml:"max_retry_attempts"`
	RetryBackoffMinutes  int                `yaml:"retry_backoff_minutes"`
}

type PlatformRefundConfig struct {
	GracePeriodHours     int                         `yaml:"grace_period_hours"`
	ProcessingFeePercent float64                     `yaml:"processing_fee_percent"`
	ProcessingFeePayer   string                      `yaml:"processing_fee_payer"` // guest | host | shared
	DisputeFreezeHours   int                         `yaml:"dispute_freeze_hours"`
	PolicyTiers          map[string]RefundPolicyTier `yaml:"policy_tiers"`
}

// RefundPolicyTier defines a cancellation/refund policy tier
type RefundPolicyTier struct {
	Name                      string  `yaml:"name"`
	Description               string  `yaml:"description"`
	CutoffHoursBeforeCheckIn  int     `yaml:"cutoff_hours_before_checkin"`
	RefundPercentBeforeCutoff float64 `yaml:"refund_percent_before_cutoff"`
	RefundPercentAfterCutoff  float64 `yaml:"refund_percent_after_cutoff"`
	MinStayDays               int     `yaml:"min_stay_days,omitempty"`            // For long_term
	NoticePeriodDays          int     `yaml:"notice_period_days,omitempty"`       // For long_term
	CancellationChargeDays    int     `yaml:"cancellation_charge_days,omitempty"` // For long_term
}

type PlatformAutoAcceptConfig struct {
	DefaultEnabled bool `yaml:"default_enabled"`
	MinNoticeHours int  `yaml:"min_notice_hours"`
}

type PlatformNotificationConfig struct {
	SendPaymentReceipts bool `yaml:"send_payment_receipts"`
	SendPayoutUpdates   bool `yaml:"send_payout_updates"`
	SendDisputeAlerts   bool `yaml:"send_dispute_alerts"`
}

type PlatformReviewConfig struct {
	ReviewWindowDays int `yaml:"review_window_days"`
}

type PlatformReconciliationConfig struct {
	SendAlerts  bool     `yaml:"send_alerts"`
	AdminRoles  []string `yaml:"admin_roles"`
	MinSeverity string   `yaml:"min_severity"` // critical | high | medium | low
}

type PlatformMessagingConfig struct {
	ArchiveAfterDays int `yaml:"archive_after_days"`
	DeleteAfterDays  int `yaml:"delete_after_days"`
}

// RateLimitYAMLConfig defines rate limiting rules
type RateLimitYAMLConfig struct {
	Verification VerificationRateLimits `yaml:"verification"`
	Leads        LeadsRateLimits        `yaml:"leads"`
}

// VerificationRateLimits defines rate limits for verification operations
type VerificationRateLimits struct {
	User    RateLimitRule `yaml:"user"`    // Per user limits
	IP      RateLimitRule `yaml:"ip"`      // Per IP address limits
	Phone   RateLimitRule `yaml:"phone"`   // Per phone number limits
	Country RateLimitRule `yaml:"country"` // Per country limits
}

// RateLimitRule defines a single rate limit rule
type RateLimitRule struct {
	Limit  int64  `yaml:"limit"`  // Maximum allowed requests
	Window string `yaml:"window"` // Time window (e.g., "24h", "1h")
}

// LeadsRateLimits defines rate limits for lead submissions.
type LeadsRateLimits struct {
	Window        string             `yaml:"window"` // Time window (e.g., "24h")
	Anonymous     LeadsRateLimitTier `yaml:"anonymous"`
	Authenticated LeadsRateLimitTier `yaml:"authenticated"`
}

// LeadsRateLimitTier defines rate limits for a requester tier.
type LeadsRateLimitTier struct {
	PerEmail        int64 `yaml:"per_email"`
	PerIP           int64 `yaml:"per_ip"`
	PerListingEmail int64 `yaml:"per_listing_email"`
	PerListingIP    int64 `yaml:"per_listing_ip"`
	PerUser         int64 `yaml:"per_user"`
}

// LoadYAMLConfig loads service configuration from YAML files
// It loads from defaults/ and optionally overrides/ directories
func LoadYAMLConfig() (*ServiceConfig, error) {
	// Get the config directory (where this file is located)
	configDir := filepath.Join("config")

	defaultsPath := filepath.Join(configDir, "defaults")
	overridesPath := filepath.Join(configDir, "overrides")

	return loadYAMLConfigFromPaths(defaultsPath, overridesPath)
}

// loadYAMLConfigFromPaths loads and merges YAML configs from specified paths
func loadYAMLConfigFromPaths(defaultsPath, overridesPath string) (*ServiceConfig, error) {
	cfg := &ServiceConfig{}

	// Load defaults first
	if err := loadYAMLFiles(cfg, defaultsPath); err != nil {
		return nil, fmt.Errorf("failed to load default configs: %w", err)
	}

	// Load overrides (if they exist) - these will merge/replace defaults
	if _, err := os.Stat(overridesPath); err == nil {
		if err := loadYAMLFiles(cfg, overridesPath); err != nil {
			// Overrides are optional, log but don't fail
			fmt.Printf("WARN: Failed to load override configs: %v\n", err)
		}
	}

	return cfg, nil
}

// loadYAMLFiles loads all YAML files from a directory into the config
func loadYAMLFiles(cfg *ServiceConfig, dirPath string) error {
	files := []string{"calendar.yaml", "queue.yaml", "features.yaml", "platform.yaml", "promotion.yaml", "ratelimit.yaml"}

	for _, filename := range files {
		filePath := filepath.Join(dirPath, filename)

		// Skip if file doesn't exist
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filename, err)
		}

		// Unmarshal into a temporary struct to allow partial loading
		tempCfg := &ServiceConfig{}
		if err := yaml.Unmarshal(data, tempCfg); err != nil {
			return fmt.Errorf("failed to parse %s: %w", filename, err)
		}

		// Merge into main config
		mergeServiceConfig(cfg, tempCfg)
	}

	return nil
}

// mergeServiceConfig merges source config into destination
// Non-zero values from source override destination
func mergeServiceConfig(dst, src *ServiceConfig) {
	// Merge Calendar config
	if src.Calendar.MaintenanceHour != 0 {
		dst.Calendar.MaintenanceHour = src.Calendar.MaintenanceHour
	}
	if src.Calendar.WeeklyCheckDay != 0 {
		dst.Calendar.WeeklyCheckDay = src.Calendar.WeeklyCheckDay
	}
	if src.Calendar.CleanupOldDays != 0 {
		dst.Calendar.CleanupOldDays = src.Calendar.CleanupOldDays
	}
	if src.Calendar.SchedulerEnabled {
		dst.Calendar.SchedulerEnabled = src.Calendar.SchedulerEnabled
	}
	if src.Calendar.ImmediateGeneration {
		dst.Calendar.ImmediateGeneration = src.Calendar.ImmediateGeneration
	}
	if src.Calendar.RetryDelayMinutes != 0 {
		dst.Calendar.RetryDelayMinutes = src.Calendar.RetryDelayMinutes
	}

	// Merge Queue config
	if src.Queue.Subjects != nil {
		if dst.Queue.Subjects == nil {
			dst.Queue.Subjects = make(map[string]string)
		}
		for k, v := range src.Queue.Subjects {
			dst.Queue.Subjects[k] = v
		}
	}

	// Merge Features config
	if src.Features.EmailQueueEnabled {
		dst.Features.EmailQueueEnabled = src.Features.EmailQueueEnabled
	}
	if src.Features.OAuthEnabled {
		dst.Features.OAuthEnabled = src.Features.OAuthEnabled
	}
	if src.Features.RateLimitingEnabled {
		dst.Features.RateLimitingEnabled = src.Features.RateLimitingEnabled
	}

	// Merge Platform config
	mergePlatformConfig(&dst.Platform, &src.Platform)

	// Merge Promotion config
	mergePromotionConfig(&dst.Promotion, &src.Promotion)
}

func mergePlatformConfig(dst, src *PlatformYAMLConfig) {
	// Currency
	if src.Currency.Code != "" {
		dst.Currency.Code = src.Currency.Code
	}
	if src.Currency.MinorUnit != 0 {
		dst.Currency.MinorUnit = src.Currency.MinorUnit
	}

	// Fees
	if src.Fees.HostCommissionPercent != 0 {
		dst.Fees.HostCommissionPercent = src.Fees.HostCommissionPercent
	}
	if src.Fees.GuestServicePercent != 0 {
		dst.Fees.GuestServicePercent = src.Fees.GuestServicePercent
	}
	if src.Fees.PayoutProcessingPercent != 0 {
		dst.Fees.PayoutProcessingPercent = src.Fees.PayoutProcessingPercent
	}
	if src.Fees.MinimumServiceFeeMinor != 0 {
		dst.Fees.MinimumServiceFeeMinor = src.Fees.MinimumServiceFeeMinor
	}
	if src.Fees.MaximumServiceFeePercent != 0 {
		dst.Fees.MaximumServiceFeePercent = src.Fees.MaximumServiceFeePercent
	}

	// Taxes
	if src.Taxes.VATPercent != 0 {
		dst.Taxes.VATPercent = src.Taxes.VATPercent
	}

	// Payouts
	if src.Payouts.EscrowReleaseHours != 0 {
		dst.Payouts.EscrowReleaseHours = src.Payouts.EscrowReleaseHours
	}
	if src.Payouts.EscrowReleaseEvent.IsValid() {
		dst.Payouts.EscrowReleaseEvent = src.Payouts.EscrowReleaseEvent
	}
	if src.Payouts.DisbursementProvider != "" {
		dst.Payouts.DisbursementProvider = src.Payouts.DisbursementProvider
	}
	if src.Payouts.BatchIntervalMinutes != 0 {
		dst.Payouts.BatchIntervalMinutes = src.Payouts.BatchIntervalMinutes
	}
	if src.Payouts.MaxRetryAttempts != 0 {
		dst.Payouts.MaxRetryAttempts = src.Payouts.MaxRetryAttempts
	}
	if src.Payouts.RetryBackoffMinutes != 0 {
		dst.Payouts.RetryBackoffMinutes = src.Payouts.RetryBackoffMinutes
	}

	// Refunds
	if src.Refunds.GracePeriodHours != 0 {
		dst.Refunds.GracePeriodHours = src.Refunds.GracePeriodHours
	}
	if src.Refunds.ProcessingFeePercent != 0 {
		dst.Refunds.ProcessingFeePercent = src.Refunds.ProcessingFeePercent
	}
	if src.Refunds.DisputeFreezeHours != 0 {
		dst.Refunds.DisputeFreezeHours = src.Refunds.DisputeFreezeHours
	}

	// Auto accept
	if src.AutoAccept.DefaultEnabled {
		dst.AutoAccept.DefaultEnabled = src.AutoAccept.DefaultEnabled
	}
	if src.AutoAccept.MinNoticeHours != 0 {
		dst.AutoAccept.MinNoticeHours = src.AutoAccept.MinNoticeHours
	}

	// Reviews
	if src.Reviews.ReviewWindowDays != 0 {
		dst.Reviews.ReviewWindowDays = src.Reviews.ReviewWindowDays
	}

	// Notifications
	if src.Notifications.SendPaymentReceipts {
		dst.Notifications.SendPaymentReceipts = src.Notifications.SendPaymentReceipts
	}
	if src.Notifications.SendPayoutUpdates {
		dst.Notifications.SendPayoutUpdates = src.Notifications.SendPayoutUpdates
	}
	if src.Notifications.SendDisputeAlerts {
		dst.Notifications.SendDisputeAlerts = src.Notifications.SendDisputeAlerts
	}

	// Reconciliation
	if src.Reconciliation.SendAlerts {
		dst.Reconciliation.SendAlerts = src.Reconciliation.SendAlerts
	}
	if len(src.Reconciliation.AdminRoles) > 0 {
		dst.Reconciliation.AdminRoles = src.Reconciliation.AdminRoles
	}
	if src.Reconciliation.MinSeverity != "" {
		dst.Reconciliation.MinSeverity = src.Reconciliation.MinSeverity
	}

	// Messaging
	if src.Messaging.ArchiveAfterDays != 0 {
		dst.Messaging.ArchiveAfterDays = src.Messaging.ArchiveAfterDays
	}
	if src.Messaging.DeleteAfterDays != 0 {
		dst.Messaging.DeleteAfterDays = src.Messaging.DeleteAfterDays
	}
}

func mergePromotionConfig(dst, src *PromotionYAMLConfig) {
	// Currency
	if src.Currency.Code != "" {
		dst.Currency.Code = src.Currency.Code
	}
	if src.Currency.MinorUnit != 0 {
		dst.Currency.MinorUnit = src.Currency.MinorUnit
	}

	// Free Tier
	if src.FreeTier.MaxListings != 0 {
		dst.FreeTier.MaxListings = src.FreeTier.MaxListings
	}
	if src.FreeTier.MaxPhotosPerListing != 0 {
		dst.FreeTier.MaxPhotosPerListing = src.FreeTier.MaxPhotosPerListing
	}
	if src.FreeTier.MaxVirtualTours != 0 {
		dst.FreeTier.MaxVirtualTours = src.FreeTier.MaxVirtualTours
	}

	// Listing Promotions
	if src.ListingPromotions != nil {
		if dst.ListingPromotions == nil {
			dst.ListingPromotions = make(map[string]ListingPromotionConfig)
		}
		for k, v := range src.ListingPromotions {
			dst.ListingPromotions[k] = v
		}
	}

	// Subscription Plans
	if src.SubscriptionPlans != nil {
		if dst.SubscriptionPlans == nil {
			dst.SubscriptionPlans = make(map[string]SubscriptionPlanConfig)
		}
		for k, v := range src.SubscriptionPlans {
			dst.SubscriptionPlans[k] = v
		}
	}

	// Addons
	if src.Addons != nil {
		if dst.Addons == nil {
			dst.Addons = make(map[string]AddonConfig)
		}
		for k, v := range src.Addons {
			dst.Addons[k] = v
		}
	}

	// Billing
	if src.Billing.TrialPeriodDays != 0 {
		dst.Billing.TrialPeriodDays = src.Billing.TrialPeriodDays
	}
	if src.Billing.GracePeriodDays != 0 {
		dst.Billing.GracePeriodDays = src.Billing.GracePeriodDays
	}
	if src.Billing.UsageResetDay != 0 {
		dst.Billing.UsageResetDay = src.Billing.UsageResetDay
	}

	// Analytics
	if src.Analytics.RetentionDays != 0 {
		dst.Analytics.RetentionDays = src.Analytics.RetentionDays
	}
	if src.Analytics.FunnelEvents != nil {
		dst.Analytics.FunnelEvents = src.Analytics.FunnelEvents
	}

	// Rate Limits
	if src.RateLimits.MaxActivePromotionsPerListing != 0 {
		dst.RateLimits.MaxActivePromotionsPerListing = src.RateLimits.MaxActivePromotionsPerListing
	}
	if src.RateLimits.MinPromotionDurationHours != 0 {
		dst.RateLimits.MinPromotionDurationHours = src.RateLimits.MinPromotionDurationHours
	}
}
