package http

import (
	"context"
	"encoding/json"
	"errors"
	paymentdomain "hauslet/internal/modules/payments/domain"
	paymentJob "hauslet/internal/queue/jobs/payments"
	"io"
	"net/http"
	"strings"
	"time"
)

// HandlePaystackWebhook processes Paystack webhook events
func (h *WebhookHandler) HandlePaystackWebhook(w http.ResponseWriter, r *http.Request) {
	h.log.Info(" received Paystack webhook")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read webhook body", "error", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify webhook signature
	signature := r.Header.Get("X-Paystack-Signature")
	if signature == "" {
		h.log.Warn("webhook received without signature")
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	valid, err := h.paymentClient.VerifyWebhookSignature("paystack", signature, body)
	if err != nil || !valid {
		h.log.Error("webhook signature verification failed", "error", err)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Minimal parse for sanity checks
	var envelope struct {
		Event string `json:"event"`
		Data  struct {
			Reference            string `json:"reference"`
			TransactionReference string `json:"transaction_reference"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		h.log.Error("failed to parse webhook envelope", "error", err)
		http.Error(w, "Invalid webhook data", http.StatusBadRequest)
		return
	}

	eventType := strings.TrimSpace(envelope.Event)
	reference := strings.TrimSpace(envelope.Data.Reference)
	if reference == "" {
		reference = strings.TrimSpace(envelope.Data.TransactionReference)
	}

	if eventType == "" || reference == "" {
		h.log.Error("webhook missing event or reference", "event", eventType, "ref", reference)
		http.Error(w, "Invalid webhook data", http.StatusBadRequest)
		return
	}

	if isPaymentEvent(eventType) {
		if _, err := h.paymentService.GetPaymentByReference(r.Context(), reference); err != nil {
			if errors.Is(err, paymentdomain.ErrPaymentNotFound) {
				h.log.Warn("webhook reference not found", "event", eventType, "ref", reference)
				http.Error(w, "Unknown reference", http.StatusNotFound)
				return
			}
			h.log.Error("webhook reference lookup failed", "error", err)
			http.Error(w, "Failed to validate reference", http.StatusInternalServerError)
			return
		}
	}

	if h.queueClient != nil && h.queueSubject != "" {
		job := paymentJob.PaymentWebhookJob{
			Provider:   "paystack",
			EventType:  eventType,
			Reference:  reference,
			Payload:    json.RawMessage(body),
			ReceivedAt: time.Now(),
		}

		pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.queueClient.Publish(pubCtx, h.queueSubject, job); err != nil {
			h.log.Warn("failed to publish payment webhook job", "error", err)
			if !h.queueClient.AllowFallback() {
				http.Error(w, "Queue unavailable", http.StatusServiceUnavailable)
				return
			}
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "queued",
			})
			return
		}
	}

	// Parse webhook event for synchronous fallback
	event, err := h.paymentClient.ParseWebhookEvent("paystack", body)
	if err != nil {
		h.log.Error("failed to parse webhook event", "error", err)
		http.Error(w, "Invalid webhook data", http.StatusBadRequest)
		return
	}

	if event.Reference == "" {
		event.Reference = reference
	}

	h.log.Info(" processing webhook event", "type", event.Type, "ref", event.Reference)
	// Handle different event types
	if err := h.ProcessEvent(r.Context(), event); err != nil {
		h.log.Error("failed to process webhook event", "error", err)
		http.Error(w, "Failed to process event", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}
