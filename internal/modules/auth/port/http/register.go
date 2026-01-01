package http

import (
	"encoding/json"
	"hauslet/internal/modules/auth/domain"
	"net/http"
)

// Register handles user registration with email and password
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "Registration details"
// @Success 201 {object} domain.RegisterResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /register [post]
func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode registration request", "error", err)
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Validate request using domain validation
	if err := req.Validate(); err != nil {
		h.log.Warn("registration validation failed", "error", err)

		// Check if it's a validation error with a field
		if valErr, ok := err.(*domain.ValidationError); ok {
			h.sendError(w, valErr.Message, http.StatusBadRequest, valErr.Field)
			return
		}

		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	// Create user
	user, err := h.authService.CreatePasswordUser(r.Context(), req.Email, req.Password, req.Name, req.BirthDate)
	if err != nil {
		h.log.Error("failed to create user", "error", err)

		// Check for specific errors
		switch err.Error() {
		case domain.ErrUserAlreadyExists.Error():
			h.sendError(w, "Email already registered", http.StatusConflict, "email")
			return
		default:
			h.sendError(w, "Failed to create user account", http.StatusInternalServerError, "")
			return
		}
	}

	h.log.Info("user registered successfully", "email", user.PrimaryEmail)

	// Generate OTP and send welcome email (non-blocking, fire-and-forget)
	otpCode, err := h.authService.GenerateEmailOTP(r.Context(), user.PrimaryEmail)
	if err != nil {
		h.log.Warn("failed to generate OTP", "email", user.PrimaryEmail, "error", err)
	} else {
		if err := h.authService.SendWelcomeEmail(r.Context(), user.PrimaryEmail, user.Name, otpCode); err != nil {
			h.log.Warn("failed to send welcome email", "email", user.PrimaryEmail, "error", err)
		} else {
			h.log.Info("welcome OTP email queued", "email", user.PrimaryEmail)
		}
	}

	// Send success response
	response := domain.RegisterResponse{
		ID:      user.ID.String(),
		Email:   user.PrimaryEmail,
		Name:    user.Name,
		Message: "Registration successful. Please verify your email to activate your account.",
	}

	h.sendSuccess(w, response, http.StatusCreated)
}
