package http

import (
	"encoding/json"
	"hauslet/internal/modules/auth/domain"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"net"
	"net/http"
	"strings"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
)

// VerifyEmailRequest represents the email verification request
type VerifyEmailRequest struct {
	Email            string `json:"email"`
	Code             string `json:"code"`
	LoginAfterVerify bool   `json:"login_after_verify,omitempty"`
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

func resolveClientIP(r *http.Request) string {
	if ip := authmiddleware.GetIPFromContext(r.Context()); ip != "" {
		return ip
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if ip, _, found := strings.Cut(forwarded, ","); found {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(forwarded)
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// VerifyEmail handles email verification with OTP
// @Summary Verify email address
// @Description Verify user's email address using OTP code
// @Tags auth
// @Accept json
// @Produce json
// @Param request body VerifyEmailRequest true "Verification details"
// @Success 200 {object} map[string]interface{}
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

	// Default behavior remains unchanged unless explicitly requested by client.
	if !req.LoginAfterVerify {
		h.sendSuccess(w, map[string]string{
			"message": "Email verified successfully. You can now log in.",
		}, http.StatusOK)
		return
	}

	user, err := h.authService.GetUserByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		h.log.Error("Email verified but failed to load user for auto-login", "email", req.Email, "error", err)
		h.sendError(w, "Email verified but failed to complete login", http.StatusInternalServerError, "")
		return
	}
	if !user.IsActive {
		h.log.Warn("Email verified but auto-login blocked for inactive account", "email", req.Email, "user_id", user.ID)
		h.sendError(w, "Account is deactivated", http.StatusForbidden, "")
		return
	}

	// Store metadata so session creation keeps IP/User-Agent context.
	h.authService.StoreRequestMetadata(user.PrimaryEmail, resolveClientIP(r), r.Header.Get("User-Agent"))

	// Intentionally use password provider semantics so existing 2FA policies are applied.
	claims := token.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{h.authService.GetSiteURL()},
		},
		User: &token.User{
			ID:    "password_" + user.ID.String(),
			Name:  user.Name,
			Email: user.PrimaryEmail,
		},
	}

	tokenService := h.authService.OAuthService().TokenService()
	issuedClaims, err := tokenService.Set(w, claims)
	if err != nil {
		h.log.Error("Failed to issue JWT after email verification", "email", req.Email, "user_id", user.ID, "error", err)
		h.sendError(w, "Email verified but failed to complete login", http.StatusInternalServerError, "")
		return
	}

	if issuedClaims.User != nil && issuedClaims.User.StrAttr("login_state") == "2fa_pending" {
		h.sendSuccess(w, map[string]any{
			"message":      "Email verified. Two-factor authentication required to complete login.",
			"requires_2fa": true,
			"temp_token":   issuedClaims.User.StrAttr("2fa_temp_token"),
			"method":       issuedClaims.User.StrAttr("2fa_method"),
		}, http.StatusOK)
		return
	}

	h.sendSuccess(w, map[string]any{
		"message": "Email verified and login successful",
		"user": map[string]any{
			"id":    user.ID.String(),
			"email": user.PrimaryEmail,
			"name":  user.Name,
			"role":  user.Role,
		},
	}, http.StatusOK)
}
