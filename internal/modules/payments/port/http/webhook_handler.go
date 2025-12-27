package http

import (
	"context"
	"encoding/json"
	"fmt"
	paymentdomain "hauslet/internal/modules/payments/domain"
	"hauslet/internal/modules/payments/service"
	"hauslet/internal/platform/payment"
	"io"
	"net/http"

	"github.com/go-pkgz/lgr"
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

// WebhookHandler handles payment provider webhooks
type WebhookHandler struct {
	paymentService service.PaymentService
	paymentClient  *payment.Client
	bookingHooks   BookingHooks
	financeHooks   FinanceHooks
	payoutHooks    PayoutHooks
	log            *lgr.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(
	paymentService service.PaymentService,
	paymentClient *payment.Client,
	bookingHooks BookingHooks,
	financeHooks FinanceHooks,
	payoutHooks PayoutHooks,
	log *lgr.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
		paymentClient:  paymentClient,
		bookingHooks:   bookingHooks,
		financeHooks:   financeHooks,
		payoutHooks:    payoutHooks,
		log:            log,
	}
}

// HandlePaystackWebhook processes Paystack webhook events
func (h *WebhookHandler) HandlePaystackWebhook(w http.ResponseWriter, r *http.Request) {
	h.log.Logf("INFO received Paystack webhook")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Logf("ERROR failed to read webhook body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify webhook signature
	signature := r.Header.Get("X-Paystack-Signature")
	if signature == "" {
		h.log.Logf("WARN webhook received without signature")
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	valid, err := h.paymentClient.VerifyWebhookSignature("paystack", signature, body)
	if err != nil || !valid {
		h.log.Logf("ERROR webhook signature verification failed: %v", err)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse webhook event
	event, err := h.paymentClient.ParseWebhookEvent("paystack", body)
	if err != nil {
		h.log.Logf("ERROR failed to parse webhook event: %v", err)
		http.Error(w, "Invalid webhook data", http.StatusBadRequest)
		return
	}

	h.log.Logf("INFO processing webhook event: type=%s, ref=%s", event.Type, event.Reference)

	// Handle different event types
	ctx := r.Context()
	switch event.Type {
	case "charge.success":
		// Payment succeeded
		if err := h.handleChargeSuccess(ctx, event); err != nil {
			h.log.Logf("ERROR failed to handle charge.success: %v", err)
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "charge.failed":
		// Payment failed
		if err := h.handleChargeFailed(ctx, event); err != nil {
			h.log.Logf("ERROR failed to handle charge.failed: %v", err)
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "transfer.success":
		// Payout succeeded
		if err := h.handleTransferSuccess(ctx, event); err != nil {
			h.log.Logf("ERROR failed to handle transfer.success: %v", err)
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "transfer.failed":
		// Payout failed
		if err := h.handleTransferFailed(ctx, event); err != nil {
			h.log.Logf("ERROR failed to handle transfer.failed: %v", err)
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "refund.processed":
		// Refund processed successfully
		if err := h.handleRefundProcessed(ctx, event); err != nil {
			h.log.Logf("ERROR failed to handle refund.processed: %v", err)
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "refund.failed":
		// Refund failed
		if err := h.handleRefundFailed(ctx, event); err != nil {
			h.log.Logf("ERROR failed to handle refund.failed: %v", err)
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	default:
		h.log.Logf("INFO unhandled webhook event type: %s", event.Type)
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// handleChargeSuccess handles successful payment webhook
func (h *WebhookHandler) handleChargeSuccess(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Logf("INFO handling charge.success for ref=%s", event.Reference)

	// Verify payment status
	pmt, err := h.paymentService.VerifyPayment(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to verify payment: %w", err)
	}

	h.log.Logf("INFO payment verified: id=%s, status=%s", pmt.ID, pmt.Status)

	// Extract and save authorization code from webhook for card tokenization
	if err := h.extractAndSaveAuthorization(ctx, event.RawData, pmt); err != nil {
		h.log.Logf("WARN failed to extract authorization code: %v", err)
		// Don't fail the webhook - payment already succeeded
		// Authorization saving is optional for future one-click payments
	}

	// If payment is for a booking, record in finance ledger FIRST, then notify booking
	if pmt.BookingID != nil {
		// Finance records transaction FIRST (idempotent)
		if h.financeHooks != nil {
			h.log.Logf("INFO recording charge in finance: booking=%s, amount=%d", *pmt.BookingID, pmt.Amount)

			if err := h.financeHooks.OnPaymentSucceeded(ctx, *pmt.BookingID, pmt.ID, pmt.Amount, string(pmt.Currency)); err != nil {
				h.log.Logf("ERROR failed to record charge in finance: %v", err)
				// Continue - don't fail webhook, but alert admin
				// Finance ledger can be corrected manually
			}
		}

		// Then update booking status
		if h.bookingHooks != nil {
			h.log.Logf("INFO notifying booking module of payment success: booking=%s", *pmt.BookingID)

			if err := h.bookingHooks.OnPaymentSucceeded(ctx, *pmt.BookingID, pmt); err != nil {
				h.log.Logf("ERROR failed to notify booking of payment success: %v", err)
				// Don't fail the webhook - payment already succeeded
				// The booking can be confirmed manually or via retry
			}
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
		h.log.Logf("INFO authorization not reusable or empty, skipping save")
		return nil
	}

	h.log.Logf("INFO extracting reusable authorization: code=%s, last4=%s, brand=%s, bank=%s, customer=%s",
		auth.AuthorizationCode, auth.Last4, auth.Brand, auth.Bank, customer.CustomerCode)

	// Parse expiry month and year
	expMonth, expYear, err := parseExpiry(auth.ExpMonth, auth.ExpYear)
	if err != nil {
		h.log.Logf("WARN failed to parse expiry dates: %v", err)
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
	h.log.Logf("INFO payment method saved successfully: id=%s, user=%s, card=%s",
		savedPM.ID, savedPM.UserID, cardDisplay)

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
	h.log.Logf("WARN handling charge.failed for ref=%s", event.Reference)

	// Get the payment by reference
	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		h.log.Logf("ERROR failed to get payment by reference: %v", err)
		return fmt.Errorf("failed to get payment: %w", err)
	}

	h.log.Logf("WARN payment failed: id=%s, ref=%s", pmt.ID, event.Reference)

	// If payment is for a booking, notify booking module
	if pmt.BookingID != nil && h.bookingHooks != nil {
		h.log.Logf("INFO notifying booking module of payment failure: booking=%s", *pmt.BookingID)

		reason := fmt.Sprintf("Payment charge failed (status: %s)", event.Status)

		if err := h.bookingHooks.OnPaymentFailed(ctx, *pmt.BookingID, pmt, reason); err != nil {
			h.log.Logf("ERROR failed to notify booking of payment failure: %v", err)
			// Don't fail the webhook - just log the error
		}
	}

	return nil
}

// handleTransferSuccess handles successful payout webhook
func (h *WebhookHandler) handleTransferSuccess(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Logf("INFO handling transfer.success for ref=%s", event.Reference)

	// The reference in the event is the disbursement ID (set when initiating transfer)
	// Notify the payout service to update disbursement status
	if h.payoutHooks != nil {
		transferCode := event.Reference
		if event.ProviderTxID != "" {
			transferCode = event.ProviderTxID
		}

		h.log.Logf("INFO notifying payout service of transfer success: ref=%s", transferCode)

		if err := h.payoutHooks.OnTransferSuccess(ctx, transferCode); err != nil {
			h.log.Logf("ERROR failed to update disbursement status: %v", err)
			// Don't fail webhook - transfer already succeeded
			// Can be updated manually or via reconciliation
			return nil
		}

		h.log.Logf("INFO disbursement marked as completed: ref=%s", transferCode)
	} else {
		h.log.Logf("WARN payout hooks not configured, transfer success not processed")
	}

	return nil
}

// handleTransferFailed handles failed payout webhook
func (h *WebhookHandler) handleTransferFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Logf("WARN handling transfer.failed for ref=%s", event.Reference)

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

		h.log.Logf("WARN notifying payout service of transfer failure: ref=%s, reason=%s", transferCode, reason)

		if err := h.payoutHooks.OnTransferFailed(ctx, transferCode, reason); err != nil {
			h.log.Logf("ERROR failed to update disbursement failure status: %v", err)
			// Don't fail webhook - we still want to acknowledge receipt
			return nil
		}

		h.log.Logf("INFO disbursement marked as failed with retry scheduled: ref=%s", transferCode)
	} else {
		h.log.Logf("WARN payout hooks not configured, transfer failure not processed")
	}

	return nil
}

// handleRefundProcessed handles successful refund webhook
func (h *WebhookHandler) handleRefundProcessed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Logf("INFO handling refund.processed for ref=%s", event.Reference)

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

	h.log.Logf("INFO refund processed: payment_id=%s, refunded_amount=%d, status=%s",
		pmt.ID, verifiedPmt.RefundedAmount, verifiedPmt.Status)

	// If refund is for a booking, record in finance ledger FIRST, then notify booking
	if verifiedPmt.BookingID != nil {
		// Finance records refund FIRST (idempotent)
		if h.financeHooks != nil {
			h.log.Logf("INFO recording refund in finance: booking=%s, amount=%d", *verifiedPmt.BookingID, verifiedPmt.RefundedAmount)

			if err := h.financeHooks.OnRefundProcessed(ctx, *verifiedPmt.BookingID, verifiedPmt.ID, verifiedPmt.RefundedAmount, string(verifiedPmt.Currency)); err != nil {
				h.log.Logf("ERROR failed to record refund in finance: %v", err)
				// Continue - don't fail webhook, but alert admin
				// Finance ledger can be corrected manually
			}
		}

		// Then update booking status
		if h.bookingHooks != nil {
			h.log.Logf("INFO notifying booking module of refund: booking=%s", *verifiedPmt.BookingID)

			if err := h.bookingHooks.OnPaymentRefunded(ctx, *verifiedPmt.BookingID, verifiedPmt); err != nil {
				h.log.Logf("ERROR failed to notify booking of refund: %v", err)
				// Don't fail the webhook - just log the error
			}
		}
	}

	return nil
}

// handleRefundFailed handles failed refund webhook
func (h *WebhookHandler) handleRefundFailed(ctx context.Context, event *payment.UnifiedEvent) error {
	h.log.Logf("WARN handling refund.failed for ref=%s", event.Reference)

	// Get the payment by reference
	pmt, err := h.paymentService.GetPaymentByReference(ctx, event.Reference)
	if err != nil {
		return fmt.Errorf("failed to get payment by reference: %w", err)
	}

	// Log the failure with payment details
	h.log.Logf("WARN refund failed: payment_id=%s, ref=%s, provider_tx=%s",
		pmt.ID, event.Reference, event.ProviderTxID)

	// TODO: Create a failed transaction record and notify business/admin
	// For now, just log the failure
	// In a production system, you might want to:
	// 1. Create a failed transaction record
	// 2. Send an alert to the business owner
	// 3. Queue for manual review

	return nil
}
