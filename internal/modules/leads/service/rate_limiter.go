package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/leads/domain"
	"hauslet/internal/platform/ratelimit"
	"time"

	"github.com/google/uuid"
)

// DefaultRateLimitConfig defines default rate limits for leads.
var DefaultRateLimitConfig = RateLimitConfig{
	Window: 24 * time.Hour,
	Anonymous: RateLimitTier{
		PerEmail:        3,
		PerIP:           10,
		PerListingEmail: 5,
		PerListingIP:    5,
	},
	Authenticated: RateLimitTier{
		PerUser:         20,
		PerListingEmail: 10,
		PerListingIP:    10,
	},
}

// RateLimitTier defines rate limits for a requester tier.
type RateLimitTier struct {
	PerEmail        int64
	PerIP           int64
	PerListingEmail int64
	PerListingIP    int64
	PerUser         int64
}

// RateLimitConfig defines lead rate limits and window duration.
type RateLimitConfig struct {
	Window        time.Duration
	Anonymous     RateLimitTier
	Authenticated RateLimitTier
}

// RateLimiter handles Redis-based rate limiting for lead submissions.
type RateLimiter struct {
	limiter ratelimit.Limiter
	config  RateLimitConfig
}

// NewRateLimiter creates a new instance of RateLimiter
func NewRateLimiter(limiter ratelimit.Limiter, config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		limiter: limiter,
		config:  normalizeRateLimitConfig(config),
	}
}

// CheckRateLimit checks if a lead submission is within rate limits
// Returns an error if any rate limit is exceeded
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, userID *uuid.UUID, email, ipAddress string, listingID uuid.UUID) error {
	if rl == nil || rl.limiter == nil {
		return nil
	}

	keys := rl.buildKeys(userID, email, ipAddress, listingID)
	if len(keys) == 0 {
		return nil
	}

	results, err := rl.limiter.CheckMultiple(ctx, keys...)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	for _, result := range results {
		if !result.Allowed {
			return rateLimitErrorForKey(result.Key)
		}
	}

	for _, key := range keys {
		_, _ = rl.limiter.Increment(ctx, key)
	}

	return nil
}

// GetRateLimitStatus returns the current rate limit status for an email/IP combination
// Returns the number of leads submitted and the limits for each category
func (rl *RateLimiter) GetRateLimitStatus(ctx context.Context, userID *uuid.UUID, email, ipAddress string, listingID uuid.UUID) (*RateLimitStatus, error) {
	if rl == nil || rl.limiter == nil {
		return nil, fmt.Errorf("rate limiter not configured")
	}

	keys := rl.buildKeys(userID, email, ipAddress, listingID)
	if len(keys) == 0 {
		return nil, fmt.Errorf("no rate limit keys available")
	}

	status := &RateLimitStatus{}
	for _, key := range keys {
		count, err := rl.limiter.GetCurrent(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get rate limit count: %w", err)
		}
		remaining := int64(key.Limit) - count
		if remaining < 0 {
			remaining = 0
		}

		switch key.Type {
		case ratelimit.KeyTypeEmail:
			status.EmailCount = int(count)
			status.EmailLimit = int(key.Limit)
			status.EmailRemaining = int(remaining)
		case ratelimit.KeyTypeIP:
			status.IPCount = int(count)
			status.IPLimit = int(key.Limit)
			status.IPRemaining = int(remaining)
		case ratelimit.KeyTypeListingEmail, ratelimit.KeyTypeListingIP:
			status.ListingCount = int(count)
			status.ListingLimit = int(key.Limit)
			status.ListingRemaining = int(remaining)
		case ratelimit.KeyTypeUser:
			status.UserCount = int(count)
			status.UserLimit = int(key.Limit)
			status.UserRemaining = int(remaining)
		}
	}

	return status, nil
}

// RateLimitStatus represents the current rate limit status
type RateLimitStatus struct {
	EmailCount      int
	EmailLimit      int
	EmailRemaining  int
	IPCount         int
	IPLimit         int
	IPRemaining     int
	ListingCount    int
	ListingLimit    int
	ListingRemaining int
	UserCount       int
	UserLimit       int
	UserRemaining   int
}

