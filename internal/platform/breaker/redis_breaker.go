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
	state, err := cb.GetState(ctx, providerName)
	if err != nil {
		// Default to allowing on error
		cb.logger.Error("failed to get circuit breaker state, allowing request",
			"provider", providerName,
			"error", err,
		)
		return true, nil
	}

	switch state {
	case StateClosed:
		return true, nil

	case StateOpen:
		// Check if timeout has expired
		if cb.shouldTransitionToHalfOpen(ctx, providerName) {
			// Transition to half-open
			if err := cb.setState(ctx, providerName, StateHalfOpen); err != nil {
				cb.logger.Error("failed to transition to half-open",
					"provider", providerName,
					"error", err,
				)
				return false, nil
			}
			cb.logger.Info("circuit breaker transitioned to half-open",
				"provider", providerName,
			)
			return true, nil
		}
		return false, nil

	case StateHalfOpen:
		// Allow one test request
		return true, nil

	default:
		return true, nil
	}
}

// RecordSuccess records a successful request
func (cb *RedisCircuitBreaker) RecordSuccess(ctx context.Context, providerName string) error {
	state, err := cb.GetState(ctx, providerName)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Increment success counter
	successKey := cb.getKey(providerName, "success_count")
	if err := cb.redis.Incr(ctx, successKey).Err(); err != nil {
		return fmt.Errorf("failed to increment success count: %w", err)
	}

	// Update last success timestamp
	lastSuccessKey := cb.getKey(providerName, "last_success")
	if err := cb.redis.Set(ctx, lastSuccessKey, time.Now().Unix(), 0).Err(); err != nil {
		return fmt.Errorf("failed to set last success time: %w", err)
	}

	// Handle state transitions based on success
	if state == StateHalfOpen {
		// Check if we have enough successes to close
		successCount, err := cb.getCounter(ctx, successKey)
		if err != nil {
			return fmt.Errorf("failed to get success count: %w", err)
		}

		cfg := cb.getConfig(providerName)
		if successCount >= int64(cfg.SuccessThreshold) {
			// Transition to closed
			if err := cb.setState(ctx, providerName, StateClosed); err != nil {
				return fmt.Errorf("failed to close circuit: %w", err)
			}

			// Reset counters
			if err := cb.resetCounters(ctx, providerName); err != nil {
				return fmt.Errorf("failed to reset counters: %w", err)
			}

			cb.logger.Info("circuit breaker closed after successful recovery",
				"provider", providerName,
				"success_count", successCount,
			)
		}
	}

	return nil
}

// RecordFailure records a failed request
func (cb *RedisCircuitBreaker) RecordFailure(ctx context.Context, providerName string) error {
	state, err := cb.GetState(ctx, providerName)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	// Increment failure counter
	failureKey := cb.getKey(providerName, "failure_count")
	if err := cb.redis.Incr(ctx, failureKey).Err(); err != nil {
		return fmt.Errorf("failed to increment failure count: %w", err)
	}

	// Update last failure timestamp
	lastFailureKey := cb.getKey(providerName, "last_failure")
	if err := cb.redis.Set(ctx, lastFailureKey, time.Now().Unix(), 0).Err(); err != nil {
		return fmt.Errorf("failed to set last failure time: %w", err)
	}

	// Handle state transitions based on failure
	if state == StateClosed {
		// Check if we should open the circuit
		failureCount, err := cb.getCounter(ctx, failureKey)
		if err != nil {
			return fmt.Errorf("failed to get failure count: %w", err)
		}

		cfg := cb.getConfig(providerName)
		if failureCount >= int64(cfg.FailureThreshold) {
			// Transition to open
			if err := cb.setState(ctx, providerName, StateOpen); err != nil {
				return fmt.Errorf("failed to open circuit: %w", err)
			}

			// Set opened timestamp
			openedKey := cb.getKey(providerName, "opened_at")
			if err := cb.redis.Set(ctx, openedKey, time.Now().Unix(), 0).Err(); err != nil {
				return fmt.Errorf("failed to set opened time: %w", err)
			}

			cb.logger.Warn("circuit breaker opened due to failures",
				"provider", providerName,
				"failure_count", failureCount,
			)
		}
	} else if state == StateHalfOpen {
		// One failure in half-open means back to open
		if err := cb.setState(ctx, providerName, StateOpen); err != nil {
			return fmt.Errorf("failed to reopen circuit: %w", err)
		}

		// Reset success counter
		successKey := cb.getKey(providerName, "success_count")
		if err := cb.redis.Del(ctx, successKey).Err(); err != nil {
			return fmt.Errorf("failed to reset success count: %w", err)
		}

		cb.logger.Warn("circuit breaker reopened after half-open failure",
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

// Helper: shouldTransitionToHalfOpen checks if timeout has expired
func (cb *RedisCircuitBreaker) shouldTransitionToHalfOpen(ctx context.Context, providerName string) bool {
	openedAt := cb.getTimestamp(ctx, cb.getKey(providerName, "opened_at"))
	if openedAt == nil {
		return true
	}

	cfg := cb.getConfig(providerName)
	return time.Since(*openedAt) >= cfg.Timeout
}

// Helper: resetCounters resets all counters for a provider
func (cb *RedisCircuitBreaker) resetCounters(ctx context.Context, providerName string) error {
	keys := []string{
		cb.getKey(providerName, "failure_count"),
		cb.getKey(providerName, "success_count"),
		cb.getKey(providerName, "opened_at"),
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
