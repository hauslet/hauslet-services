package redis

import (
	"context"
	"fmt"
	"hauslet/config"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient is an interface that both the real Redis client and mock can implement
type RedisClient interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Publish(ctx context.Context, channel string, message any) *redis.IntCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	SAdd(ctx context.Context, key string, members ...any) *redis.IntCmd
	SRem(ctx context.Context, key string, members ...any) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	// List operations
	RPush(ctx context.Context, key string, values ...any) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
}

var (
	redisClient RedisClient
	once        sync.Once
)

// InitRedis initializes the Redis client
func InitRedis(cfg *config.RedisConfig, ctx context.Context) error {
	var initErr error // 1. Declare error variable outside the closure

	once.Do(func() {
		var opt *redis.Options
		var err error

		if strings.HasPrefix(cfg.Addr, "redis://") {
			// Use full redis:// URL
			opt, err = redis.ParseURL(cfg.Addr)
			if err != nil {
				// 2. Assign to the outer variable instead of returning
				initErr = fmt.Errorf("failed to parse Redis URL: %v", err)
				return
			}
		} else {
			// Fallback to plain host:port (local dev)
			opt = &redis.Options{Addr: cfg.Addr}
		}

		redisClient = redis.NewClient(opt)

		// Test connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			// 2. Assign to the outer variable
			initErr = fmt.Errorf("failed to connect to Redis: %v", err)
			return
		}
	})

	// 3. Return the captured error
	return initErr
}

// GetRedis returns the singleton Redis client
func GetRedis() (RedisClient, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Redis client is not initialized. Call InitRedis() first.")
	}
	return redisClient, nil
}

// SetRedis sets the Redis client (for testing)
func SetRedis(client RedisClient) {
	redisClient = client
}

// CloseRedis closes the Redis client connection
func CloseRedis() error {
	if redisClient != nil {
		if r, ok := redisClient.(*redis.Client); ok {
			if err := r.Close(); err != nil {
				return fmt.Errorf("failed to close Redis client: %v", err)
			}
		}
	}
	return nil
}
