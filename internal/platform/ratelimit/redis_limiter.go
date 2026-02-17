package ratelimit

import (
	"context"
	"fmt"
	"hauslet/internal/platform/redis"
	"strconv"
	"time"
)

//
// Redis-backed rate limiter using Lua (atomic, production-safe)
//

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

// Lua script
// KEYS[1] = redis key
// ARGV[1] = limit
// ARGV[2] = window (seconds)
//
// Returns: { allowed, current, remaining, ttl }
// allowed   -> 1 | 0
// current   -> current count
// remaining -> remaining allowed requests
// ttl       -> seconds until reset
var rateLimitScript = redis.NewScript(`
  local limit = tonumber(ARGV[1])
  local window = tonumber(ARGV[2])

  local current = redis.call("GET", KEYS[1])
  if current then
    current = tonumber(current)
  else
    current = 0
  end

  if current >= limit then
    local ttl = redis.call("TTL", KEYS[1])
    if ttl < 0 then
      ttl = window
    end
    return {0, current, 0, ttl}
  end

  current = redis.call("INCR", KEYS[1])

  if current == 1 then
    redis.call("EXPIRE", KEYS[1], window)
  end

  local ttl = redis.call("TTL", KEYS[1])
  if ttl < 0 then
    ttl = window
  end

  return {1, current, limit - current, ttl}
`)

// Multi-key atomic check-and-increment.
// KEYS[i]        = redis key for key i
// ARGV[2i-1]     = limit for key i
// ARGV[2i]       = window (seconds) for key i
//
// Returns on denied: {0, failed_index, c1, r1, t1, ..., ci, ri, ti}
// Returns on allowed: {1, c1, r1, t1, ..., cn, rn, tn}
var multiRateLimitScript = redis.NewScript(`
  local n = #KEYS
  local currents = {}

  -- First pass: read all keys, fail fast without touching anything
  for i = 1, n do
    local limit  = tonumber(ARGV[(i-1)*2 + 1])
    local window = tonumber(ARGV[(i-1)*2 + 2])
    local val    = redis.call("GET", KEYS[i])
    local current = val and tonumber(val) or 0
    currents[i] = current

    if current >= limit then
      -- Build denied result with state for keys 1..i (nothing was modified)
      local result = {0, i}
      for j = 1, i - 1 do
        local jlimit  = tonumber(ARGV[(j-1)*2 + 1])
        local jwindow = tonumber(ARGV[(j-1)*2 + 2])
        local jttl = redis.call("TTL", KEYS[j])
        if jttl < 0 then jttl = jwindow end
        result[#result+1] = currents[j]
        result[#result+1] = jlimit - currents[j]
        result[#result+1] = jttl
      end
      local ttl = redis.call("TTL", KEYS[i])
      if ttl < 0 then ttl = window end
      result[#result+1] = current
      result[#result+1] = 0
      result[#result+1] = ttl
      return result
    end
  end

  -- All passed: now increment everything
  local result = {1}
  for i = 1, n do
    local limit  = tonumber(ARGV[(i-1)*2 + 1])
    local window = tonumber(ARGV[(i-1)*2 + 2])
    local current = redis.call("INCR", KEYS[i])
    if current == 1 then
      redis.call("EXPIRE", KEYS[i], window)
    end
    local ttl = redis.call("TTL", KEYS[i])
    if ttl < 0 then ttl = window end
    result[#result+1] = current
    result[#result+1] = limit - current
    result[#result+1] = ttl
  end
  return result
`)

//
// =======================
// READ-ONLY (BEST EFFORT)
// =======================
//

