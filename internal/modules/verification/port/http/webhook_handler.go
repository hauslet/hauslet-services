package http

import (
	"context"
	"encoding/json"
	"hauslet/cmd/api/server/middleware"
	authservice "hauslet/internal/modules/auth/service"
	"hauslet/internal/modules/verification/service"
	"hauslet/internal/platform/ratelimit"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
)

// WebhookHandler handles verification provider webhooks (Dojah, Veriff, etc.)
type WebhookHandler struct {
	verificationService service.VerificationService
	log                 *slog.Logger
}

// NewWebhookHandler creates a new verification webhook handler
func NewWebhookHandler(
	verificationService service.VerificationService,
	log *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		verificationService: verificationService,
		log:                 log,
	}
}

// SetupRoutes configures verification webhook routes (without rate limiting)
func (h *WebhookHandler) SetupRoutes(r chi.Router) {
	r.Post("/webhooks/verification/dojah", h.HandleDojahWebhook)
	r.Post("/webhooks/verification/veriff", h.HandleVeriffWebhook)
}

// SetupRoutesWithRateLimiting configures verification webhook routes with rate limiting
func (h *WebhookHandler) SetupRoutesWithRateLimiting(r chi.Router, limiter ratelimit.Limiter) {
	// Webhooks need higher limits than user endpoints
	webhookLimiter := middleware.RateLimitIP(limiter, 100, time.Minute)

	r.With(webhookLimiter).Post("/webhooks/verification/dojah", h.HandleDojahWebhook)
	r.With(webhookLimiter).Post("/webhooks/verification/veriff", h.HandleVeriffWebhook)
}

// HTTPHandler handles verification REST API endpoints
type HTTPHandler struct {
	verificationSvc service.VerificationService
	ctx             context.Context
	log             *slog.Logger
}

// NewHTTPHandler creates a new verification HTTP handler
func NewHTTPHandler(ctx context.Context, verificationSvc service.VerificationService, log *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		verificationSvc: verificationSvc,
		ctx:             ctx,
		log:             log,
	}
}

// SetupRoutes configures verification REST endpoints behind auth middleware
func (h *HTTPHandler) SetupRoutes(r chi.Router, authService authservice.AuthService) {

	authMiddleware := authService.OAuthService().Middleware()

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)

		// Session management
		r.Post("/verification/sessions", h.CreateSession)
		r.Get("/verification/sessions/{sessionID}", h.GetSession)
		r.Get("/verification/sessions", h.GetSessionByType)

		// Identity verification (KYC — Dojah/Veriff)
		r.Post("/verification/identity", h.SubmitIdentityVerification)

		// Phone verification (OTP)
		r.Post("/verification/phone/otp", h.GeneratePhoneOTP)
		r.Post("/verification/phone/verify", h.VerifyPhoneOTP)

		// Address verification (manual review)
		r.Post("/verification/address", h.SubmitAddressVerification)

		// Business verification (CAC lookup + manual review)
		r.Post("/verification/business", h.SubmitBusinessVerification)

		// Listing verification (manual review)
		r.Post("/verification/listing", h.SubmitListingVerification)

		// Evidence management
		r.Get("/verification/sessions/{sessionID}/evidence", h.ListSessionEvidence)
		r.Get("/verification/evidence/{evidenceID}", h.GetEvidence)
		r.Get("/verification/evidence/{evidenceID}/url", h.GenerateEvidenceURL)

		// Attempts
		r.Get("/verification/sessions/{sessionID}/attempts", h.ListAttempts)
	})
}

