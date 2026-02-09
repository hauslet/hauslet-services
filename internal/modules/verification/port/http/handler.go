package http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/modules/verification/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ============================================================
// Session Handlers
// ============================================================

// CreateSession creates a new verification session
// @Summary Create verification session
// @Description Creates a new verification session of the specified type
// @Tags verification
// @Accept json
// @Produce json
// @Param request body CreateSessionHTTPRequest true "Session creation request"
// @Success 201 {object} domain.VerificationSession
// @Failure 400 {object} object "Bad Request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/sessions [post]
func (h *HTTPHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req CreateSessionHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	// Parse verification type
	vType := domain.VerificationType(req.Type)
	if !vType.IsValid() {
		h.sendError(w, fmt.Sprintf("Invalid verification type: %s", req.Type), http.StatusBadRequest, "type")
		return
	}

	// Parse tier
	tier := domain.VerificationTier(req.Tier)
	if !tier.IsValid() {
		h.sendError(w, fmt.Sprintf("Invalid verification tier: %s", req.Tier), http.StatusBadRequest, "tier")
		return
	}

	// Parse target ID if provided
	var targetID *uuid.UUID
	if req.TargetID != nil {
		parsed, err := uuid.Parse(*req.TargetID)
		if err != nil {
			h.sendError(w, "Invalid target_id format", http.StatusBadRequest, "target_id")
			return
		}
		targetID = &parsed
	}

	// Parse verification data from JSON
	var data domain.VerificationData
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &data); err != nil {
			h.sendError(w, "Invalid verification data", http.StatusBadRequest, "data")
			return
		}
	}

	session, err := h.verificationSvc.CreateSession(r.Context(), service.CreateSessionRequest{
		UserID:    userID,
		TargetID:  targetID,
		Type:      vType,
		Tier:      tier,
		Data:      data,
		Country:   req.Country,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, session, http.StatusCreated)
}

// GetSession retrieves a verification session by ID
// @Summary Get verification session
// @Description Retrieves a verification session by its ID
// @Tags verification
// @Produce json
// @Param sessionID path string true "Session ID"
// @Success 200 {object} domain.VerificationSession
// @Failure 404 {object} object "Not Found"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/sessions/{sessionID} [get]
func (h *HTTPHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		h.sendError(w, "Invalid session ID", http.StatusBadRequest, "session_id")
		return
	}

	session, err := h.verificationSvc.GetSession(r.Context(), sessionID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, session, http.StatusOK)
}

// GetSessionByType retrieves a user's verification session by type
// @Summary Get verification session by type
// @Description Retrieves the current user's verification session for a given type
// @Tags verification
// @Produce json
// @Param type query string true "Verification type (identity, phone, address, business, listing)"
// @Success 200 {object} domain.VerificationSession
// @Failure 404 {object} object "Not Found"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/sessions [get]
func (h *HTTPHandler) GetSessionByType(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	vTypeStr := r.URL.Query().Get("type")
	if vTypeStr == "" {
		h.sendError(w, "Query parameter 'type' is required", http.StatusBadRequest, "type")
		return
	}

	vType := domain.VerificationType(vTypeStr)
	if !vType.IsValid() {
		h.sendError(w, fmt.Sprintf("Invalid verification type: %s", vTypeStr), http.StatusBadRequest, "type")
		return
	}

	session, err := h.verificationSvc.GetSessionByUser(r.Context(), userID, userID, vType)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, session, http.StatusOK)
}

// ============================================================
// Identity Verification Handlers
// ============================================================

