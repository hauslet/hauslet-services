package http

import (
	"encoding/json"
	"hauslet/internal/modules/payments/domain"
	"hauslet/internal/platform/payment"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// CreatePaymentRequest represents the request body for creating a payment
type CreatePaymentRequest struct {
	Amount          int64             `json:"amount"`
	Currency        payment.Currency  `json:"currency"`
	Market          domain.Market     `json:"market"`
	PayerID         uuid.UUID         `json:"payer_id"`
	PayerEmail      string            `json:"payer_email"`
	PayerName       string            `json:"payer_name"`
	BookingID       *uuid.UUID        `json:"booking_id,omitempty"`
	BusinessID      *uuid.UUID        `json:"business_id,omitempty"`
	PaymentMethodID *uuid.UUID        `json:"payment_method_id,omitempty"`
	CallbackURL     string            `json:"callback_url,omitempty"`
	Description     string            `json:"description"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// CreatePayment creates a new payment (admin only)
func (h *AdminHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode create payment request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate required fields
	if req.Amount <= 0 {
		h.sendError(w, "Amount must be greater than zero", http.StatusBadRequest, "amount")
		return
	}

	if req.PayerEmail == "" {
		h.sendError(w, "Payer email is required", http.StatusBadRequest, "payer_email")
		return
	}

	// Convert to domain input
	domainInput := domain.CreatePaymentInput{
		Amount:          req.Amount,
		Currency:        req.Currency,
		Market:          req.Market,
		PayerID:         req.PayerID,
		PayerEmail:      req.PayerEmail,
		PayerName:       req.PayerName,
		Description:     req.Description,
		Metadata:        req.Metadata,
		BookingID:       req.BookingID,
		BusinessID:      req.BusinessID,
		PaymentMethodID: req.PaymentMethodID,
		CallbackURL:     req.CallbackURL,
	}

	// Create payment
	pmt, err := h.paymentService.CreatePayment(r.Context(), domainInput)
	if err != nil {
		h.log.Error("failed to create payment", "error", err)
		h.sendError(w, "Failed to create payment", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, pmt, http.StatusCreated)
}

// VerifyPayment verifies a payment by reference (admin only)
func (h *AdminHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	// Extract reference from URL
	reference := chi.URLParam(r, "reference")
	if reference == "" {
		h.sendError(w, "Payment reference is required", http.StatusBadRequest, "reference")
		return
	}

	// Verify payment
	pmt, err := h.paymentService.VerifyPayment(r.Context(), reference)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			h.sendError(w, "Payment not found", http.StatusNotFound, "")
			return
		}
		h.log.Error("failed to verify payment", "reference", reference, "error", err)
		h.sendError(w, "Failed to verify payment", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, pmt, http.StatusOK)
}

// RefundPaymentRequest represents the request body for refunding a payment
type RefundPaymentRequest struct {
	PaymentID  uuid.UUID `json:"payment_id"`
	Amount     *int64    `json:"amount,omitempty"`
	Reason     string    `json:"reason"`
	RefundedBy uuid.UUID `json:"refunded_by"`
}

// RefundPayment processes a payment refund (admin/support only)
func (h *AdminHandler) RefundPayment(w http.ResponseWriter, r *http.Request) {
	var req RefundPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode refund payment request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate required fields
	if req.Reason == "" {
		h.sendError(w, "Refund reason is required", http.StatusBadRequest, "reason")
		return
	}

	// Convert to domain input
	domainInput := domain.RefundPaymentInput{
		PaymentID:  req.PaymentID,
		Amount:     req.Amount,
		Reason:     req.Reason,
		RefundedBy: req.RefundedBy,
	}

	// Process refund
	pmt, err := h.paymentService.RefundPayment(r.Context(), domainInput)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			h.sendError(w, "Payment not found", http.StatusNotFound, "")
			return
		}
		h.log.Error("failed to refund payment", "payment_id", req.PaymentID, "error", err)
		h.sendError(w, "Failed to process refund", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, pmt, http.StatusOK)
}

// ListAllPayments lists payments with optional filters (admin only)
func (h *AdminHandler) ListAllPayments(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	limitParam := r.URL.Query().Get("limit")
	offsetParam := r.URL.Query().Get("offset")
	payerIDParam := r.URL.Query().Get("payer_id")
	bookingIDParam := r.URL.Query().Get("booking_id")
	statusParam := r.URL.Query().Get("status")

	// Parse pagination
	limit := 50
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = min(l, 100)
		}
	}

	offset := 0
	if offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o > 0 {
			offset = o
		}
	}

	var payments []domain.Payment
	var err error

	// Filter by payer ID if provided
	if payerIDParam != "" {
		payerID, parseErr := uuid.Parse(payerIDParam)
		if parseErr != nil {
			h.sendError(w, "Invalid payer ID", http.StatusBadRequest, "payer_id")
			return
		}

		payments, err = h.paymentService.ListPaymentsByPayer(r.Context(), payerID, limit, offset)
		if err != nil {
			h.log.Error("failed to list payments by payer", "payer_id", payerID, "error", err)
			h.sendError(w, "Failed to list payments", http.StatusInternalServerError, "")
			return
		}
	} else if bookingIDParam != "" {
		// Filter by booking ID if provided
		bookingID, parseErr := uuid.Parse(bookingIDParam)
		if parseErr != nil {
			h.sendError(w, "Invalid booking ID", http.StatusBadRequest, "booking_id")
			return
		}

		payments, err = h.paymentService.ListPaymentsByBooking(r.Context(), bookingID)
		if err != nil {
			h.log.Error("failed to list payments by booking", "booking_id", bookingID, "error", err)
			h.sendError(w, "Failed to list payments", http.StatusInternalServerError, "")
			return
		}
	} else {
		// No specific filter - require at least one filter
		h.sendError(w, "Either payer_id or booking_id must be provided", http.StatusBadRequest, "")
		return
	}

	// Apply status filter if provided
	if statusParam != "" {
		status := domain.PaymentStatus(statusParam)
		filtered := make([]domain.Payment, 0)
		for _, p := range payments {
			if p.Status == status {
				filtered = append(filtered, p)
			}
		}
		payments = filtered
	}

	h.sendSuccess(w, payments, http.StatusOK)
}
