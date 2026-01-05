package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hauslet/cmd/api/server/middleware"
	paymentdomain "hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/service"
	"hauslet/internal/platform/payment"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	paymentJob "hauslet/internal/queue/jobs/payments"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// BookingHooks defines the interface for booking lifecycle callbacks.
type BookingHooks interface {
	OnPaymentSucceeded(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment) error
	OnPaymentFailed(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment, reason string) error
	OnPaymentRefunded(ctx context.Context, bookingID uuid.UUID, payment *paymentdomain.Payment) error
}

// FinanceHooks defines the interface for finance module callbacks
type FinanceHooks interface {
	OnPaymentSucceeded(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error
	OnRefundProcessed(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) error
	OnBookingCompleted(ctx context.Context, bookingID, hostID uuid.UUID) error
}

// PayoutHooks defines the interface for payout/disbursement callbacks
type PayoutHooks interface {
	OnTransferSuccess(ctx context.Context, transferCode string) error
	OnTransferFailed(ctx context.Context, transferCode string, reason string) error
}

// PromotionHooks defines the interface for promotion/subscription lifecycle callbacks
type PromotionHooks interface {
	OnPromotionPaymentSucceeded(ctx context.Context, promotionID uuid.UUID, payment *paymentdomain.Payment) error
	OnSubscriptionPaymentSucceeded(ctx context.Context, subscriptionID uuid.UUID, payment *paymentdomain.Payment) error
}

// WebhookHandler handles payment provider webhooks
type WebhookHandler struct {
	paymentService service.PaymentService
	paymentClient  *payment.Client
	bookingHooks   BookingHooks
	financeHooks   FinanceHooks
	payoutHooks    PayoutHooks
	promotionHooks PromotionHooks
	queueClient    *platformQueue.Client
	queueSubject   string
	log            *slog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
	paymentService service.PaymentService,
	paymentClient *payment.Client,
	bookingHooks BookingHooks,
	financeHooks FinanceHooks,
	payoutHooks PayoutHooks,
	promotionHooks PromotionHooks,
	queueClient *platformQueue.Client,
	queueSubject string,
	log *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
		paymentClient:  paymentClient,
		bookingHooks:   bookingHooks,
		financeHooks:   financeHooks,
		payoutHooks:    payoutHooks,
		promotionHooks: promotionHooks,
		queueClient:    queueClient,
		queueSubject:   queueSubject,
		log:            log,
	}
}

// SetupRoutes configures webhook routes (without rate limiting)
func (h *WebhookHandler) SetupRoutes(r chi.Router) {
	// Public webhook endpoint (no authentication required)
	r.Post("/webhooks/paystack", h.HandlePaystackWebhook)
}

// SetupRoutesWithRateLimiting configures webhook routes with rate limiting for production
func (h *WebhookHandler) SetupRoutesWithRateLimiting(r chi.Router, redisClient redis.RedisClient) {
	// Helper to apply rate limiting
	applyRateLimit := func(config middleware.RateLimitConfig) func(http.Handler) http.Handler {
		return middleware.RateLimit(config, redisClient)
	}

	// Webhook endpoint with rate limiting
	// Webhooks need higher limits than user endpoints (100/min)
	// Too strict breaks legitimate provider webhooks, too loose allows DoS
	r.With(applyRateLimit(middleware.RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
	})).Post("/webhooks/paystack", h.HandlePaystackWebhook)
}

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

// ProcessEvent handles a parsed webhook event.
func (h *WebhookHandler) ProcessEvent(ctx context.Context, event *payment.UnifiedEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	switch event.Type {
	case "charge.success":
		return h.handleChargeSuccess(ctx, event)
	case "charge.failed":
		return h.handleChargeFailed(ctx, event)
	case "transfer.success":
		return h.handleTransferSuccess(ctx, event)
	case "transfer.failed":
		return h.handleTransferFailed(ctx, event)
	case "refund.processed":
		return h.handleRefundProcessed(ctx, event)
	case "refund.failed":
		return h.handleRefundFailed(ctx, event)
	default:
		if h.log != nil {
			h.log.Info(" unhandled webhook event type", "type", event.Type)
		}
		return nil
	}
}

func isPaymentEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "charge.") || strings.HasPrefix(eventType, "refund.")
}

