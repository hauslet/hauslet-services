package http

import (
	"encoding/json"
	"hauslet/internal/modules/auth/domain"
	"net/http"
)

// ForgotPasswordRequest payload
var _ = domain.ErrorResponse{}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest payload
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ForgotPassword handles password reset initiation
// @Summary Request password reset
// @Description Send a password reset email to the user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Email address"
// @Success 200 {object} map[string]string
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /auth/forgot-password [post]
func (h *HTTPHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode forgot password request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Email == "" {
		h.sendError(w, "Email is required", http.StatusBadRequest, "email")
		return
	}

	if err := h.authService.RequestPasswordReset(r.Context(), req.Email); err != nil {
		h.log.Error("Failed to process password reset", "email", req.Email, "error", err)
		// Avoid leaking details
	}

	h.sendSuccess(w, map[string]string{
		"message": "If this email is registered, a reset code has been sent.",
	}, http.StatusOK)
}

// ResetPassword handles completing the password reset
// @Summary Reset password
// @Description Reset user password using the token sent via email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /auth/reset-password [post]
func (h *HTTPHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode reset password request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Email == "" {
		h.sendError(w, "Email is required", http.StatusBadRequest, "email")
		return
	}
	if req.Token == "" {
		h.sendError(w, "Token is required", http.StatusBadRequest, "token")
		return
	}
	if req.NewPassword == "" {
		h.sendError(w, "New password is required", http.StatusBadRequest, "new_password")
		return
	}

	if err := h.authService.ResetPassword(r.Context(), req.Email, req.Token, req.NewPassword); err != nil {
		h.log.Warn("Reset password failed", "email", req.Email, "error", err)
		h.sendError(w, "Invalid token or request", http.StatusBadRequest, "")
		return
	}

	h.sendSuccess(w, map[string]string{
		"message": "Password reset successful",
	}, http.StatusOK)
}
