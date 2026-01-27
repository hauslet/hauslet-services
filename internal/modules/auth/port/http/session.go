package http

import (
	"hauslet/internal/modules/auth/domain"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"net/http"
)

// GetUserSessions returns all active sessions for the authenticated user
// @Summary Get user sessions
// @Description List all active sessions for the authenticated user
// @Tags session
// @Produce json
// @Success 200 {object} []domain.SessionResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /me/sessions [get]
// @Security BearerAuth
func (h *HTTPHandler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT claims
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Get sessions
	sessions, err := h.authService.GetUserSessions(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get sessions for user", "user_id", userID, "error", err)
		h.sendError(w, "Failed to retrieve sessions", http.StatusInternalServerError, "")
		return
	}

	// Convert to response format
	response := make([]domain.SessionResponse, len(sessions))
	for i, session := range sessions {
		response[i] = domain.SessionResponse{
			ID:        session.ID,
			UserAgent: session.UserAgent,
			IP:        session.IP,
			Provider:  session.Provider,
			CreatedAt: session.CreatedAt,
			ExpiresAt: session.ExpiresAt,
		}
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// RevokeSession revokes a specific session
// @Summary Revoke session
// @Description Revoke a specific session by ID
// @Tags session
// @Param id path string true "Session ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /me/sessions/{id} [delete]
// @Security BearerAuth
func (h *HTTPHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT claims
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}
	// Get session ID from URL
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		h.sendError(w, "Session ID is required", http.StatusBadRequest, "")
		return
	}

	// Revoke session
	if err := h.authService.RevokeSession(r.Context(), sessionID); err != nil {
		h.log.Error("failed to revoke session", "session_id", sessionID, "error", err)
		h.sendError(w, "Failed to revoke session", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("session revoked", "session_id", sessionID)

	response := map[string]string{
		"message": "Session revoked successfully",
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// RevokeAllSessions revokes all sessions except the current one
// @Summary Revoke all sessions
// @Description Revoke all sessions for the authenticated user (useful for "logout everywhere")
// @Tags session
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /me/sessions [delete]
// @Security BearerAuth
func (h *HTTPHandler) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT claims
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Revoke all sessions
	if err := h.authService.RevokeAllUserSessions(r.Context(), userID); err != nil {
		h.log.Error("failed to revoke all sessions for user", "user_id", userID, "error", err)
		h.sendError(w, "Failed to revoke sessions", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("all sessions revoked", "user_id", userID)

	response := map[string]string{
		"message": "All sessions revoked successfully",
	}

	h.sendSuccess(w, response, http.StatusOK)
}