// SubmitIdentityVerification submits KYC identity verification
// @Summary Submit identity verification
// @Description Submits identity documents for automated KYC verification via Dojah/Veriff
// @Tags verification
// @Accept json
// @Produce json
// @Param request body SubmitIdentityHTTPRequest true "Identity verification request"
// @Success 200 {object} service.SubmitVerificationResponse
// @Failure 400 {object} object "Bad Request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/identity [post]
func (h *HTTPHandler) SubmitIdentityVerification(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req SubmitIdentityHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.sendError(w, "Invalid session_id", http.StatusBadRequest, "session_id")
		return
	}

	selfie, err := base64.StdEncoding.DecodeString(req.SelfieImage)
	if err != nil {
		h.sendError(w, "Invalid base64 encoding for selfie_image", http.StatusBadRequest, "selfie_image")
		return
	}

	document, err := base64.StdEncoding.DecodeString(req.DocumentImage)
	if err != nil {
		h.sendError(w, "Invalid base64 encoding for document_image", http.StatusBadRequest, "document_image")
		return
	}

	docType := domain.DocumentType(req.DocumentType)
	if !docType.IsValid() {
		h.sendError(w, fmt.Sprintf("Invalid document type: %s", req.DocumentType), http.StatusBadRequest, "document_type")
		return
	}

	resp, err := h.verificationSvc.SubmitIdentityVerification(r.Context(), service.SubmitIdentityVerificationRequest{
		SessionID:      sessionID,
		UserID:         userID,
		SelfieImage:    selfie,
		DocumentImage:  document,
		DocumentType:   docType,
		DocumentNumber: req.DocumentNumber,
		IPAddress:      req.IPAddress,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, resp, http.StatusOK)
}

// ============================================================
// Phone Verification Handlers
// ============================================================

// GeneratePhoneOTP generates an OTP for phone verification
// @Summary Generate phone OTP
// @Description Sends an OTP code to the phone number associated with the session
// @Tags verification
// @Accept json
// @Produce json
// @Param request body GenerateOTPHTTPRequest true "OTP generation request"
// @Success 200 {object} service.GeneratePhoneOTPResponse
// @Failure 400 {object} object "Bad Request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/phone/otp [post]
func (h *HTTPHandler) GeneratePhoneOTP(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req GenerateOTPHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.sendError(w, "Invalid session_id", http.StatusBadRequest, "session_id")
		return
	}

	resp, err := h.verificationSvc.GeneratePhoneOTP(r.Context(), service.GeneratePhoneOTPRequest{
		SessionID: sessionID,
		UserID:    userID,
		IPAddress: req.IPAddress,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, resp, http.StatusOK)
}

// VerifyPhoneOTP verifies an OTP code for phone verification
// @Summary Verify phone OTP
// @Description Verifies the OTP code submitted by the user
// @Tags verification
// @Accept json
// @Produce json
// @Param request body VerifyOTPHTTPRequest true "OTP verification request"
// @Success 200 {object} service.VerifyPhoneOTPResponse
// @Failure 400 {object} object "Bad Request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/phone/verify [post]
func (h *HTTPHandler) VerifyPhoneOTP(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req VerifyOTPHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.sendError(w, "Invalid session_id", http.StatusBadRequest, "session_id")
		return
	}

	if req.OTPCode == "" {
		h.sendError(w, "otp_code is required", http.StatusBadRequest, "otp_code")
		return
	}

	resp, err := h.verificationSvc.VerifyPhoneOTP(r.Context(), service.VerifyPhoneOTPRequest{
		SessionID: sessionID,
		UserID:    userID,
		OTPCode:   req.OTPCode,
		IPAddress: req.IPAddress,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, resp, http.StatusOK)
}

// ============================================================
// Address Verification Handlers
// ============================================================

// SubmitAddressVerification submits address verification documents
// @Summary Submit address verification
// @Description Submits address proof documents for manual review
// @Tags verification
// @Accept json
// @Produce json
// @Param request body SubmitAddressHTTPRequest true "Address verification request"
// @Success 200 {object} service.SubmitVerificationResponse
// @Failure 400 {object} object "Bad Request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/address [post]
func (h *HTTPHandler) SubmitAddressVerification(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req SubmitAddressHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.sendError(w, "Invalid session_id", http.StatusBadRequest, "session_id")
		return
	}

	proofDoc, err := base64.StdEncoding.DecodeString(req.ProofDocument)
	if err != nil {
		h.sendError(w, "Invalid base64 encoding for proof_document", http.StatusBadRequest, "proof_document")
		return
	}

	resp, err := h.verificationSvc.SubmitAddressVerification(r.Context(), service.SubmitAddressVerificationRequest{
		SessionID:     sessionID,
		UserID:        userID,
		ProofDocument: proofDoc,
		DocumentType:  req.DocumentType,
		IPAddress:     req.IPAddress,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, resp, http.StatusOK)
}

// ============================================================
// Business Verification Handlers
// ============================================================

