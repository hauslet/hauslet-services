package http

import (
	"encoding/json"
	"net/http"

	"hauslet/internal/modules/auth/domain"
	authmiddleware "hauslet/internal/modules/auth/middleware"
)

// ============================================================================
// Two-Factor Authentication HTTP Handlers
// ============================================================================

// Setup2FARequest is the request body for initiating 2FA setup
type Setup2FARequest struct {
	Method      string `json:"method"`       // "email", "sms", "authenticator"
	PhoneNumber string `json:"phone_number"` // Required for SMS method
}

// Verify2FARequest is the request body for 2FA verification
type Verify2FARequest struct {
	Code string `json:"code"`
}

// Get2FAStatus returns the current 2FA status
// @Summary Get 2FA status
// @Description Returns whether 2FA is enabled and which method
// @Tags 2FA
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.TwoFactorStatus
// @Failure 401 {object} domain.ErrorResponse
// @Router /2fa/status [get]
func (h *HTTPHandler) Get2FAStatus(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "unauthorized", http.StatusUnauthorized, "")
		return
	}

	status, err := h.authService.Get2FAStatus(r.Context(), userID)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, status, http.StatusOK)
}

// InitiateSetup2FA starts the 2FA setup process
// @Summary Initiate 2FA setup
// @Description Starts 2FA setup with the selected method
// @Tags 2FA
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body Setup2FARequest true "Setup request"
// @Success 200 {object} domain.SetupResponse
// @Failure 400 {object} domain.ErrorResponse
// @Router /2fa/setup [post]
func (h *HTTPHandler) InitiateSetup2FA(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req Setup2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Method == "" {
		h.sendError(w, "method is required", http.StatusBadRequest, "method")
		return
	}

	method := domain.TwoFactorMethod(req.Method)
	if method != domain.TwoFactorEmail && method != domain.TwoFactorSMS && method != domain.TwoFactorAuthenticator {
		h.sendError(w, "invalid method, must be 'email', 'sms', or 'authenticator'", http.StatusBadRequest, "method")
		return
	}

	response, err := h.authService.InitiateSetup2FA(r.Context(), userID, method, req.PhoneNumber)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	h.sendSuccess(w, response, http.StatusOK)
}

// CompleteSetup2FA completes 2FA setup by verifying the code
// @Summary Complete 2FA setup
// @Description Verifies the code and enables 2FA
// @Tags 2FA
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body Verify2FARequest true "Verification code"
// @Success 200 {object} domain.BackupCodesResult
// @Failure 400 {object} domain.ErrorResponse
// @Router /2fa/setup/complete [post]
func (h *HTTPHandler) CompleteSetup2FA(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Code == "" {
		h.sendError(w, "code is required", http.StatusBadRequest, "code")
		return
	}

	result, err := h.authService.CompleteSetup2FA(r.Context(), userID, req.Code)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	h.sendSuccess(w, result, http.StatusOK)
}

// Send2FACode sends a 2FA verification code (for Email/SMS methods)
// @Summary Send 2FA code
// @Description Sends a new verification code for Email/SMS 2FA
// @Tags 2FA
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 400 {object} domain.ErrorResponse
// @Router /2fa/send-code [post]
func (h *HTTPHandler) Send2FACode(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "unauthorized", http.StatusUnauthorized, "")
		return
	}

	if err := h.authService.Send2FACode(r.Context(), userID); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	h.sendSuccess(w, map[string]bool{"sent": true}, http.StatusOK)
}

// Verify2FACode verifies a 2FA code during login
// @Summary Verify 2FA code
// @Description Verifies the 2FA code to complete login
// @Tags 2FA
// @Accept json
// @Produce json
// @Param request body Verify2FARequest true "Verification code"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} domain.ErrorResponse
// @Router /2fa/verify [post]
func (h *HTTPHandler) Verify2FACode(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Code == "" {
		h.sendError(w, "code is required", http.StatusBadRequest, "code")
		return
	}

	if err := h.authService.Verify2FACode(r.Context(), userID, req.Code); err != nil {
		// Try backup code
		if bkErr := h.authService.Verify2FABackupCode(r.Context(), userID, req.Code); bkErr != nil {
			h.sendError(w, "invalid verification code", http.StatusBadRequest, "code")
			return
		}
	}

	h.sendSuccess(w, map[string]bool{"verified": true}, http.StatusOK)
}

// Disable2FA disables 2FA for the user
// @Summary Disable 2FA
// @Description Disables 2FA after verifying current code
// @Tags 2FA
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body Verify2FARequest true "Current 2FA code"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} domain.ErrorResponse
// @Router /2fa/disable [post]
func (h *HTTPHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Code == "" {
		h.sendError(w, "code is required", http.StatusBadRequest, "code")
		return
	}

	if err := h.authService.Disable2FA(r.Context(), userID, req.Code); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	h.sendSuccess(w, map[string]bool{"disabled": true}, http.StatusOK)
}

// RegenerateBackupCodes generates new backup codes
// @Summary Regenerate backup codes
// @Description Generates new backup codes after verifying current 2FA code
// @Tags 2FA
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body Verify2FARequest true "Current 2FA code"
// @Success 200 {object} domain.BackupCodesResult
// @Failure 400 {object} domain.ErrorResponse
// @Router /2fa/backup-codes [post]
func (h *HTTPHandler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	userID := authmiddleware.GetUserID(r)
	if userID == "" {
		h.sendError(w, "unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "invalid request body", http.StatusBadRequest, "")
		return
	}

	if req.Code == "" {
		h.sendError(w, "code is required", http.StatusBadRequest, "code")
		return
	}

	result, err := h.authService.RegenerateBackupCodes(r.Context(), userID, req.Code)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
		return
	}

	h.sendSuccess(w, result, http.StatusOK)
}
