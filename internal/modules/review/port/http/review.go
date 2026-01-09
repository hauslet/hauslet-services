package http

import (
	"encoding/json"
	"fmt"
	"hauslet/internal/modules/review/domain"

	authmiddleware "hauslet/internal/modules/auth/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PublishReview manually publishes a standoff review (admin only)
func (h *AdminHandler) PublishReview(w http.ResponseWriter, r *http.Request) {
	// Extract admin ID from context
	adminIDStr := authmiddleware.GetUserID(r)
	if adminIDStr == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		h.log.Error("invalid admin ID", "admin_id", adminIDStr, "error", err)
		h.sendError(w, "Invalid admin ID", http.StatusBadRequest, "")
		return
	}

	// Extract review ID from URL
	reviewIDStr := chi.URLParam(r, "id")
	if reviewIDStr == "" {
		h.sendError(w, "Review ID is required", http.StatusBadRequest, "id")
		return
	}

	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		h.log.Error("invalid review ID", "review_id", reviewIDStr, "error", err)
		h.sendError(w, "Invalid review ID", http.StatusBadRequest, "id")
		return
	}

	// Publish the review
	if err := h.reviewService.PublishReview(r.Context(), reviewID, adminID); err != nil {
		h.log.Error("failed to publish review", "review_id", reviewID, "admin_id", adminID, "error", err)
		h.sendError(w, fmt.Sprintf("Failed to publish review: %v", err), http.StatusInternalServerError, "")
		return
	}

	// Get updated review to return
	review, err := h.reviewService.GetReview(r.Context(), reviewID, adminID)
	if err != nil {
		h.log.Error("failed to get review after publication", "review_id", reviewID, "error", err)
		h.sendError(w, "Failed to retrieve published review", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("review published", "review_id", reviewID, "admin_id", adminID)
	h.sendSuccess(w, review, http.StatusOK)
}

// HideReviewRequest represents the request body for hiding a review
type HideReviewRequest struct {
	Reason string `json:"reason"`
}

// HideReview hides a review for moderation (admin only)
func (h *AdminHandler) HideReview(w http.ResponseWriter, r *http.Request) {
	// Extract admin ID from context
	adminIDStr := authmiddleware.GetUserID(r)
	if adminIDStr == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		h.log.Error("invalid admin ID", "admin_id", adminIDStr, "error", err)
		h.sendError(w, "Invalid admin ID", http.StatusBadRequest, "")
		return
	}

	// Extract review ID from URL
	reviewIDStr := chi.URLParam(r, "id")
	if reviewIDStr == "" {
		h.sendError(w, "Review ID is required", http.StatusBadRequest, "id")
		return
	}

	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		h.log.Error("invalid review ID", "review_id", reviewIDStr, "error", err)
		h.sendError(w, "Invalid review ID", http.StatusBadRequest, "id")
		return
	}

	// Parse request body
	var req HideReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode request body", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate reason
	if req.Reason == "" {
		h.sendError(w, "Moderation reason is required", http.StatusBadRequest, "reason")
		return
	}

	reason := domain.ModerationReason(req.Reason)

	// Validate reason value
	validReasons := map[domain.ModerationReason]bool{
		domain.ModerationReasonSpam:            true,
		domain.ModerationReasonOffensive:       true,
		domain.ModerationReasonFraudulent:      true,
		domain.ModerationReasonIrrelevant:      true,
		domain.ModerationReasonPersonalInfo:    true,
		domain.ModerationReasonDuplicateReview: true,
		domain.ModerationReasonOther:           true,
	}

	if !validReasons[reason] {
		h.sendError(w, "Invalid moderation reason", http.StatusBadRequest, "reason")
		return
	}

	// Hide the review
	if err := h.reviewService.HideReview(r.Context(), reviewID, adminID, reason); err != nil {
		h.log.Error("failed to hide review", "review_id", reviewID, "admin_id", adminID, "reason", reason, "error", err)
		h.sendError(w, fmt.Sprintf("Failed to hide review: %v", err), http.StatusInternalServerError, "")
		return
	}

	// Get updated review to return
	review, err := h.reviewService.GetReview(r.Context(), reviewID, adminID)
	if err != nil {
		h.log.Error("failed to get review after hiding", "review_id", reviewID, "error", err)
		h.sendError(w, "Failed to retrieve hidden review", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("review hidden", "review_id", reviewID, "admin_id", adminID, "reason", reason)
	h.sendSuccess(w, review, http.StatusOK)
}

// UnhideReview unhides a review (admin only)
func (h *AdminHandler) UnhideReview(w http.ResponseWriter, r *http.Request) {
	// Extract admin ID from context
	adminIDStr := authmiddleware.GetUserID(r)
	if adminIDStr == "" {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		h.log.Error("invalid admin ID", "admin_id", adminIDStr, "error", err)
		h.sendError(w, "Invalid admin ID", http.StatusBadRequest, "")
		return
	}

	// Extract review ID from URL
	reviewIDStr := chi.URLParam(r, "id")
	if reviewIDStr == "" {
		h.sendError(w, "Review ID is required", http.StatusBadRequest, "id")
		return
	}

	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		h.log.Error("invalid review ID", "review_id", reviewIDStr, "error", err)
		h.sendError(w, "Invalid review ID", http.StatusBadRequest, "id")
		return
	}

	// Unhide the review
	if err := h.reviewService.UnhideReview(r.Context(), reviewID, adminID); err != nil {
		h.log.Error("failed to unhide review", "review_id", reviewID, "admin_id", adminID, "error", err)
		h.sendError(w, fmt.Sprintf("Failed to unhide review: %v", err), http.StatusInternalServerError, "")
		return
	}

	// Get updated review to return
	review, err := h.reviewService.GetReview(r.Context(), reviewID, adminID)
	if err != nil {
		h.log.Error("failed to get review after unhiding", "review_id", reviewID, "error", err)
		h.sendError(w, "Failed to retrieve unhidden review", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("review unhidden", "review_id", reviewID, "admin_id", adminID)
	h.sendSuccess(w, review, http.StatusOK)
}