// SubmitBusinessVerification submits business verification documents
// @Summary Submit business verification
// @Description Submits business registration documents. For Nigerian businesses, automated CAC lookup is attempted first.
// @Tags verification
// @Accept json
// @Produce json
// @Param request body SubmitBusinessHTTPRequest true "Business verification request"
// @Success 200 {object} service.SubmitVerificationResponse
// @Failure 400 {object} object "Bad Request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/business [post]
func (h *HTTPHandler) SubmitBusinessVerification(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req SubmitBusinessHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.sendError(w, "Invalid session_id", http.StatusBadRequest, "session_id")
		return
	}

	regDoc, err := base64.StdEncoding.DecodeString(req.RegistrationDocument)
	if err != nil {
		h.sendError(w, "Invalid base64 encoding for registration_document", http.StatusBadRequest, "registration_document")
		return
	}

	var taxDoc *[]byte
	if req.TaxIDDocument != nil {
		decoded, err := base64.StdEncoding.DecodeString(*req.TaxIDDocument)
		if err != nil {
			h.sendError(w, "Invalid base64 encoding for tax_id_document", http.StatusBadRequest, "tax_id_document")
			return
		}
		taxDoc = &decoded
	}

	var licDoc *[]byte
	if req.BusinessLicenseDocument != nil {
		decoded, err := base64.StdEncoding.DecodeString(*req.BusinessLicenseDocument)
		if err != nil {
			h.sendError(w, "Invalid base64 encoding for business_license_document", http.StatusBadRequest, "business_license_document")
			return
		}
		licDoc = &decoded
	}

	resp, err := h.verificationSvc.SubmitBusinessVerification(r.Context(), service.SubmitBusinessVerificationRequest{
		SessionID:               sessionID,
		UserID:                  userID,
		RegistrationDocument:    regDoc,
		TaxIDDocument:           taxDoc,
		BusinessLicenseDocument: licDoc,
		IPAddress:               req.IPAddress,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, resp, http.StatusOK)
}

// ============================================================
// Listing Verification Handlers
// ============================================================

// SubmitListingVerification submits listing verification documents
// @Summary Submit listing verification
// @Description Submits listing proof documents for manual review
// @Tags verification
// @Accept json
// @Produce json
// @Param request body SubmitListingHTTPRequest true "Listing verification request"
// @Success 200 {object} service.SubmitVerificationResponse
// @Failure 400 {object} object "Bad Request"
// @Failure 401 {object} object "Unauthorized"
// @Security BearerAuth
// @Router /verification/listing [post]
func (h *HTTPHandler) SubmitListingVerification(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	var req SubmitListingHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest, "")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		h.sendError(w, "Invalid session_id", http.StatusBadRequest, "session_id")
		return
	}

	proofDoc, err := base64.StdEncoding.DecodeString(req.ProofDocument)
	if err != nil {
		h.sendError(w, "Invalid base64 encoding for proof_document", http.StatusBadRequest, "proof_document")
		return
	}

	resp, err := h.verificationSvc.SubmitListingVerification(r.Context(), service.SubmitListingVerificationRequest{
		SessionID:     sessionID,
		UserID:        userID,
		ProofDocument: proofDoc,
		DocumentType:  req.DocumentType,
		IPAddress:     req.IPAddress,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, resp, http.StatusOK)
}

// ============================================================
// Evidence Handlers
// ============================================================

// ListSessionEvidence lists all evidence for a verification session
// @Summary List session evidence
// @Description Returns all uploaded evidence items for a verification session
// @Tags verification
// @Produce json
// @Param sessionID path string true "Session ID"
// @Success 200 {array} domain.Evidence
// @Failure 401 {object} object "Unauthorized"
// @Failure 404 {object} object "Not Found"
// @Security BearerAuth
// @Router /verification/sessions/{sessionID}/evidence [get]
func (h *HTTPHandler) ListSessionEvidence(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		h.sendError(w, "Invalid session ID", http.StatusBadRequest, "session_id")
		return
	}

	evidence, err := h.verificationSvc.ListSessionEvidence(r.Context(), sessionID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, evidence, http.StatusOK)
}