// SetupRoutesWithRateLimiting configures verification endpoints with rate limiting
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, authService authservice.AuthService, limiter ratelimit.Limiter) {
	authMiddleware := authService.OAuthService().Middleware()

	// Verification submissions are sensitive — tighter limits
	submitLimiter := middleware.RateLimitIP(limiter, 10, time.Minute)
	readLimiter := middleware.RateLimitIP(limiter, 60, time.Minute)
	otpLimiter := middleware.RateLimitIP(limiter, 5, time.Minute)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth)

		// Session management (read)
		r.With(readLimiter).Post("/verification/sessions", h.CreateSession)
		r.With(readLimiter).Get("/verification/sessions/{sessionID}", h.GetSession)
		r.With(readLimiter).Get("/verification/sessions", h.GetSessionByType)

		// Identity verification — submit
		r.With(submitLimiter).Post("/verification/identity", h.SubmitIdentityVerification)

		// Phone OTP — tightest limit
		r.With(otpLimiter).Post("/verification/phone/otp", h.GeneratePhoneOTP)
		r.With(otpLimiter).Post("/verification/phone/verify", h.VerifyPhoneOTP)

		// Document submissions
		r.With(submitLimiter).Post("/verification/address", h.SubmitAddressVerification)
		r.With(submitLimiter).Post("/verification/business", h.SubmitBusinessVerification)
		r.With(submitLimiter).Post("/verification/listing", h.SubmitListingVerification)

		// Evidence (read)
		r.With(readLimiter).Get("/verification/sessions/{sessionID}/evidence", h.ListSessionEvidence)
		r.With(readLimiter).Get("/verification/evidence/{evidenceID}", h.GetEvidence)
		r.With(readLimiter).Get("/verification/evidence/{evidenceID}/url", h.GenerateEvidenceURL)

		// Attempts (read)
		r.With(readLimiter).Get("/verification/sessions/{sessionID}/attempts", h.ListAttempts)
	})
}

// ============================================================
// Request/Response DTOs
// ============================================================

// CreateSessionHTTPRequest is the JSON body for session creation
type CreateSessionHTTPRequest struct {
	Type      string  `json:"type"`
	Tier      string  `json:"tier"`
	Country   string  `json:"country"`
	TargetID  *string `json:"target_id,omitempty"`
	IPAddress *string `json:"ip_address,omitempty"`
	UserAgent *string `json:"user_agent,omitempty"`

	// Verification data (type-specific)
	Data json.RawMessage `json:"data,omitempty"`
}

// SubmitIdentityHTTPRequest is the JSON body for identity verification
type SubmitIdentityHTTPRequest struct {
	SessionID      string  `json:"session_id"`
	SelfieImage    string  `json:"selfie_image"`   // base64
	DocumentImage  string  `json:"document_image"` // base64
	DocumentType   string  `json:"document_type"`  // passport, drivers_license, national_id, etc.
	DocumentNumber *string `json:"document_number,omitempty"`
	IPAddress      *string `json:"ip_address,omitempty"`
}

// SubmitAddressHTTPRequest is the JSON body for address verification
type SubmitAddressHTTPRequest struct {
	SessionID     string  `json:"session_id"`
	ProofDocument string  `json:"proof_document"` // base64
	DocumentType  string  `json:"document_type"`  // utility_bill, bank_statement, lease
	IPAddress     *string `json:"ip_address,omitempty"`
}

// SubmitBusinessHTTPRequest is the JSON body for business verification
type SubmitBusinessHTTPRequest struct {
	SessionID               string  `json:"session_id"`
	RegistrationDocument    string  `json:"registration_document"`               // base64
	TaxIDDocument           *string `json:"tax_id_document,omitempty"`           // base64
	BusinessLicenseDocument *string `json:"business_license_document,omitempty"` // base64
	IPAddress               *string `json:"ip_address,omitempty"`
}

// SubmitListingHTTPRequest is the JSON body for listing verification
type SubmitListingHTTPRequest struct {
	SessionID     string  `json:"session_id"`
	ProofDocument string  `json:"proof_document"` // base64
	DocumentType  string  `json:"document_type"`  // title_deed, property_tax, geo_tagged_photo
	IPAddress     *string `json:"ip_address,omitempty"`
}

// GenerateOTPHTTPRequest is the JSON body for OTP generation
type GenerateOTPHTTPRequest struct {
	SessionID string  `json:"session_id"`
	IPAddress *string `json:"ip_address,omitempty"`
}

// VerifyOTPHTTPRequest is the JSON body for OTP verification
type VerifyOTPHTTPRequest struct {
	SessionID string  `json:"session_id"`
	OTPCode   string  `json:"otp_code"`
	IPAddress *string `json:"ip_address,omitempty"`
}
