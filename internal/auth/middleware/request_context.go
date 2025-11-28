package middleware

import (
	"context"
	"net/http"
	"strings"
)

// Context keys for request metadata
type contextKey string

const (
	contextKeyIP        contextKey = "request_ip"
	contextKeyUserAgent contextKey = "request_user_agent"
)

// RequestContext middleware extracts IP and User-Agent from request
// and stores them in context for use in authentication flows
func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP address
		ip := extractIP(r)

		// Extract User-Agent
		userAgent := r.Header.Get("User-Agent")

		// Add to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextKeyIP, ip)
		ctx = context.WithValue(ctx, contextKeyUserAgent, userAgent)

		// Continue with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractIP extracts the real client IP from the request
// Checks X-Forwarded-For, X-Real-IP, and falls back to RemoteAddr
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common with proxies/load balancers)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first (client)
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (common with nginx)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	// RemoteAddr format is "IP:port", extract just IP
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// GetIPFromContext retrieves the IP address from context
func GetIPFromContext(ctx context.Context) string {
	if ip, ok := ctx.Value(contextKeyIP).(string); ok {
		return ip
	}
	return ""
}

// GetUserAgentFromContext retrieves the User-Agent from context
func GetUserAgentFromContext(ctx context.Context) string {
	if ua, ok := ctx.Value(contextKeyUserAgent).(string); ok {
		return ua
	}
	return ""
}