// GetEvidence retrieves a specific evidence item
// @Summary Get evidence
// @Description Retrieves evidence metadata by ID
// @Tags verification
// @Produce json
// @Param evidenceID path string true "Evidence ID"
// @Success 200 {object} domain.Evidence
// @Failure 401 {object} object "Unauthorized"
// @Failure 404 {object} object "Not Found"
// @Security BearerAuth
// @Router /verification/evidence/{evidenceID} [get]
func (h *HTTPHandler) GetEvidence(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	evidenceID, err := uuid.Parse(chi.URLParam(r, "evidenceID"))
	if err != nil {
		h.sendError(w, "Invalid evidence ID", http.StatusBadRequest, "evidence_id")
		return
	}

	evidence, err := h.verificationSvc.GetEvidence(r.Context(), evidenceID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, evidence, http.StatusOK)
}

// GenerateEvidenceURL generates a signed URL for evidence download
// @Summary Generate evidence download URL
// @Description Generates a presigned URL to download evidence
// @Tags verification
// @Produce json
// @Param evidenceID path string true "Evidence ID"
// @Success 200 {object} object "URL response"
// @Failure 401 {object} object "Unauthorized"
// @Failure 404 {object} object "Not Found"
// @Security BearerAuth
// @Router /verification/evidence/{evidenceID}/url [get]
func (h *HTTPHandler) GenerateEvidenceURL(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	evidenceID, err := uuid.Parse(chi.URLParam(r, "evidenceID"))
	if err != nil {
		h.sendError(w, "Invalid evidence ID", http.StatusBadRequest, "evidence_id")
		return
	}

	url, err := h.verificationSvc.GenerateEvidenceSignedURL(r.Context(), evidenceID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, map[string]string{"url": url}, http.StatusOK)
}

// ============================================================
// Attempt Handlers
// ============================================================

// ListAttempts lists all verification attempts for a session
// @Summary List verification attempts
// @Description Returns all verification attempts for a session
// @Tags verification
// @Produce json
// @Param sessionID path string true "Session ID"
// @Success 200 {array} domain.VerificationAttempt
// @Failure 401 {object} object "Unauthorized"
// @Failure 404 {object} object "Not Found"
// @Security BearerAuth
// @Router /verification/sessions/{sessionID}/attempts [get]
func (h *HTTPHandler) ListAttempts(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized, "")
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		h.sendError(w, "Invalid session ID", http.StatusBadRequest, "session_id")
		return
	}

	attempts, err := h.verificationSvc.ListAttempts(r.Context(), sessionID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendSuccess(w, attempts, http.StatusOK)
}

// ============================================================
// Helper Methods
// ============================================================

// extractUserID extracts and parses the authenticated user ID from the request
func (h *HTTPHandler) extractUserID(r *http.Request) (uuid.UUID, error) {
	userIDStr := authmiddleware.GetUserID(r)
	if userIDStr == "" {
		return uuid.Nil, fmt.Errorf("unauthorized")
	}
	return uuid.Parse(userIDStr)
}

// sendSuccess writes a JSON success response
func (h *HTTPHandler) sendSuccess(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// sendError writes a JSON error response
func (h *HTTPHandler) sendError(w http.ResponseWriter, message string, statusCode int, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Field   string `json:"field,omitempty"`
	}{
		Error:   http.StatusText(statusCode),
		Message: message,
		Field:   field,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// handleServiceError maps service/domain errors to appropriate HTTP status codes
func (h *HTTPHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSessionNotFound):
		h.sendError(w, err.Error(), http.StatusNotFound, "")
	case errors.Is(err, domain.ErrSessionExpired):
		h.sendError(w, err.Error(), http.StatusGone, "")
	case errors.Is(err, domain.ErrInvalidSessionStatus):
		h.sendError(w, err.Error(), http.StatusConflict, "")
	case errors.Is(err, domain.ErrSessionTypeMismatch):
		h.sendError(w, err.Error(), http.StatusBadRequest, "")
	case errors.Is(err, domain.ErrMaxAttemptsExceeded):
		h.sendError(w, err.Error(), http.StatusTooManyRequests, "")
	case errors.Is(err, domain.ErrUnauthorized):
		h.sendError(w, err.Error(), http.StatusForbidden, "")
	default:
		h.log.Error("verification handler error", "error", err)
		h.sendError(w, "Internal server error", http.StatusInternalServerError, "")
	}
}
