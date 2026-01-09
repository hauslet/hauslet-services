package http

import (
	"log/slog"
	"time"

	"hauslet/cmd/api/server/middleware"
	"hauslet/internal/modules/verification/service"
	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
)

// WebhookHandler handles verification provider webhooks (Dojah, Veriff, etc.)
type WebhookHandler struct {
	verificationService service.VerificationService
	log                 *slog.Logger
}

// NewWebhookHandler creates a new verification webhook handler
func NewWebhookHandler(
	verificationService service.VerificationService,
	log *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		verificationService: verificationService,
		log:                 log,
	}
}

// SetupRoutes configures verification webhook routes (without rate limiting)
func (h *WebhookHandler) SetupRoutes(r chi.Router) {
	r.Post("/webhooks/verification/dojah", h.HandleDojahWebhook)
	r.Post("/webhooks/verification/veriff", h.HandleVeriffWebhook)
}

// SetupRoutesWithRateLimiting configures verification webhook routes with rate limiting
func (h *WebhookHandler) SetupRoutesWithRateLimiting(r chi.Router, limiter ratelimit.Limiter) {
	// Webhooks need higher limits than user endpoints
	webhookLimiter := middleware.RateLimitIP(limiter, 100, time.Minute)

	r.With(webhookLimiter).Post("/webhooks/verification/dojah", h.HandleDojahWebhook)
	r.With(webhookLimiter).Post("/webhooks/verification/veriff", h.HandleVeriffWebhook)
}
