package http

import (
	"encoding/json"
	"hauslet/internal/modules/auth/domain"
	"net/http"
)

// VerifyEmailRequest represents the email verification request
type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// Validate validates the verification request
func (req *VerifyEmailRequest) Validate() error {
	if req.Email == "" {
		return &domain.ValidationError{
			Field:   "email",
			Message: "Email is required",
		}
	}
	if req.Code == "" {
		return &domain.ValidationError{
			Field:   "code",
			Message: "Verification code is required",
		}
	}
	if len(req.Code) != 6 {
		return &domain.ValidationError{
			Field:   "code",
			Message: "Verification code must be 6 digits",
		}
	}
	return nil
}

// VerifyEmail handles email verification with OTP
// @Summary Verify email address
// @Description Verify user's email address using OTP code
// @Tags auth
// @Accept json
// @Produce json
// @Param request body VerifyEmailRequest true "Verification details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /auth/verify-email [post]
func (h *HTTPHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode verification request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.log.Warn("Verification validation failed", "error", err)

		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}

		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Verify OTP and activate email identity (service handles all steps)
	if err := h.authService.VerifyAndActivateEmail(r.Context(), req.Email, req.Code); err != nil {
		h.log.Warn("Email verification failed", "email", req.Email, "error", err)
		h.sendError(w, "Invalid or expired verification code", http.StatusBadRequest, "code")
		return
	}

	h.log.Info("Email verified successfully", "email", req.Email)

	// Send success response
	h.sendSuccess(w, map[string]string{
		"message": "Email verified successfully. You can now log in.",
	}, http.StatusOK)
}
