package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"hauslet/internal/modules/verification/service"
)

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
		h.log.Error("failed to read webhook body", "provider", providerName, "error", err)
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
		h.log.Error("failed to process verification webhook", "provider", providerName, "error", err)

		// Return appropriate status based on error type
		switch err.Error() {
		case "invalid webhook signature":
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		default:
			// Still return 200 to prevent retries for business logic errors
			// Log the error but acknowledge receipt
			h.log.Warn("webhook processing failed but acknowledged", "provider", providerName, "error", err)
		}
	}

	h.log.Info("verification webhook processed successfully", "provider", providerName)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "received",
	})
}
