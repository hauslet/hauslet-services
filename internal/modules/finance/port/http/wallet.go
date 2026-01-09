package http

import (
	"hauslet/internal/modules/finance/domain"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetWallet retrieves a wallet by ID
func (h *HTTPHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	// Extract wallet ID from URL
	walletIDStr := chi.URLParam(r, "id")
	if walletIDStr == "" {
		h.sendError(w, "Wallet ID is required", http.StatusBadRequest, "id")
		return
	}

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		h.log.Error("invalid wallet ID", "wallet_id", walletIDStr, "error", err)
		h.sendError(w, "Invalid wallet ID", http.StatusBadRequest, "id")
		return
	}

	// Get wallet from service
	wallet, err := h.financeService.GetWallet(r.Context(), walletID)
	if err != nil {
		if err == domain.ErrWalletNotFound {
			h.sendError(w, "Wallet not found", http.StatusNotFound, "")
			return
		}
		h.log.Error("failed to get wallet", "wallet_id", walletID, "error", err)
		h.sendError(w, "Failed to retrieve wallet", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, wallet, http.StatusOK)
}

// GetUserWallets lists all wallets for a user
func (h *HTTPHandler) GetUserWallets(w http.ResponseWriter, r *http.Request) {
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

	// Get wallets from service
	wallets, err := h.financeService.ListUserWallets(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to list wallets for user", "user_id", userID, "error", err)
		h.sendError(w, "Failed to retrieve user wallets", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, wallets, http.StatusOK)
}

// GetWalletLedger retrieves ledger entries for a wallet
func (h *HTTPHandler) GetWalletLedger(w http.ResponseWriter, r *http.Request) {
	// Extract wallet ID from URL
	walletIDStr := chi.URLParam(r, "id")
	if walletIDStr == "" {
		h.sendError(w, "Wallet ID is required", http.StatusBadRequest, "id")
		return
	}

	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		h.log.Error("invalid wallet ID", "wallet_id", walletIDStr, "error", err)
		h.sendError(w, "Invalid wallet ID", http.StatusBadRequest, "id")
		return
	}

	// Extract query parameters
	limitParam := r.URL.Query().Get("limit")
	offsetParam := r.URL.Query().Get("offset")

	// Parse pagination
	limit := 20
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

	// Get ledger entries from service
	entries, err := h.financeService.GetWalletHistory(r.Context(), walletID, limit, offset)
	if err != nil {
		h.log.Error("failed to get wallet ledger", "wallet_id", walletID, "error", err)
		h.sendError(w, "Failed to retrieve wallet ledger", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, entries, http.StatusOK)
}

// GetDisbursement retrieves a disbursement by ID
func (h *HTTPHandler) GetDisbursement(w http.ResponseWriter, r *http.Request) {
	// Extract disbursement ID from URL
	disbursementIDStr := chi.URLParam(r, "id")
	if disbursementIDStr == "" {
		h.sendError(w, "Disbursement ID is required", http.StatusBadRequest, "id")
		return
	}

	disbursementID, err := uuid.Parse(disbursementIDStr)
	if err != nil {
		h.log.Error("invalid disbursement ID", "disbursement_id", disbursementIDStr, "error", err)
		h.sendError(w, "Invalid disbursement ID", http.StatusBadRequest, "id")
		return
	}

	// Get disbursement from service
	disbursement, err := h.payoutService.GetDisbursement(r.Context(), disbursementID)
	if err != nil {
		if err == domain.ErrDisbursementNotFound {
			h.sendError(w, "Disbursement not found", http.StatusNotFound, "")
			return
		}
		h.log.Error("failed to get disbursement", "disbursement_id", disbursementID, "error", err)
		h.sendError(w, "Failed to retrieve disbursement", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, disbursement, http.StatusOK)
}