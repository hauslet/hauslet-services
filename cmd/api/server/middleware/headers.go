package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable HSTS (HTTP Strict Transport Security)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Stronger Content Security Policy
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self'; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' data:; "+
				"connect-src 'self'; "+
				"media-src 'self'; "+
				"frame-src 'self'; "+
				"object-src 'self'; "+
				"base-uri 'none'; "+
				"frame-ancestors 'none';")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Disable caching for sensitive content
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		next.ServeHTTP(w, r)
	})
}
