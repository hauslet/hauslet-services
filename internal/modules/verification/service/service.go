package service

import (
	"hauslet/config"
	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/modules/verification/port"
	"hauslet/internal/modules/verification/repository"
	"hauslet/internal/platform/breaker"
	"hauslet/internal/platform/evidence"
	"hauslet/internal/platform/kyc"
	"hauslet/internal/platform/ratelimit"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/sms"
	"log/slog"

	"github.com/google/uuid"
)

// verificationService implements VerificationService
type verificationService struct {
	repo            repository.VerificationRepository
	kycClient       *kyc.Client
	smsClient       *sms.Client
	evidenceStore   evidence.Store
	rateLimiter     ratelimit.Limiter
	circuitBreaker  breaker.CircuitBreaker
	redisClient     redis.RedisClient
	profileAdapter  port.ProfileAdapter
	businessAdapter port.BusinessAdapter
	config          *config.GlobalConfig
	logger          *slog.Logger
}

// NewVerificationService creates a new verification service
func NewVerificationService(
	repo repository.VerificationRepository,
	kycClient *kyc.Client,
	smsClient *sms.Client,
	evidenceStore evidence.Store,
	rateLimiter ratelimit.Limiter,
	circuitBreaker breaker.CircuitBreaker,
	redisClient redis.RedisClient,
	profileAdapter port.ProfileAdapter,
	businessAdapter port.BusinessAdapter,
	cfg *config.GlobalConfig,
	logger *slog.Logger,
) VerificationService {
	// Use no-op adapters if nil
	if profileAdapter == nil {
		profileAdapter = &port.NoopProfileAdapter{}
	}
	if businessAdapter == nil {
		businessAdapter = &port.NoopBusinessAdapter{}
	}

	return &verificationService{
		repo:            repo,
		kycClient:       kycClient,
		smsClient:       smsClient,
		evidenceStore:   evidenceStore,
		rateLimiter:     rateLimiter,
		circuitBreaker:  circuitBreaker,
		redisClient:     redisClient,
		profileAdapter:  profileAdapter,
		businessAdapter: businessAdapter,
		config:          cfg,
		logger:          logger,
	}
}

// CreateSessionRequest contains data for creating a verification session
type CreateSessionRequest struct {
	UserID    uuid.UUID
	Type      domain.VerificationType
	Tier      domain.VerificationTier
	Data      domain.VerificationData
	Country   string
	IPAddress *string
	UserAgent *string
}

// SubmitIdentityVerificationRequest contains data for submitting identity verification
type SubmitIdentityVerificationRequest struct {
	SessionID      uuid.UUID
	UserID         uuid.UUID // For authorization
	SelfieImage    []byte
	DocumentImage  []byte
	DocumentType   domain.DocumentType
	DocumentNumber *string
	IPAddress      *string
}

// SubmitAddressVerificationRequest contains data for submitting address verification
type SubmitAddressVerificationRequest struct {
	SessionID     uuid.UUID
	UserID        uuid.UUID // For authorization
	ProofDocument []byte
	DocumentType  string // utility_bill, bank_statement, lease, etc.
	IPAddress     *string
}

// SubmitBusinessVerificationRequest contains data for submitting business verification
type SubmitBusinessVerificationRequest struct {
	SessionID               uuid.UUID
	UserID                  uuid.UUID // For authorization
	RegistrationDocument    []byte
	TaxIDDocument           *[]byte
	BusinessLicenseDocument *[]byte
	IPAddress               *string
}

// GeneratePhoneOTPRequest contains data for generating OTP
type GeneratePhoneOTPRequest struct {
	SessionID uuid.UUID
	UserID    uuid.UUID // For authorization
	IPAddress *string
}

// GeneratePhoneOTPResponse contains OTP generation result
type GeneratePhoneOTPResponse struct {
	Session     *domain.VerificationSession
	OTPSent     bool
	ExpiresAt   *string // ISO 8601 timestamp
	SMSProvider string
	Message     string
}

// VerifyPhoneOTPRequest contains data for verifying OTP
type VerifyPhoneOTPRequest struct {
	SessionID uuid.UUID
	UserID    uuid.UUID // For authorization
	OTPCode   string
	IPAddress *string
}

// VerifyPhoneOTPResponse contains OTP verification result
type VerifyPhoneOTPResponse struct {
	Session   *domain.VerificationSession
	Verified  bool
	Remaining int // Remaining attempts
	Message   string
}

// SubmitVerificationResponse contains the result of verification submission
type SubmitVerificationResponse struct {
	Attempt      *domain.VerificationAttempt
	Session      *domain.VerificationSession
	ProviderName string
	Status       string
	Message      string
}

// UploadEvidenceRequest contains data for uploading evidence
type UploadEvidenceRequest struct {
	SessionID uuid.UUID
	UserID    uuid.UUID // For authorization
	Type      domain.EvidenceType
	Data      []byte
	MimeType  string
	IPAddress *string
}

// ProcessWebhookRequest contains webhook data from KYC providers
type ProcessWebhookRequest struct {
	ProviderName string
	Payload      []byte
	Headers      map[string]string
	Signature    string
}