// Check verifies if a key appears within limits (NOT atomic, informational only)
func (rl *RedisLimiter) Check(ctx context.Context, key LimitKey) (*CheckResult, error) {
	current, err := rl.GetCurrent(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get current count: %w", err)
	}

	allowed := current < key.Limit
	remaining := max(key.Limit-current, 0)

	var retryAt *time.Time
	if !allowed {
		ttl, err := rl.getTTL(ctx, key)
		if err == nil && ttl > 0 {
			t := time.Now().Add(ttl)
			retryAt = &t
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

// CheckMultiple is informational only (NOT enforcement)
func (rl *RedisLimiter) CheckMultiple(ctx context.Context, keys ...LimitKey) ([]*CheckResult, error) {
	results := make([]*CheckResult, len(keys))

	for i, key := range keys {
		res, err := rl.Check(ctx, key)
		if err != nil {
			return nil, err
		}
		results[i] = res

		if !res.Allowed {
			return results[:i+1], nil
		}
	}

	return results, nil
}

//
// =======================
// ENFORCEMENT (ATOMIC)
// =======================
//

// CheckAndIncrement atomically checks and consumes a request
func (rl *RedisLimiter) CheckAndIncrement(ctx context.Context, key LimitKey) (*CheckResult, error) {
	r, err := rl.runRateLimitScript(ctx, key)
	if err != nil {
		return nil, err
	}

	var retryAt *time.Time
	if !r.allowed && r.ttl > 0 {
		t := time.Now().Add(r.ttl)
		retryAt = &t
	}

	return &CheckResult{
		Allowed:   r.allowed,
		Key:       key,
		Current:   r.current,
		Limit:     key.Limit,
		Remaining: r.remaining,
		RetryAt:   retryAt,
		Window:    key.Window,
	}, nil
}

// CheckAndIncrementMultiple enforces ALL limits atomically (short-circuits on failure)
func (rl *RedisLimiter) CheckAndIncrementMultiple(
	ctx context.Context,
	keys ...LimitKey,
) ([]*CheckResult, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	redisKeys := make([]string, len(keys))
	args := make([]any, len(keys)*2)
	for i, key := range keys {
		redisKeys[i] = key.RedisKey()
		args[i*2] = key.Limit
		args[i*2+1] = int64(key.Window.Seconds())
	}

	res, err := multiRateLimitScript.Run(ctx, rl.redis, redisKeys, args...).Result()
	if err != nil {
		return nil, err
	}

	values, ok := res.([]any)
	if !ok || len(values) < 1 {
		return nil, fmt.Errorf("unexpected multi-key script result: %v", res)
	}

	if toInt64(values[0]) == 0 {
		// Denied. values = {0, failed_index, c1, r1, t1, ..., ci, ri, ti}
		if len(values) < 2 {
			return nil, fmt.Errorf("truncated denied result: %v", res)
		}
		failedIdx := int(toInt64(values[1])) // 1-indexed
		results := make([]*CheckResult, failedIdx)

		for i := 0; i < failedIdx; i++ {
			offset := 2 + i*3
			if len(values) < offset+3 {
				return nil, fmt.Errorf("truncated denied result at key %d: %v", i+1, res)
			}
			key := keys[i]
			isAllowed := i < failedIdx-1
			current := toInt64(values[offset])
			remaining := max(toInt64(values[offset+1]), 0)
			ttl := time.Duration(max(toInt64(values[offset+2]), 0)) * time.Second

			var retryAt *time.Time
			if !isAllowed && ttl > 0 {
				t := time.Now().Add(ttl)
				retryAt = &t
			}

			results[i] = &CheckResult{
				Allowed:   isAllowed,
				Key:       key,
				Current:   current,
				Limit:     key.Limit,
				Remaining: remaining,
				RetryAt:   retryAt,
				Window:    key.Window,
			}
		}
		return results, nil
	}

	// All allowed. values = {1, c1, r1, t1, ..., cn, rn, tn}
	results := make([]*CheckResult, len(keys))
	for i, key := range keys {
		offset := 1 + i*3
		if len(values) < offset+3 {
			return nil, fmt.Errorf("truncated allowed result at key %d: %v", i+1, res)
		}
		results[i] = &CheckResult{
			Allowed:   true,
			Key:       key,
			Current:   toInt64(values[offset]),
			Limit:     key.Limit,
			Remaining: max(toInt64(values[offset+1]), 0),
			Window:    key.Window,
		}
	}
	return results, nil
}

// Increment increments ONLY if allowed.
// Returns -1 if rate-limited.
func (rl *RedisLimiter) Increment(ctx context.Context, key LimitKey) (int64, error) {
	r, err := rl.runRateLimitScript(ctx, key)
	if err != nil {
		return 0, err
	}

	if !r.allowed {
		return -1, nil
	}

	return r.current, nil
}

//
// =======================
// REDIS HELPERS
// =======================
//

// Reset deletes the counter for a key
func (rl *RedisLimiter) Reset(ctx context.Context, key LimitKey) error {
	return rl.redis.Del(ctx, key.RedisKey()).Err()
}

// GetCurrent returns the current count (read-only)
func (rl *RedisLimiter) GetCurrent(ctx context.Context, key LimitKey) (int64, error) {
	val, err := rl.redis.Get(ctx, key.RedisKey()).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

// getTTL returns remaining TTL
func (rl *RedisLimiter) getTTL(ctx context.Context, key LimitKey) (time.Duration, error) {
	ttl, err := rl.redis.TTL(ctx, key.RedisKey()).Result()
	if err != nil || ttl <= 0 {
		return key.Window, nil
	}
	return ttl, nil
}

//
// =======================
// INTERNAL LUA HANDLING
// =======================
//

type rateLimitScriptResult struct {
	allowed   bool
	current   int64
	remaining int64
	ttl       time.Duration
}

func (rl *RedisLimiter) runRateLimitScript(
	ctx context.Context,
	key LimitKey,
) (*rateLimitScriptResult, error) {

	res, err := rateLimitScript.Run(
		ctx,
		rl.redis,
		[]string{key.RedisKey()},
		key.Limit,
		int64(key.Window.Seconds()),
	).Result()

	if err != nil {
		return nil, err
	}

	return parseRateLimitScriptResult(res)
}

func parseRateLimitScriptResult(res any) (*rateLimitScriptResult, error) {
	values, ok := res.([]any)
	if !ok || len(values) != 4 {
		return nil, fmt.Errorf("unexpected script result: %v", res)
	}

	allowed := toInt64(values[0]) == 1
	current := toInt64(values[1])
	remaining := max(toInt64(values[2]), 0)
	ttl := time.Duration(max(toInt64(values[3]), 0)) * time.Second

	return &rateLimitScriptResult{
		allowed:   allowed,
		current:   current,
		remaining: remaining,
		ttl:       ttl,
	}, nil
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case string:
		i, _ := strconv.ParseInt(t, 10, 64)
		return i
	case []byte:
		i, _ := strconv.ParseInt(string(t), 10, 64)
		return i
	default:
		return 0
	}
}
