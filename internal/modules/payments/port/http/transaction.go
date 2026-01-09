package http

import (
	"hauslet/internal/modules/payments/domain"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetBookingTransactions retrieves all transactions for a booking (admin only)
func (h *AdminHandler) GetBookingTransactions(w http.ResponseWriter, r *http.Request) {
	// Extract booking ID from URL
	bookingIDStr := chi.URLParam(r, "bookingID")
	if bookingIDStr == "" {
		h.sendError(w, "Booking ID is required", http.StatusBadRequest, "bookingID")
		return
	}

	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		h.log.Error("invalid booking ID", "booking_id", bookingIDStr, "error", err)
		h.sendError(w, "Invalid booking ID", http.StatusBadRequest, "bookingID")
		return
	}

	// Get transactions for booking
	transactions, err := h.paymentService.ListTransactionsByBooking(r.Context(), bookingID)
	if err != nil {
		h.log.Error("failed to list transactions for booking", "booking_id", bookingID, "error", err)
		h.sendError(w, "Failed to retrieve transactions", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, transactions, http.StatusOK)
}

// ListAllTransactions lists transactions with optional filters (admin only)
func (h *AdminHandler) ListAllTransactions(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	paymentIDParam := r.URL.Query().Get("payment_id")
	txTypeParam := r.URL.Query().Get("type")
	statusParam := r.URL.Query().Get("status")
	limitParam := r.URL.Query().Get("limit")
	offsetParam := r.URL.Query().Get("offset")

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

	var transactions []domain.Transaction
	var err error

	// Filter by payment ID if provided
	if paymentIDParam != "" {
		paymentID, parseErr := uuid.Parse(paymentIDParam)
		if parseErr != nil {
			h.sendError(w, "Invalid payment ID", http.StatusBadRequest, "payment_id")
			return
		}

		transactions, err = h.paymentService.ListTransactionsByPayment(r.Context(), paymentID)
		if err != nil {
			h.log.Error("failed to list transactions by payment", "payment_id", paymentID, "error", err)
			h.sendError(w, "Failed to list transactions", http.StatusInternalServerError, "")
			return
		}
	} else {
		// No payment ID filter - require it for now
		h.sendError(w, "payment_id parameter is required", http.StatusBadRequest, "")
		return
	}

	// Apply type filter if provided
	if txTypeParam != "" {
		txType := domain.TransactionType(txTypeParam)
		filtered := make([]domain.Transaction, 0)
		for _, tx := range transactions {
			if tx.Type == txType {
				filtered = append(filtered, tx)
			}
		}
		transactions = filtered
	}

	// Apply status filter if provided
	if statusParam != "" {
		status := domain.TransactionStatus(statusParam)
		filtered := make([]domain.Transaction, 0)
		for _, tx := range transactions {
			if tx.Status == status {
				filtered = append(filtered, tx)
			}
		}
		transactions = filtered
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start >= len(transactions) {
		h.sendSuccess(w, []domain.Transaction{}, http.StatusOK)
		return
	}
	if end > len(transactions) {
		end = len(transactions)
	}

	h.sendSuccess(w, transactions[start:end], http.StatusOK)
}
