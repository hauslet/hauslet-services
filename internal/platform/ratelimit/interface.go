package ratelimit

import "context"

// Limiter defines the interface for rate limiting operations
type Limiter interface {
	// Check verifies if a single key is within rate limits
	Check(ctx context.Context, key LimitKey) (*CheckResult, error)

	// CheckAndIncrement atomically checks and increments a key
	CheckAndIncrement(ctx context.Context, key LimitKey) (*CheckResult, error)

	// CheckMultiple verifies multiple keys (all must pass)
	CheckMultiple(ctx context.Context, keys ...LimitKey) ([]*CheckResult, error)

	// Increment increments the counter for a key
	Increment(ctx context.Context, key LimitKey) (int64, error)

	// Reset resets the counter for a key
	Reset(ctx context.Context, key LimitKey) error

	// GetCurrent returns the current count for a key
	GetCurrent(ctx context.Context, key LimitKey) (int64, error)
}
