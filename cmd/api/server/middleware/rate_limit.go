package middleware

import (
	"encoding/json"
	"fmt"

	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitConfig holds the configuration for rate limiting
type RateLimitConfig struct {
	Requests int           // Number of requests allowed
	Window   time.Duration // Time window for the requests
}

// RateLimit creates a middleware that limits requests based on IP address using Redis
func RateLimit(limitConfig RateLimitConfig, redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get client IP
			ip := r.RemoteAddr
			if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				ip = forwardedFor
			}

			// Create Redis key for this IP
			key := fmt.Sprintf("rate_limit:%s", ip)

			// Get current count
			count, err := redisClient.Get(ctx, key).Int()
			if err != nil && err != redis.Nil {
				// Log error but allow request to proceed
				next.ServeHTTP(w, r)
				return
			}

			// If key doesn't exist, create it
			if err == redis.Nil {
				err = redisClient.Set(ctx, key, 1, limitConfig.Window).Err()
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
			} else if count >= limitConfig.Requests {
				// Rate limit exceeded
				responseBody := map[string]any{
					"message": fmt.Sprintf("Rate limit exceeded. Try again in %v", limitConfig.Window),
					"title":   "Too Many Requests",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(responseBody)
				return
			} else {
				// Increment counter
				err = redisClient.Incr(ctx, key).Err()
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
