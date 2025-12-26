package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ServiceConfig holds all YAML-based service configurations
type ServiceConfig struct {
	Calendar CalendarYAMLConfig `yaml:"calendar"`
	Queue    QueueYAMLConfig    `yaml:"queue"`
	Features FeatureYAMLConfig  `yaml:"features"`
	Platform PlatformYAMLConfig `yaml:"platform"`
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

// QueueYAMLConfig defines queue/NATS service settings
type QueueYAMLConfig struct {
	StreamName string            `yaml:"stream_name"`
	Subjects   map[string]string `yaml:"subjects"`
	Consumers  map[string]string `yaml:"consumers"`
}

// FeatureYAMLConfig defines feature flags
type FeatureYAMLConfig struct {
	EmailQueueEnabled   bool `yaml:"email_queue_enabled"`
	OAuthEnabled        bool `yaml:"oauth_enabled"`
	RateLimitingEnabled bool `yaml:"rate_limiting_enabled"`
}

// PlatformYAMLConfig defines platform-wide knobs (fees, payouts, etc.).
type PlatformYAMLConfig struct {
	Currency      PlatformCurrencyConfig     `yaml:"currency"`
	Fees          PlatformFeesConfig         `yaml:"fees"`
	Payouts       PlatformPayoutConfig       `yaml:"payouts"`
	Refunds       PlatformRefundConfig       `yaml:"refunds"`
	AutoAccept    PlatformAutoAcceptConfig   `yaml:"auto_accept"`
	Notifications PlatformNotificationConfig `yaml:"notifications"`
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

type PlatformPayoutConfig struct {
	EscrowReleaseHours   int `yaml:"escrow_release_hours"`
	BatchIntervalMinutes int `yaml:"batch_interval_minutes"`
	MaxRetryAttempts     int `yaml:"max_retry_attempts"`
	RetryBackoffMinutes  int `yaml:"retry_backoff_minutes"`
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
	Name                     string  `yaml:"name"`
	Description              string  `yaml:"description"`
	CutoffHoursBeforeCheckIn int     `yaml:"cutoff_hours_before_checkin"`
	RefundPercentBeforeCutoff float64 `yaml:"refund_percent_before_cutoff"`
	RefundPercentAfterCutoff  float64 `yaml:"refund_percent_after_cutoff"`
	MinStayDays              int     `yaml:"min_stay_days,omitempty"`              // For long_term
	NoticePeriodDays         int     `yaml:"notice_period_days,omitempty"`         // For long_term
	CancellationChargeDays   int     `yaml:"cancellation_charge_days,omitempty"`   // For long_term
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
	files := []string{"calendar.yaml", "queue.yaml", "features.yaml", "platform.yaml"}

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
	if src.Queue.StreamName != "" {
		dst.Queue.StreamName = src.Queue.StreamName
	}
	if src.Queue.Subjects != nil {
		if dst.Queue.Subjects == nil {
			dst.Queue.Subjects = make(map[string]string)
		}
		for k, v := range src.Queue.Subjects {
			dst.Queue.Subjects[k] = v
		}
	}
	if src.Queue.Consumers != nil {
		if dst.Queue.Consumers == nil {
			dst.Queue.Consumers = make(map[string]string)
		}
		for k, v := range src.Queue.Consumers {
			dst.Queue.Consumers[k] = v
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

	// Payouts
	if src.Payouts.EscrowReleaseHours != 0 {
		dst.Payouts.EscrowReleaseHours = src.Payouts.EscrowReleaseHours
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
}