// handleChargeSuccess handles successful payment webhook
func (h *WebhookHandler) handleChargeSuccess(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Info(" handling charge.success for ref", "ref", event.Reference)

	// Verify payment status
	pmt, err := h.paymentService.VerifyPayment(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to verify payment: %w", err)
	}

	h.log.Info(" payment verified", "id", pmt.ID, "status", pmt.Status)

	// Extract and save authorization code from webhook for card tokenization
	if err := h.extractAndSaveAuthorization(ctx, event.RawData, pmt); err != nil {
		h.log.Warn("failed to extract authorization code", "error", err)
		// Don't fail the webhook - payment already succeeded
		// Authorization saving is optional for future one-click payments
	}

	// If payment is for a booking, record in finance ledger FIRST, then notify booking
	if pmt.BookingID != nil {
		// Finance records transaction FIRST (idempotent)
		if h.financeHooks != nil {
			h.log.Info(" recording charge in finance", "booking", *pmt.BookingID, "amount", pmt.Amount)

			if err := h.financeHooks.OnPaymentSucceeded(ctx, *pmt.BookingID, pmt.ID, pmt.Amount, string(pmt.Currency)); err != nil {
				h.log.Error("failed to record charge in finance", "error", err)
				// Continue - don't fail webhook, but alert admin
				// Finance ledger can be corrected manually
			}
		}

		// Then update booking status
		if h.bookingHooks != nil {
			h.log.Info(" notifying booking module of payment success", "booking", *pmt.BookingID)

			if err := h.bookingHooks.OnPaymentSucceeded(ctx, *pmt.BookingID, pmt); err != nil {
				h.log.Error("failed to notify booking of payment success", "error", err)
				// Don't fail the webhook - payment already succeeded
				// The booking can be confirmed manually or via retry
			}
		}
	}

	// If payment is for a promotion, activate it
	if pmt.ResourceType == paymentdomain.ResourceTypePromotion && pmt.ResourceID != nil {
		if h.promotionHooks != nil {
			h.log.Info(" activating promotion after payment success",
				"promotion", *pmt.ResourceID,
				"amount", pmt.Amount,
				"payment_id", pmt.ID,
			)

			if err := h.promotionHooks.OnPromotionPaymentSucceeded(ctx, *pmt.ResourceID, pmt); err != nil {
				h.log.Error("failed to activate promotion", "error", err)
				// Don't fail the webhook - payment already succeeded
				// The promotion can be activated manually or via retry
			}
		} else {
			h.log.Warn("promotion hooks not configured, payment succeeded but promotion not activated",
				"promotion", *pmt.ResourceID,
			)
		}
	}

	// If payment is for a subscription, activate it
	if pmt.ResourceType == paymentdomain.ResourceTypeSubscription && pmt.ResourceID != nil {
		if h.promotionHooks != nil {
			h.log.Info(" activating subscription after payment success",
				"subscription", *pmt.ResourceID,
				"amount", pmt.Amount,
				"payment_id", pmt.ID,
			)

			if err := h.promotionHooks.OnSubscriptionPaymentSucceeded(ctx, *pmt.ResourceID, pmt); err != nil {
				h.log.Error("failed to activate subscription", "error", err)
				// Don't fail the webhook - payment already succeeded
				// The subscription can be activated manually or via retry
			}
		} else {
			h.log.Warn("promotion hooks not configured, payment succeeded but subscription not activated",
				"subscription", *pmt.ResourceID,
			)
		}
	}

	return nil
}

