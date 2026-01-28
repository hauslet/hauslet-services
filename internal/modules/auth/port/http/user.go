package http

import (
	"encoding/json"
	"hauslet/internal/modules/auth/domain"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"net/http"

	"github.com/go-pkgz/auth/v2/token"
)

// GetCurrentUser returns the authenticated user's profile
// @Summary Get current user profile
// @Description Returns the authenticated user's information
// @Tags user
// @Produce json
// @Success 200 {object} domain.UserResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} map[string]interface{} "2FA Required"
// @Failure 500 {object} domain.ErrorResponse
// @Router /me [get]
// @Security BearerAuth
func (h *HTTPHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Check for 2FA pending state which allows partial access (but blocks profile access)
	if user, err := token.GetUserInfo(r); err == nil {
		if user.StrAttr("login_state") == "2fa_pending" {
			// Return 403 Forbidden with details needed for frontend to show 2FA input
			response := map[string]any{
				"requires_2fa": true,
				"temp_token":   user.StrAttr("2fa_temp_token"),
				"method":       user.StrAttr("2fa_method"),
				"messsage":     "Two-factor authentication required",
			}
			h.log.Info("Blocking access to /me - 2FA required", "user_id", user.ID)
			h.sendSuccess(w, response, http.StatusForbidden)
			return
		}
	}

	// Extract user from JWT claims (set by auth middleware)
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Get user from database
	user, err := h.authService.GetUser(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get user", "user_id", userID, "error", err)
		h.sendError(w, "Failed to retrieve user profile", http.StatusInternalServerError, "")
		return
	}

	if user == nil {
		h.log.Warn("User not found", "user_id", userID)
		h.sendError(w, "User not found", http.StatusNotFound, "")
		return
	}

	// Return user profile
	response := domain.UserResponse{
		ID:        user.ID.String(),
		Email:     user.PrimaryEmail,
		Name:      user.Name,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// UpdateCurrentUser updates the authenticated user's profile
// @Summary Update current user profile
// @Description Update user's name or email
// @Tags user
// @Accept json
// @Produce json
// @Param request body domain.UpdateUserRequest true "Update details"
// @Success 200 {object} domain.UserResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /me [put]
// @Security BearerAuth
func (h *HTTPHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT claims
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}
	// Decode request body
	var req domain.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode update request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request using domain validation
	if err := req.Validate(); err != nil {
		h.log.Warn("Update validation failed", "error", err)

		// Check if it's a validation error with a field
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}

		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Get current user
	user, err := h.authService.GetUser(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get user", "user_id", userID, "error", err)
		h.sendError(w, "Failed to retrieve user", http.StatusInternalServerError, "")
		return
	}

	if user == nil {
		h.sendError(w, "User not found", http.StatusNotFound, "")
		return
	}

	// Update fields (only if provided)
	updated := false

	if req.Name != "" && req.Name != user.Name {
		user.Name = req.Name
		updated = true
	}

	if req.Email != "" && req.Email != user.PrimaryEmail {
		user.PrimaryEmail = req.Email
		updated = true
	}

	if !updated {
		h.sendError(w, "No changes provided", http.StatusBadRequest, "")
		return
	}

	// Save updated user
	if err := h.authService.UpdateUser(r.Context(), user); err != nil {
		h.log.Error("Failed to update user %s: %v", userID, err)
		h.sendError(w, "Failed to update user profile", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("User profile updated", "user_id", userID)

	// Return updated user
	response := domain.UserResponse{
		ID:        user.ID.String(),
		Email:     user.PrimaryEmail,
		Name:      user.Name,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// ChangePassword allows users to change their password
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags user
// @Accept json
// @Produce json
// @Param request body domain.ChangePasswordRequest true "Password change details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /change-password [post]
// @Security BearerAuth
func (h *HTTPHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT claims
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Decode request body
	var req domain.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode password change request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request using domain validation
	if err := req.Validate(); err != nil {
		h.log.Warn("Password change validation failed", "error", err)

		// Check if it's a validation error with a field
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}

		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Change password
	if err := h.authService.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		h.log.Error("Failed to change password for user", "user_id", userID, "error", err)

		// Check for specific errors
		switch err.Error() {
		case "invalid password":
			h.sendError(w, "Current password is incorrect", http.StatusBadRequest, "old_password")
			return
		case "password identity not found":
			h.sendError(w, "Password authentication not set up for this account", http.StatusBadRequest, "")
			return
		default:
			h.sendError(w, "Failed to change password", http.StatusInternalServerError, "")
			return
		}
	}

	h.log.Info("Password changed for user", "user_id", userID)

	// Return success
	response := map[string]string{
		"message": "Password changed successfully",
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// GetUserIdentities lists all authentication methods for the current user
// @Summary Get user identities
// @Description List all authentication methods (OAuth providers and password) linked to the user
// @Tags user
// @Produce json
// @Success 200 {object} []domain.UserIdentityResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /me/identities [get]
// @Security BearerAuth
func (h *HTTPHandler) GetUserIdentities(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT claims
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Get identities
	identities, err := h.authService.ListUserIdentities(r.Context(), userID)
	if err != nil {
		h.log.Error("Failed to get identities for user", "user_id", userID, "error", err)
		h.sendError(w, "Failed to retrieve identities", http.StatusInternalServerError, "")
		return
	}

	// Convert to response format
	response := make([]domain.UserIdentityResponse, len(identities))
	for i, identity := range identities {
		response[i] = domain.UserIdentityResponse{
			ID:            identity.ID.String(),
			Provider:      identity.Provider,
			Email:         identity.Email,
			EmailVerified: identity.EmailVerified,
		}
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// UnlinkIdentity removes an authentication method from the user's account
// @Summary Unlink identity
// @Description Remove an OAuth provider or password authentication from user account
// @Tags user
// @Param id path string true "Identity ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /me/identities/{id} [delete]
// @Security BearerAuth
func (h *HTTPHandler) UnlinkIdentity(w http.ResponseWriter, r *http.Request) {
	// Extract user from JWT claims
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.log.Error("failed to get user info")
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	// Get identity ID from URL
	identityID := r.URL.Query().Get("id")
	if identityID == "" {
		h.sendError(w, "Identity ID is required", http.StatusBadRequest, "")
		return
	}

	// Unlink identity
	if err := h.authService.UnlinkIdentity(r.Context(), identityID); err != nil {
		h.log.Error("Failed to unlink identity", "identity_id", identityID, "error", err)

		if err.Error() == "cannot unlink the last identity: user must have at least one login method" {
			h.sendError(w, "Cannot remove last authentication method", http.StatusBadRequest, "")
			return
		}

		h.sendError(w, "Failed to unlink identity", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("Identity unlinked", "identity_id", identityID)

	response := map[string]string{
		"message": "Identity unlinked successfully",
	}

	h.sendSuccess(w, response, http.StatusOK)
}
