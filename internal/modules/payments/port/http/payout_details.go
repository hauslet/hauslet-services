package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetUserPayoutDetails retrieves payout details for a specific user (admin only)
func (h *AdminHandler) GetUserPayoutDetails(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL
	userIDStr := chi.URLParam(r, "userId")
	if userIDStr == "" {
		h.sendError(w, "User ID is required", http.StatusBadRequest, "userId")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.log.Error("invalid user ID", "user_id", userIDStr, "error", err)
		h.sendError(w, "Invalid user ID", http.StatusBadRequest, "userId")
		return
	}

	// Get payout details for user
	details, err := h.paymentService.ListPayoutDetailsByUserID(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to list payout details for user", "user_id", userID, "error", err)
		h.sendError(w, "Failed to retrieve payout details", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, details, http.StatusOK)
}
