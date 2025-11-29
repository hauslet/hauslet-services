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
	EmailQueueEnabled  bool `yaml:"email_queue_enabled"`
	OAuthEnabled       bool `yaml:"oauth_enabled"`
	RateLimitingEnabled bool `yaml:"rate_limiting_enabled"`
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
	files := []string{"calendar.yaml", "queue.yaml", "features.yaml"}

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
}
