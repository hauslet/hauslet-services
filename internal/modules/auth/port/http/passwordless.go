package http

import (
	"encoding/json"
	"hauslet/internal/modules/auth/domain"
	"net/http"

	"github.com/go-pkgz/auth/v2/token"
)

// PasswordlessLoginRequest represents the passwordless login verification request.
type PasswordlessLoginRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// Validate validates the passwordless login request.
func (req *PasswordlessLoginRequest) Validate() error {
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

// VerifyPasswordlessCode handles passwordless login via 6-digit code.
// This is the alternative to the magic link flow.
// @Summary Verify passwordless login code
// @Description Verify the 6-digit code sent to email and issue a JWT session
// @Tags auth
// @Accept json
// @Produce json
// @Param request body PasswordlessLoginRequest true "Email and verification code"
// @Success 200 {object} map[string]any
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /auth/passwordless/verify [post]
func (h *HTTPHandler) VerifyPasswordlessCode(w http.ResponseWriter, r *http.Request) {
	var req PasswordlessLoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("Failed to decode passwordless login request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	if err := req.Validate(); err != nil {
		h.log.Warn("Passwordless login validation failed", "error", err)
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Verify OTP code
	if err := h.authService.VerifyPasswordlessOTP(r.Context(), req.Email, req.Code); err != nil {
		h.log.Warn("Passwordless OTP verification failed", "email", req.Email, "error", err)
		h.sendError(w, "Invalid or expired verification code", http.StatusUnauthorized, "code")
		return
	}

	// Look up user by email
	user, err := h.authService.GetUserByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		h.log.Error("User not found for passwordless login", "email", req.Email, "error", err)
		h.sendError(w, "User not found", http.StatusUnauthorized, "")
		return
	}

	if !user.IsActive {
		h.log.Warn("Inactive user attempted passwordless login", "email", req.Email, "user_id", user.ID)
		h.sendError(w, "Account is deactivated", http.StatusUnauthorized, "")
		return
	}

	// Delete OTP after successful verification
	if err := h.authService.DeletePasswordlessOTP(r.Context(), req.Email); err != nil {
		h.log.Warn("Failed to delete passwordless OTP", "email", req.Email, "error", err)
		// Continue - OTP will expire anyway
	}

	// Build claims for the JWT
	claims := token.Claims{
		User: &token.User{
			ID:    "email_" + user.ID.String(),
			Name:  user.Name,
			Email: user.PrimaryEmail,
		},
	}
	claims.User.SetStrAttr("uid", user.ID.String())
	claims.User.SetStrAttr("email", user.PrimaryEmail)
	claims.User.SetStrAttr("role", string(user.Role))
	claims.User.SetStrAttr("provider", "email")

	// Issue JWT token and set cookie
	tokenService := h.authService.OAuthService().TokenService()
	if _, err := tokenService.Set(w, claims); err != nil {
		h.log.Error("Failed to issue JWT for passwordless login", "user_id", user.ID, "error", err)
		h.sendError(w, "Failed to complete login", http.StatusInternalServerError, "")
		return
	}

	h.log.Info("Passwordless login successful", "email", req.Email, "user_id", user.ID)

	h.sendSuccess(w, map[string]any{
		"message": "Login successful",
		"user": map[string]any{
			"id":    user.ID,
			"email": user.PrimaryEmail,
			"name":  user.Name,
			"role":  user.Role,
		},
	}, http.StatusOK)
}
