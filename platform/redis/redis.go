package redis

import (
	"context"
	"fmt"
	"hauslet/config"
	"log"
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
}

var (
	redisClient RedisClient
	once        sync.Once
)

// InitRedis initializes the Redis client
func InitRedis(cfg *config.RedisConfig, ctx context.Context) {
	once.Do(func() {
		var opt *redis.Options
		var err error

		if strings.HasPrefix(cfg.Addr, "redis://") {
			// Use full redis:// URL
			opt, err = redis.ParseURL(cfg.Addr)
			if err != nil {
				panic(fmt.Sprintf("failed to parse Redis URL: %v", err))
			}
		} else {
			// Fallback to plain host:port (local dev)
			opt = &redis.Options{Addr: cfg.Addr}
		}

		redisClient = redis.NewClient(opt)

		// Test connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			panic(fmt.Sprintf("failed to connect to Redis: %v", err))
		}

		log.Println("Redis is up and running")
	})
}

// GetRedis returns the singleton Redis client
func GetRedis() RedisClient {
	if redisClient == nil {
		panic("Redis client is not initialized. Call InitRedis() first.")
	}
	return redisClient
}

// SetRedis sets the Redis client (for testing)
func SetRedis(client RedisClient) {
	redisClient = client
}

// CloseRedis closes the Redis client connection
func CloseRedis() {
	if redisClient != nil {
		if r, ok := redisClient.(*redis.Client); ok {
			if err := r.Close(); err != nil {
				log.Printf("failed to close Redis client: %v", err)
			}
		}
	}
}