// extractAndSaveAuthorization extracts authorization code from Paystack webhook and saves it
func (h *WebhookHandler) extractAndSaveAuthorization(ctx context.Context, rawData json.RawMessage, pmt *paymentdomain.Payment) error {
	// Parse Paystack webhook data
	var webhook struct {
		Event string `json:"event"`
		Data  struct {
			Authorization struct {
				AuthorizationCode string `json:"authorization_code"`
				CardType          string `json:"card_type"`
				Last4             string `json:"last4"`
				ExpMonth          string `json:"exp_month"`
				ExpYear           string `json:"exp_year"`
				Bank              string `json:"bank"`
				Brand             string `json:"brand"`
				Reusable          bool   `json:"reusable"`
			} `json:"authorization"`
			Customer struct {
				ID           int    `json:"id"`
				CustomerCode string `json:"customer_code"`
				Email        string `json:"email"`
			} `json:"customer"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rawData, &webhook); err != nil {
		return fmt.Errorf("failed to parse webhook data: %w", err)
	}

	auth := webhook.Data.Authorization
	customer := webhook.Data.Customer

	// Only save if authorization is reusable and we have the code
	if !auth.Reusable || auth.AuthorizationCode == "" {
		h.log.Info(" authorization not reusable or empty, skipping save")
		return nil
	}

	h.log.Info(" extracting reusable authorization", "code", auth.AuthorizationCode, "last4", auth.Last4, "brand", auth.Brand, "bank", auth.Bank, "customer", customer.CustomerCode)

	// Parse expiry month and year
	expMonth, expYear, err := parseExpiry(auth.ExpMonth, auth.ExpYear)
	if err != nil {
		h.log.Warn("failed to parse expiry dates", "error", err)
		// Continue without expiry - it's optional metadata
	}

	// Create payment method input with full card metadata
	// This provides better UX when displaying saved payment methods to users
	customerCode := customer.CustomerCode
	input := paymentdomain.CreatePaymentMethodInput{
		UserID:            pmt.PayerID,
		AuthorizationCode: auth.AuthorizationCode,
		Provider:          "paystack",
		Currency:          pmt.Currency,
		Last4Digits:       &auth.Last4,
		CardType:          &auth.CardType,
		Brand:             &auth.Brand,
		ExpiryMonth:       expMonth,
		ExpiryYear:        expYear,
		BankName:          &auth.Bank,
		CustomerCode:      &customerCode,
		SetAsDefault:      false, // Don't auto-set as default
	}

	// Save the payment method
	savedPM, err := h.paymentService.SavePaymentMethod(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save payment method: %w", err)
	}

	// Log success with card details for monitoring
	cardDisplay := "unknown card"
	if savedPM.Last4Digits != nil && savedPM.Brand != nil {
		cardDisplay = fmt.Sprintf("%s ending in %s", *savedPM.Brand, *savedPM.Last4Digits)
	}
	h.log.Info(" payment method saved successfully", "id", savedPM.ID, "user", savedPM.UserID, "card", cardDisplay)

	return nil
}

// parseExpiry converts string expiry values to integers
func parseExpiry(monthStr, yearStr string) (*int, *int, error) {
	if monthStr == "" || yearStr == "" {
		return nil, nil, fmt.Errorf("empty expiry values")
	}

	var month, year int
	if _, err := fmt.Sscanf(monthStr, "%d", &month); err != nil {
		return nil, nil, fmt.Errorf("invalid month: %w", err)
	}
	if _, err := fmt.Sscanf(yearStr, "%d", &year); err != nil {
		return nil, nil, fmt.Errorf("invalid year: %w", err)
	}

	// Validate month range
	if month < 1 || month > 12 {
		return nil, nil, fmt.Errorf("month must be between 1 and 12")
	}

	return &month, &year, nil
}

// handleChargeFailed handles failed payment webhook
func (h *WebhookHandler) handleChargeFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Warn("handling charge.failed for ref", "ref", event.Reference)

	// Get the payment by reference
	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		h.log.Error("failed to get payment by reference", "error", err)
		return fmt.Errorf("failed to get payment: %w", err)
	}

	h.log.Warn("payment failed", "id", pmt.ID, "ref", event.Reference)

	// If payment is for a booking, notify booking module
	if pmt.BookingID != nil && h.bookingHooks != nil {
		h.log.Info(" notifying booking module of payment failure", "booking", *pmt.BookingID)

		reason := fmt.Sprintf("Payment charge failed (status: %s)", event.Status)

		if err := h.bookingHooks.OnPaymentFailed(ctx, *pmt.BookingID, pmt, reason); err != nil {
			h.log.Error("failed to notify booking of payment failure", "error", err)
			// Don't fail the webhook - just log the error
		}
	}

	return nil
}

// handleTransferSuccess handles successful payout webhook
func (h *WebhookHandler) handleTransferSuccess(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Info(" handling transfer.success for ref", "ref", event.Reference)

	// The reference in the event is the disbursement ID (set when initiating transfer)
	// Notify the payout service to update disbursement status
	if h.payoutHooks != nil {
		transferCode := event.Reference
		if event.ProviderTxID != "" {
			transferCode = event.ProviderTxID
		}

		h.log.Info(" notifying payout service of transfer success", "ref", transferCode)

		if err := h.payoutHooks.OnTransferSuccess(ctx, transferCode); err != nil {
			h.log.Error("failed to update disbursement status", "error", err)
			// Don't fail webhook - transfer already succeeded
			// Can be updated manually or via reconciliation
			return nil
		}

		h.log.Info(" disbursement marked as completed", "ref", transferCode)
	} else {
		h.log.Warn("payout hooks not configured, transfer success not processed")
	}

	return nil
}

// handleTransferFailed handles failed payout webhook
func (h *WebhookHandler) handleTransferFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Warn("handling transfer.failed for ref", "ref", event.Reference)

	// Notify the payout service to mark disbursement as failed and schedule retry
	if h.payoutHooks != nil {
		transferCode := event.Reference
		if event.ProviderTxID != "" {
			transferCode = event.ProviderTxID
		}

		// Extract failure reason from event
		reason := "Transfer failed"
		if event.Status != "" {
			reason = fmt.Sprintf("Transfer failed: %s", event.Status)
		}

		h.log.Warn("notifying payout service of transfer failure", "ref", transferCode, "reason", reason)

		if err := h.payoutHooks.OnTransferFailed(ctx, transferCode, reason); err != nil {
			h.log.Error("failed to update disbursement failure status", "error", err)
			// Don't fail webhook - we still want to acknowledge receipt
			return nil
		}

		h.log.Info(" disbursement marked as failed with retry scheduled", "ref", transferCode)
	} else {
		h.log.Warn("payout hooks not configured, transfer failure not processed")
	}

	return nil
}

// handleRefundProcessed handles successful refund webhook
func (h *WebhookHandler) handleRefundProcessed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Info(" handling refund.processed for ref", "ref", event.Reference)

	// Get the payment by reference
	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to get payment by reference: %w", err)
	}

	// Verify the refund status with the provider to get updated refund amount
	// This ensures we have the latest state from Paystack
	verifiedPmt, err := h.paymentService.VerifyPayment(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to verify payment after refund: %w", err)
	}

	h.log.Info("refund processed",
		"payment_id", pmt.ID,
		"refunded_amount", verifiedPmt.RefundedAmount,
		"status", verifiedPmt.Status)

	// If refund is for a booking, record in finance ledger FIRST, then notify booking
	if verifiedPmt.BookingID != nil {
		// Finance records refund FIRST (idempotent)
		if h.financeHooks != nil {
			h.log.Info(" recording refund in finance", "booking", *verifiedPmt.BookingID, "amount", verifiedPmt.RefundedAmount)

			if err := h.financeHooks.OnRefundProcessed(ctx, *verifiedPmt.BookingID, verifiedPmt.ID, verifiedPmt.RefundedAmount, string(verifiedPmt.Currency)); err != nil {
				h.log.Error("failed to record refund in finance", "error", err)
				// Continue - don't fail webhook, but alert admin
				// Finance ledger can be corrected manually
			}
		}

		// Then update booking status
		if h.bookingHooks != nil {
			h.log.Info(" notifying booking module of refund", "booking", *verifiedPmt.BookingID)

			if err := h.bookingHooks.OnPaymentRefunded(ctx, *verifiedPmt.BookingID, verifiedPmt); err != nil {
				h.log.Error("failed to notify booking of refund", "error", err)
				// Don't fail the webhook - just log the error
			}
		}
	}

	return nil
}

// handleRefundFailed handles failed refund webhook
func (h *WebhookHandler) handleRefundFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.WarnContext(ctx, "handling refund.failed", "ref", event.Reference)

	// Get the payment by reference
	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to get payment by reference: %w", err)
	}

	// Log the failure with payment details
	h.log.Warn("refund failed",
		"payment_id", pmt.ID,
		"ref", event.Reference,
		"provider_tx", event.ProviderTxID)

	// TODO: Create a failed transaction record and notify business/admin
	// For now, just log the failure
	// In a production system, you might want to:
	// 1. Create a failed transaction record
	// 2. Send an alert to the business owner
	// 3. Queue for manual review

	return nil
}
