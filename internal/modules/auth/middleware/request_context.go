package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// contextKey is unexported to prevent collisions
type contextKey struct{}

var requestMetaKey = contextKey{}

// RequestMeta holds the extracted metadata
type RequestMeta struct {
	IP        string
	UserAgent string
	Referrer  string
}

// RequestContext middleware extracts IP, User-Agent, and Referrer
// and stores them in context for use in downstream handlers.
func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := RequestMeta{
			IP:        extractIP(r),
			UserAgent: r.Header.Get("User-Agent"),
			Referrer:  r.Referer(),
		}

		// Store the entire struct in one go, avoiding multiple context wraps
		ctx := context.WithValue(r.Context(), requestMetaKey, meta)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractIP extracts the client IP.
// NOTE: Only trust headers like X-Forwarded-For if you are behind a trusted proxy.
func extractIP(r *http.Request) string {
	// 1. Try X-Forwarded-For (Standard for proxies/LBs)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Taking the first IP is standard for obtaining the original client,
		// but see the security warning below.
		if ip, _, found := strings.Cut(forwarded, ","); found {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(forwarded)
	}

	// 2. Try X-Real-IP (Nginx standard)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// 3. Fallback to RemoteAddr
	// net.SplitHostPort safely handles "IP:Port" for both IPv4 and IPv6
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If RemoteAddr doesn't have a port (rare in HTTP), return as is
		return r.RemoteAddr
	}

	return ip
}

// GetRequestMeta retrieves the metadata struct from context
func GetRequestMeta(ctx context.Context) RequestMeta {
	if meta, ok := ctx.Value(requestMetaKey).(RequestMeta); ok {
		return meta
	}
	return RequestMeta{}
}

// GetIPFromContext helper to just get IP
func GetIPFromContext(ctx context.Context) string {
	return GetRequestMeta(ctx).IP
}

// GetUserAgentFromContext helper to just get UA
func GetUserAgentFromContext(ctx context.Context) string {
	return GetRequestMeta(ctx).UserAgent
}

// GetReferrerFromContext helper to just get Referrer
func GetReferrerFromContext(ctx context.Context) string {
	return GetRequestMeta(ctx).Referrer
}
