package setup

import (
	"context"
	"log/slog"
	"time"

	"hauslet/config"
	"hauslet/internal/platform/breaker"
	"hauslet/internal/platform/evidence"
	"hauslet/internal/platform/kyc"
	"hauslet/internal/platform/ratelimit"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/sms"
	"hauslet/internal/platform/storage"
)

// InitKYCClient initializes the KYC client with Dojah and Veriff adapters.
func InitKYCClient(cfg *config.GlobalConfig, log *slog.Logger) *kyc.Client {
	// Create provider factory with KYC config (handles adapter initialization internally)
	factory := kyc.NewProviderFactory(cfg.Services.KYC)

	// Create KYC client
	client := kyc.New(factory)

	return client
}

// InitSMSClient initializes the SMS client with Termii (primary) and Twilio (fallback).
func InitSMSClient(cfg *config.GlobalConfig, log *slog.Logger) *sms.Client {
	// Initialize Termii adapter (primary)
	termiiAdapter := sms.NewTermiiAdapter(
		cfg.Services.SMS.TermiiAPIKey,
		cfg.Services.SMS.TermiiSenderID,
	)

	// Initialize Twilio adapter (fallback)
	twilioAdapter := sms.NewTwilioAdapter(
		cfg.Services.SMS.TwilioAccountSID,
		cfg.Services.SMS.TwilioAuthToken,
		cfg.Services.SMS.TwilioFromNumber,
	)

	// Create SMS client with fallback
	client := sms.New(termiiAdapter, twilioAdapter, log)

	return client
}

// InitEvidenceStore initializes the evidence storage with R2 backend.
func InitEvidenceStore(r2Storage *storage.R2Storage) evidence.Store {
	store := evidence.NewR2Store(r2Storage)
	return store
}

// InitRateLimiter initializes the Redis-backed rate limiter with YAML configuration.
func InitRateLimiter(redisClient redis.RedisClient, cfg *config.GlobalConfig) ratelimit.Limiter {
	limiter := ratelimit.NewRedisLimiter(redisClient)
	return limiter
}

// InitCircuitBreaker initializes the Redis-backed circuit breaker.
func InitCircuitBreaker(ctx context.Context, redisClient redis.RedisClient, log *slog.Logger) (breaker.CircuitBreaker, error) {
	// Circuit breaker configuration for KYC providers
	providerConfig := breaker.ProviderConfig{
		"dojah": {
			FailureThreshold: 5,                // Open circuit after 5 consecutive failures
			SuccessThreshold: 2,                // Close circuit after 2 consecutive successes in half-open state
			Timeout:          30 * time.Second, // Wait 30s before transitioning to half-open
			HalfOpenRequests: 1,                // Allow 1 request in half-open state
		},
		"veriff": {
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
			HalfOpenRequests: 1,
		},
	}

	cb := breaker.NewRedisCircuitBreaker(redisClient, providerConfig, log)

	return cb, nil
}
