package breaker

import (
	"context"
	"fmt"
	"hauslet/internal/platform/redis"
	"log/slog"
	"time"
)

// RedisCircuitBreaker implements CircuitBreaker using Redis for state storage
type RedisCircuitBreaker struct {
	redis  redis.RedisClient
	config ProviderConfig
	logger *slog.Logger
}

// NewRedisCircuitBreaker creates a new Redis-backed circuit breaker
func NewRedisCircuitBreaker(redisClient redis.RedisClient, config ProviderConfig, logger *slog.Logger) *RedisCircuitBreaker {
	return &RedisCircuitBreaker{
		redis:  redisClient,
		config: config,
		logger: logger,
	}
}

// AllowRequest checks if a request is allowed for the provider
func (cb *RedisCircuitBreaker) AllowRequest(ctx context.Context, providerName string) (bool, error) {
	cfg := cb.getConfig(providerName)
	keys := []string{
		cb.getKey(providerName, "state"),
		cb.getKey(providerName, "opened_at"),
		cb.getKey(providerName, "probe_count"),
	}
	args := []any{
		time.Now().Unix(),
		cfg.Timeout.Seconds(),
		cfg.HalfOpenRequests,
		10, // probe key TTL (seconds)
	}

	allowed, err := allowRequestScript.Run(ctx, cb.redis, keys, args...).Bool()
	if err != nil {
		// Default to allowing on error to avoid blocking traffic if Redis is down
		cb.logger.Error("failed to run allow_request script, allowing request",
			"provider", providerName,
			"error", err,
		)
		return true, nil
	}

	return allowed, nil
}

// RecordSuccess records a successful request
func (cb *RedisCircuitBreaker) RecordSuccess(ctx context.Context, providerName string) error {
	cfg := cb.getConfig(providerName)
	keys := []string{
		cb.getKey(providerName, "state"),
		cb.getKey(providerName, "success_count"),
		cb.getKey(providerName, "last_success"),
		cb.getKey(providerName, "failure_count"),
		cb.getKey(providerName, "opened_at"),
		cb.getKey(providerName, "probe_count"),
	}
	args := []any{
		cfg.SuccessThreshold,
		time.Now().Unix(),
	}

	closed, err := recordSuccessScript.Run(ctx, cb.redis, keys, args...).Bool()
	if err != nil {
		return fmt.Errorf("failed to record success: %w", err)
	}

	if closed {
		cb.logger.Info("circuit breaker closed after successful recovery",
			"provider", providerName,
		)
	}

	return nil
}

// RecordFailure records a failed request
func (cb *RedisCircuitBreaker) RecordFailure(ctx context.Context, providerName string) error {
	cfg := cb.getConfig(providerName)
	keys := []string{
		cb.getKey(providerName, "state"),
		cb.getKey(providerName, "failure_count"),
		cb.getKey(providerName, "last_failure"),
		cb.getKey(providerName, "opened_at"),
		cb.getKey(providerName, "success_count"),
		cb.getKey(providerName, "probe_count"),
	}
	args := []any{
		cfg.FailureThreshold,
		time.Now().Unix(),
	}

	opened, err := recordFailureScript.Run(ctx, cb.redis, keys, args...).Bool()
	if err != nil {
		return fmt.Errorf("failed to record failure: %w", err)
	}

	if opened {
		cb.logger.Warn("circuit breaker opened due to failures",
			"provider", providerName,
		)
	}

	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *RedisCircuitBreaker) GetState(ctx context.Context, providerName string) (State, error) {
	stateKey := cb.getKey(providerName, "state")

	val, err := cb.redis.Get(ctx, stateKey).Result()
	if err != nil {
		// Default to closed if key doesn't exist
		if err.Error() == "redis: nil" {
			return StateClosed, nil
		}
		return StateClosed, fmt.Errorf("failed to get state: %w", err)
	}

	return State(val), nil
}

// GetStats returns statistics for the circuit breaker
func (cb *RedisCircuitBreaker) GetStats(ctx context.Context, providerName string) (*Stats, error) {
	state, err := cb.GetState(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	stats := &Stats{
		ProviderName:  providerName,
		State:         state,
		LastCheckedAt: time.Now(),
	}

	// Get counters
	stats.FailureCount, _ = cb.getCounter(ctx, cb.getKey(providerName, "failure_count"))
	stats.SuccessCount, _ = cb.getCounter(ctx, cb.getKey(providerName, "success_count"))

	// Get timestamps
	stats.LastFailureAt = cb.getTimestamp(ctx, cb.getKey(providerName, "last_failure"))
	stats.LastSuccessAt = cb.getTimestamp(ctx, cb.getKey(providerName, "last_success"))
	stats.OpenedAt = cb.getTimestamp(ctx, cb.getKey(providerName, "opened_at"))

	return stats, nil
}

// Reset manually resets the circuit breaker to closed state
func (cb *RedisCircuitBreaker) Reset(ctx context.Context, providerName string) error {
	if err := cb.setState(ctx, providerName, StateClosed); err != nil {
		return fmt.Errorf("failed to set state: %w", err)
	}

	if err := cb.resetCounters(ctx, providerName); err != nil {
		return fmt.Errorf("failed to reset counters: %w", err)
	}

	cb.logger.Info("circuit breaker manually reset",
		"provider", providerName,
	)

	return nil
}

// Helper: setState sets the circuit breaker state
func (cb *RedisCircuitBreaker) setState(ctx context.Context, providerName string, state State) error {
	stateKey := cb.getKey(providerName, "state")
	return cb.redis.Set(ctx, stateKey, string(state), 0).Err()
}

// Helper: resetCounters resets all counters for a provider
func (cb *RedisCircuitBreaker) resetCounters(ctx context.Context, providerName string) error {
	keys := []string{
		cb.getKey(providerName, "failure_count"),
		cb.getKey(providerName, "success_count"),
		cb.getKey(providerName, "opened_at"),
		cb.getKey(providerName, "probe_count"),
	}

	for _, key := range keys {
		if err := cb.redis.Del(ctx, key).Err(); err != nil {
			return err
		}
	}

	return nil
}

// Helper: getKey generates a Redis key
func (cb *RedisCircuitBreaker) getKey(providerName, suffix string) string {
	return fmt.Sprintf("breaker:%s:%s", providerName, suffix)
}

// Helper: getCounter gets a counter value
func (cb *RedisCircuitBreaker) getCounter(ctx context.Context, key string) (int64, error) {
	val, err := cb.redis.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}

	var count int64
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil {
		return 0, err
	}

	return count, nil
}

// Helper: getTimestamp gets a timestamp value
func (cb *RedisCircuitBreaker) getTimestamp(ctx context.Context, key string) *time.Time {
	val, err := cb.redis.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	var unix int64
	if _, err := fmt.Sscanf(val, "%d", &unix); err != nil {
		return nil
	}

	t := time.Unix(unix, 0)
	return &t
}

// Helper: getConfig gets configuration for a provider
func (cb *RedisCircuitBreaker) getConfig(providerName string) Config {
	if cfg, ok := cb.config[providerName]; ok {
		return cfg
	}
	return DefaultConfig()
}
