package port

import (
	"encoding/json"
	"net/http"
)

// ForgotPasswordRequest payload
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
func (h *HTTPHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Logf("ERROR Failed to decode forgot password request: %v", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Email == "" {
		h.sendError(w, "Email is required", http.StatusBadRequest, "email")
		return
	}

	if err := h.authService.RequestPasswordReset(r.Context(), req.Email); err != nil {
		h.log.Logf("ERROR Failed to process password reset for %s: %v", req.Email, err)
		// Avoid leaking details
	}

	h.sendSuccess(w, map[string]string{
		"message": "If this email is registered, a reset code has been sent.",
	}, http.StatusOK)
}

// ResetPassword handles completing the password reset
func (h *HTTPHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Logf("ERROR Failed to decode reset password request: %v", err)
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
		h.log.Logf("WARN Reset password failed for %s: %v", req.Email, err)
		h.sendError(w, "Invalid token or request", http.StatusBadRequest, "")
		return
	}

	h.sendSuccess(w, map[string]string{
		"message": "Password reset successful",
	}, http.StatusOK)
}
