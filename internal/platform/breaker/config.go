package breaker

import "time"

// Config defines circuit breaker configuration
type Config struct {
	FailureThreshold int           // Number of failures before opening circuit
	SuccessThreshold int           // Number of successes to close circuit (from half-open)
	Timeout          time.Duration // How long circuit stays open before half-open
	HalfOpenRequests int           // Max concurrent requests in half-open state
}

// DefaultConfig returns default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		HalfOpenRequests: 1,
	}
}

// ProviderConfig maps provider names to their circuit breaker configs
type ProviderConfig map[string]Config

// DefaultProviderConfig returns default configuration for KYC providers
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		"dojah": {
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			HalfOpenRequests: 1,
		},
		"veriff": {
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			HalfOpenRequests: 1,
		},
	}
}
