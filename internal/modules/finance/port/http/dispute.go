package http

import (
	"encoding/json"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/finance/domain"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ListDisputes lists all disputes with optional status filter
func (h *HTTPHandler) ListDisputes(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	statusParam := r.URL.Query().Get("status")
	limitParam := r.URL.Query().Get("limit")
	offsetParam := r.URL.Query().Get("offset")

	// Parse status filter
	var status *domain.DisputeStatus
	if statusParam != "" {
		s := domain.DisputeStatus(statusParam)
		status = &s
	}

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

	// Get disputes from service
	disputes, err := h.financeService.ListDisputes(r.Context(), status, limit, offset)
	if err != nil {
		h.log.Error("failed to list disputes", "error", err)
		h.sendError(w, "Failed to retrieve disputes", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, disputes, http.StatusOK)
}

// InvestigateDispute marks a dispute as under investigation
func (h *HTTPHandler) InvestigateDispute(w http.ResponseWriter, r *http.Request) {
	// Extract dispute ID from URL
	disputeIDStr := chi.URLParam(r, "id")
	if disputeIDStr == "" {
		h.sendError(w, "Dispute ID is required", http.StatusBadRequest, "id")
		return
	}

	disputeID, err := uuid.Parse(disputeIDStr)
	if err != nil {
		h.log.Error("invalid dispute ID", "dispute_id", disputeIDStr, "error", err)
		h.sendError(w, "Invalid dispute ID", http.StatusBadRequest, "id")
		return
	}

	// Get admin user ID from token
	adminID := authmiddleware.GetUserID(r)
	if adminID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		h.log.Error("invalid admin ID", "admin_id", adminID, "error", err)
		h.sendError(w, "Invalid user ID", http.StatusInternalServerError, "")
		return
	}

	// Mark dispute as investigating
	if err := h.financeService.InvestigateDispute(r.Context(), disputeID, adminUUID); err != nil {
		h.log.Error("failed to investigate dispute", "dispute_id", disputeID, "error", err)
		h.sendError(w, "Failed to mark dispute as investigating", http.StatusInternalServerError, "")
		return
	}

	// Get updated dispute
	dispute, err := h.financeService.GetDispute(r.Context(), disputeID)
	if err != nil {
		h.log.Error("failed to get dispute after investigation", "dispute_id", disputeID, "error", err)
		h.sendError(w, "Failed to retrieve updated dispute", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("dispute marked as investigating", "dispute_id", disputeID, "admin_id", adminUUID)
	h.sendSuccess(w, dispute, http.StatusOK)
}

// ResolveDisputeRequest represents the request body for resolving a dispute
type ResolveDisputeRequest struct {
	Outcome      domain.DisputeStatus `json:"outcome"`
	RefundAmount int64                `json:"refund_amount"`
	Reason       string               `json:"reason"`
	Notes        string               `json:"notes"`
}

// Validate validates the resolve dispute request
func (r *ResolveDisputeRequest) Validate() error {
	if r.Outcome != domain.DisputeStatusResolvedRefund && r.Outcome != domain.DisputeStatusResolvedRelease {
		return domain.ErrInvalidDisputeResolution
	}

	if r.Reason == "" {
		return &ValidationError{Field: "reason", Message: "reason is required"}
	}

	if r.Outcome == domain.DisputeStatusResolvedRefund && r.RefundAmount <= 0 {
		return &ValidationError{Field: "refund_amount", Message: "refund amount must be greater than zero for refund outcome"}
	}

	return nil
}

// ValidationError represents a field-specific validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ResolveDispute resolves a dispute
func (h *HTTPHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	// Extract dispute ID from URL
	disputeIDStr := chi.URLParam(r, "id")
	if disputeIDStr == "" {
		h.sendError(w, "Dispute ID is required", http.StatusBadRequest, "id")
		return
	}

	disputeID, err := uuid.Parse(disputeIDStr)
	if err != nil {
		h.log.Error("invalid dispute ID", "dispute_id", disputeIDStr, "error", err)
		h.sendError(w, "Invalid dispute ID", http.StatusBadRequest, "id")
		return
	}

	// Get admin user ID from token
	adminID := authmiddleware.GetUserID(r)
	if adminID == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		h.log.Error("invalid admin ID", "admin_id", adminID, "error", err)
		h.sendError(w, "Invalid user ID", http.StatusInternalServerError, "")
		return
	}

	// Decode request body
	var req ResolveDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode resolve dispute request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.log.Warn("resolve dispute validation failed", "error", err)
		if valErr, ok := err.(*ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Resolve dispute
	if err := h.financeService.ResolveDispute(
		r.Context(),
		disputeID,
		adminUUID,
		req.Outcome,
		req.RefundAmount,
		req.Reason,
		req.Notes,
	); err != nil {
		h.log.Error("failed to resolve dispute", "dispute_id", disputeID, "error", err)
		h.sendError(w, "Failed to resolve dispute", http.StatusInternalServerError, "")
		return
	}

	// Get updated dispute
	dispute, err := h.financeService.GetDispute(r.Context(), disputeID)
	if err != nil {
		h.log.Error("failed to get dispute after resolution", "dispute_id", disputeID, "error", err)
		h.sendError(w, "Failed to retrieve updated dispute", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("dispute resolved", "dispute_id", disputeID, "outcome", req.Outcome, "admin_id", adminUUID)
	h.sendSuccess(w, dispute, http.StatusOK)
}