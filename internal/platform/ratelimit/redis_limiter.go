package ratelimit

import (
	"context"
	"fmt"
	"hauslet/internal/platform/redis"
	"time"
)

// RedisLimiter implements Limiter using Redis
type RedisLimiter struct {
	redis redis.RedisClient
}

// NewRedisLimiter creates a new Redis-backed rate limiter
func NewRedisLimiter(redisClient redis.RedisClient) *RedisLimiter {
	return &RedisLimiter{
		redis: redisClient,
	}
}

// Check verifies if a single key is within rate limits
func (rl *RedisLimiter) Check(ctx context.Context, key LimitKey) (*CheckResult, error) {
	// Get current count
	current, err := rl.GetCurrent(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get current count: %w", err)
	}

	// Calculate remaining
	remaining := max(key.Limit - current, 0)

	// Check if allowed
	allowed := current < key.Limit

	// Calculate retry time if rate limited
	var retryAt *time.Time
	if !allowed {
		// Get TTL to calculate when limit resets
		ttl, err := rl.getTTL(ctx, key)
		if err == nil && ttl > 0 {
			retry := time.Now().Add(ttl)
			retryAt = &retry
		}
	}

	return &CheckResult{
		Allowed:   allowed,
		Key:       key,
		Current:   current,
		Limit:     key.Limit,
		Remaining: remaining,
		RetryAt:   retryAt,
		Window:    key.Window,
	}, nil
}

// CheckMultiple verifies multiple keys (all must pass)
func (rl *RedisLimiter) CheckMultiple(ctx context.Context, keys ...LimitKey) ([]*CheckResult, error) {
	results := make([]*CheckResult, len(keys))

	for i, key := range keys {
		result, err := rl.Check(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to check key %s:%s: %w", key.Type, key.Value, err)
		}
		results[i] = result

		// If any key fails, return early
		if !result.Allowed {
			return results[:i+1], nil
		}
	}

	return results, nil
}

// Increment increments the counter for a key and returns new value
func (rl *RedisLimiter) Increment(ctx context.Context, key LimitKey) (int64, error) {
	redisKey := key.RedisKey()

	// Increment counter
	count, err := rl.redis.Incr(ctx, redisKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Set expiration on first increment
	if count == 1 {
		if err := rl.redis.Expire(ctx, redisKey, key.Window).Err(); err != nil {
			return count, fmt.Errorf("failed to set expiration: %w", err)
		}
	}

	return count, nil
}

// Reset resets the counter for a key
func (rl *RedisLimiter) Reset(ctx context.Context, key LimitKey) error {
	redisKey := key.RedisKey()

	if err := rl.redis.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to reset counter: %w", err)
	}

	return nil
}

// GetCurrent returns the current count for a key
func (rl *RedisLimiter) GetCurrent(ctx context.Context, key LimitKey) (int64, error) {
	redisKey := key.RedisKey()

	// Get current value
	val, err := rl.redis.Get(ctx, redisKey).Result()
	if err != nil {
		// Key doesn't exist yet
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get current count: %w", err)
	}

	// Parse value
	var count int64
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	return count, nil
}

// Helper: getTTL gets the TTL for a key
func (rl *RedisLimiter) getTTL(ctx context.Context, key LimitKey) (time.Duration, error) {
	redisKey := key.RedisKey()

	ttl, err := rl.redis.TTL(ctx, redisKey).Result()
	if err != nil {
		return key.Window, fmt.Errorf("failed to read TTL: %w", err)
	}

	// Redis returns -2 when key does not exist, -1 when key exists without expiry.
	if ttl <= 0 {
		return key.Window, nil
	}

	return ttl, nil
}
