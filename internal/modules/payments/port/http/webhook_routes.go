package http

import (
	"hauslet/cmd/api/server/middleware"
	"hauslet/internal/platform/ratelimit"
	"time"

	"github.com/go-chi/chi/v5"
)

// SetupRoutes configures webhook routes (without rate limiting)
func (h *WebhookHandler) SetupRoutes(r chi.Router) {
	// Public webhook endpoint (no authentication required)
	r.Post("/webhooks/paystack", h.HandlePaystackWebhook)
}

// SetupRoutesWithRateLimiting configures webhook routes with rate limiting for production
func (h *WebhookHandler) SetupRoutesWithRateLimiting(r chi.Router, limiter ratelimit.Limiter) {

	// Webhook endpoint with rate limiting
	// Webhooks need higher limits than user endpoints (100/min)
	// Too strict breaks legitimate provider webhooks, too loose allows DoS
	r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
		Post("/webhooks/paystack", h.HandlePaystackWebhook)
}
