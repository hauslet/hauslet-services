package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
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
	applyRateLimit := func(config middleware.RateLimitConfig) func(http.Handler) http.Handler {
		policy := middleware.RateLimitPolicy{
			Keys: func(r *http.Request) []ratelimit.LimitKey {
				ip := middleware.ClientIP(r)
				if ip == "" {
					return nil
				}
				return []ratelimit.LimitKey{
					{
						Type:   ratelimit.KeyTypeIP,
						Value:  ip,
						Limit:  int64(config.Requests),
						Window: config.Window,
					},
				}
			},
		}
		return middleware.RateLimitWithLimiter(limiter, policy)
	}

	// Webhooks need higher limits than user endpoints
	webhookRateLimit := applyRateLimit(middleware.RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
	})

	r.With(webhookRateLimit).Post("/webhooks/verification/dojah", h.HandleDojahWebhook)
	r.With(webhookRateLimit).Post("/webhooks/verification/veriff", h.HandleVeriffWebhook)
}

// HandleDojahWebhook processes Dojah KYC provider webhooks
func (h *WebhookHandler) HandleDojahWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleProviderWebhook(w, r, "dojah")
}

// HandleVeriffWebhook processes Veriff KYC provider webhooks
func (h *WebhookHandler) HandleVeriffWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleProviderWebhook(w, r, "veriff")
}

// handleProviderWebhook is the common handler for all KYC provider webhooks
func (h *WebhookHandler) handleProviderWebhook(w http.ResponseWriter, r *http.Request, providerName string) {
	h.log.Info("received verification webhook",
		"provider", providerName,
		"content_type", r.Header.Get("Content-Type"),
	)

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read webhook body",
			"provider", providerName,
			"error", err,
		)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Extract signature based on provider
	signature := h.extractSignature(r, providerName)

	// Collect headers for webhook parsing (normalize to lowercase)
	headers := make(map[string]string)
	for key := range r.Header {
		headers[strings.ToLower(key)] = r.Header.Get(key)
	}

	// Build request for service
	req := service.ProcessWebhookRequest{
		ProviderName: providerName,
		Payload:      body,
		Headers:      headers,
		Signature:    signature,
	}

	// Process webhook via service (handles signature verification internally)
	if err := h.verificationService.ProcessWebhook(r.Context(), req); err != nil {
		h.log.Error("failed to process verification webhook",
			"provider", providerName,
			"error", err,
		)

		// Return appropriate status based on error type
		switch err.Error() {
		case "invalid webhook signature":
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		default:
			// Still return 200 to prevent retries for business logic errors
			// Log the error but acknowledge receipt
			h.log.Warn("webhook processing failed but acknowledged",
				"provider", providerName,
				"error", err,
			)
		}
	}

	h.log.Info("verification webhook processed successfully",
		"provider", providerName,
	)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "received",
	})
}

// extractSignature extracts the webhook signature based on provider conventions
func (h *WebhookHandler) extractSignature(r *http.Request, providerName string) string {
	switch providerName {
	case "dojah":
		// Dojah uses x-dojah-signature or x-dojah-signature-v2 header
		sig := r.Header.Get("x-dojah-signature")
		if sig == "" {
			sig = r.Header.Get("x-dojah-signature-v2")
		}
		return sig
	case "veriff":
		// Veriff uses x-hmac-signature header
		return r.Header.Get("x-hmac-signature")
	default:
		// Generic signature header
		return r.Header.Get("x-webhook-signature")
	}
}