func (rl *RateLimiter) buildKeys(userID *uuid.UUID, email, ipAddress string, listingID uuid.UUID) []ratelimit.LimitKey {
	tier := rl.config.Anonymous
	if userID != nil && *userID != uuid.Nil {
		tier = rl.config.Authenticated
	}

	keys := make([]ratelimit.LimitKey, 0, 5)
	window := rl.config.Window

	if userID != nil && *userID != uuid.Nil && tier.PerUser > 0 {
		keys = append(keys, ratelimit.LimitKey{
			Type:   ratelimit.KeyTypeUser,
			Value:  userID.String(),
			Limit:  tier.PerUser,
			Window: window,
		})
	}

	if email != "" && tier.PerEmail > 0 {
		keys = append(keys, ratelimit.LimitKey{
			Type:   ratelimit.KeyTypeEmail,
			Value:  email,
			Limit:  tier.PerEmail,
			Window: window,
		})
	}

	if ipAddress != "" && tier.PerIP > 0 {
		keys = append(keys, ratelimit.LimitKey{
			Type:   ratelimit.KeyTypeIP,
			Value:  ipAddress,
			Limit:  tier.PerIP,
			Window: window,
		})
	}

	if listingID != uuid.Nil && email != "" && tier.PerListingEmail > 0 {
		keys = append(keys, ratelimit.LimitKey{
			Type:   ratelimit.KeyTypeListingEmail,
			Value:  fmt.Sprintf("%s:%s", listingID.String(), email),
			Limit:  tier.PerListingEmail,
			Window: window,
		})
	}

	if listingID != uuid.Nil && ipAddress != "" && tier.PerListingIP > 0 {
		keys = append(keys, ratelimit.LimitKey{
			Type:   ratelimit.KeyTypeListingIP,
			Value:  fmt.Sprintf("%s:%s", listingID.String(), ipAddress),
			Limit:  tier.PerListingIP,
			Window: window,
		})
	}

	return keys
}

func normalizeRateLimitConfig(config RateLimitConfig) RateLimitConfig {
	if config.Window <= 0 {
		config.Window = DefaultRateLimitConfig.Window
	}

	if config.Anonymous.PerEmail <= 0 {
		config.Anonymous.PerEmail = DefaultRateLimitConfig.Anonymous.PerEmail
	}
	if config.Anonymous.PerIP <= 0 {
		config.Anonymous.PerIP = DefaultRateLimitConfig.Anonymous.PerIP
	}
	if config.Anonymous.PerListingEmail <= 0 {
		config.Anonymous.PerListingEmail = DefaultRateLimitConfig.Anonymous.PerListingEmail
	}
	if config.Anonymous.PerListingIP <= 0 {
		config.Anonymous.PerListingIP = DefaultRateLimitConfig.Anonymous.PerListingIP
	}

	if config.Authenticated.PerUser <= 0 {
		config.Authenticated.PerUser = DefaultRateLimitConfig.Authenticated.PerUser
	}
	if config.Authenticated.PerListingEmail <= 0 {
		config.Authenticated.PerListingEmail = DefaultRateLimitConfig.Authenticated.PerListingEmail
	}
	if config.Authenticated.PerListingIP <= 0 {
		config.Authenticated.PerListingIP = DefaultRateLimitConfig.Authenticated.PerListingIP
	}

	return config
}

func rateLimitErrorForKey(key ratelimit.LimitKey) error {
	switch key.Type {
	case ratelimit.KeyTypeEmail:
		return domain.ErrEmailRateLimitReached
	case ratelimit.KeyTypeIP:
		return domain.ErrIPRateLimitReached
	case ratelimit.KeyTypeListingEmail, ratelimit.KeyTypeListingIP:
		return domain.ErrListingRateLimitReached
	default:
		return domain.ErrRateLimitExceeded
	}
}
