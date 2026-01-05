package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"hauslet/internal/platform/ratelimit"
)

// RateLimitConfig holds the configuration for rate limiting
type RateLimitConfig struct {
	Requests int           // Number of requests allowed
	Window   time.Duration // Time window for the requests
}

// RateLimitPolicy defines how to build limit keys for a request.
type RateLimitPolicy struct {
	Keys func(r *http.Request) []ratelimit.LimitKey
}

// RateLimitWithLimiter enforces rate limits using the platform limiter.
func RateLimitWithLimiter(limiter ratelimit.Limiter, policy RateLimitPolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil || policy.Keys == nil {
				next.ServeHTTP(w, r)
				return
			}

			keys := policy.Keys(r)
			if len(keys) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			results, err := limiter.CheckMultiple(r.Context(), keys...)
			if err != nil {
				// Fail open on limiter errors.
				next.ServeHTTP(w, r)
				return
			}

			for _, result := range results {
				if !result.Allowed {
					writeRateLimitResponse(w, result.Window)
					return
				}
			}

			for _, key := range keys {
				count, err := limiter.Increment(r.Context(), key)
				if err != nil {
					// Fail open on limiter errors.
					next.ServeHTTP(w, r)
					return
				}
				if count < 0 {
					writeRateLimitResponse(w, key.Window)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP returns the best-effort client IP for rate limiting.
func ClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}

func writeRateLimitResponse(w http.ResponseWriter, window time.Duration) {
	responseBody := map[string]any{
		"message": fmt.Sprintf("Rate limit exceeded. Try again in %v", window),
		"title":   "Too Many Requests",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(responseBody)
}
