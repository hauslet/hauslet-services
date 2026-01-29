package service

import (
	"context"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// VerificationService handles all verification business logic and authorization
type VerificationService interface {
	// Session Management
	CreateSession(ctx context.Context, req CreateSessionRequest) (*domain.VerificationSession, error)
	GetSession(ctx context.Context, sessionID uuid.UUID, requesterID uuid.UUID) (*domain.VerificationSession, error)
	GetSessionByUser(ctx context.Context, userID uuid.UUID, requesterID uuid.UUID, vType domain.VerificationType) (*domain.VerificationSession, error)

	// Identity Verification (KYC)
	SubmitIdentityVerification(ctx context.Context, req SubmitIdentityVerificationRequest) (*SubmitVerificationResponse, error)

	// Phone Verification (OTP)
	GeneratePhoneOTP(ctx context.Context, req GeneratePhoneOTPRequest) (*GeneratePhoneOTPResponse, error)
	VerifyPhoneOTP(ctx context.Context, req VerifyPhoneOTPRequest) (*VerifyPhoneOTPResponse, error)

	// Address Verification
	SubmitAddressVerification(ctx context.Context, req SubmitAddressVerificationRequest) (*SubmitVerificationResponse, error)

	// Business Verification
	// Business Verification
	SubmitBusinessVerification(ctx context.Context, req SubmitBusinessVerificationRequest) (*SubmitVerificationResponse, error)

	// Listing Verification
	SubmitListingVerification(ctx context.Context, req SubmitListingVerificationRequest) (*SubmitVerificationResponse, error)

	// Evidence Management
	UploadEvidence(ctx context.Context, req UploadEvidenceRequest) (*domain.Evidence, error)
	GetEvidence(ctx context.Context, evidenceID uuid.UUID, requesterID uuid.UUID) (*domain.Evidence, error)
	ListSessionEvidence(ctx context.Context, sessionID uuid.UUID, requesterID uuid.UUID) ([]*domain.Evidence, error)
	GenerateEvidenceSignedURL(ctx context.Context, evidenceID uuid.UUID, requesterID uuid.UUID) (string, error)

	// Webhook Processing
	ProcessWebhook(ctx context.Context, req ProcessWebhookRequest) error

	// Session Queries
	ListAttempts(ctx context.Context, sessionID uuid.UUID, requesterID uuid.UUID) ([]*domain.VerificationAttempt, error)

	// Admin Operations
	ExpireOldSessions(ctx context.Context) (int, error)
	ApproveVerificationSession(ctx context.Context, sessionID uuid.UUID) error
}
