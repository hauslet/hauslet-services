package http

import (
	"encoding/json"
	"hauslet/internal/modules/auth/domain"
	"net/http"
)

// ResendOTPRequest represents the resend OTP request
type ResendOTPRequest struct {
	Email string `json:"email"`
}

// Validate validates the resend OTP request
func (req *ResendOTPRequest) Validate() error {
	if req.Email == "" {
		return &domain.ValidationError{
			Field:   "email",
			Message: "Email is required",
		}
	}
	return nil
}

// ResendOTP handles resending OTP verification code
// @Summary Resend OTP code
// @Description Resend verification code to user's email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ResendOTPRequest true "Email address"
// @Success 200 {object} map[string]string
// @Failure 400 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /auth/resend-otp [post]
func (h *HTTPHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req ResendOTPRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode resend OTP request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.log.Warn("resend OTP validation failed", "error", err)

		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}

		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Get user by email
	user, err := h.authService.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		h.log.Warn("user not found for email", "email", req.Email, "error", err)
		// Don't reveal if user exists or not for security
		h.sendSuccess(w, map[string]string{
			"message": "If this email is registered and not verified, a new verification code has been sent.",
		}, http.StatusOK)
		return
	}

	// Check if user's email is already verified
	identities, err := h.authService.ListUserIdentities(r.Context(), user.ID.String())
	if err != nil {
		h.log.Error("failed to list user identities", "error", err)
		h.sendError(w, "Failed to resend verification code", http.StatusInternalServerError, "")
		return
	}

	// Check if email is already verified
	for _, identity := range identities {
		if identity.Provider == "password" && identity.Email == req.Email && identity.EmailVerified {
			h.log.Info("email already verified", "email", req.Email)
			h.sendError(w, "Email is already verified", http.StatusBadRequest, "email")
			return
		}
	}

	// Generate new OTP (this will overwrite the old one in Redis)
	otpCode, err := h.authService.GenerateEmailOTP(r.Context(), req.Email)
	if err != nil {
		h.log.Error("failed to generate OTP", "email", req.Email, "error", err)
		h.sendError(w, "Failed to generate verification code", http.StatusInternalServerError, "")
		return
	}

	// Send welcome email with new OTP
	if err := h.authService.SendWelcomeEmail(r.Context(), req.Email, user.Name, otpCode); err != nil {
		h.log.Error("failed to send OTP email", "email", req.Email, "error", err)
		h.sendError(w, "Failed to send verification email", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("resent OTP", "email", req.Email)

	// Send success response
	h.sendSuccess(w, map[string]string{
		"message": "Verification code has been resent to your email.",
	}, http.StatusOK)
}
