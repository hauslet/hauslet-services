package graphql

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/modules/verification/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles verification-specific GraphQL operations
type Resolver struct {
	verificationService service.VerificationService
	log                 *slog.Logger
}

func NewResolver(verificationService service.VerificationService, log *slog.Logger) *Resolver {
	return &Resolver{
		verificationService: verificationService,
		log:                 log,
	}
}

// =======================
// Queries
// =======================

// MyVerificationSession retrieves the user's verification session by type
func (r *Resolver) MyVerificationSession(ctx context.Context, vType domain.VerificationType) (*domain.VerificationSession, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	session, err := r.verificationService.GetSessionByUser(ctx, userUUID, userUUID, vType)
	if err != nil {
		if err == domain.ErrSessionNotFound {
			return nil, nil
		}
		r.log.Error("failed to get verification session", "error", err)
		return nil, err
	}

	return session, nil
}

// VerificationSession retrieves a specific verification session (admin or owner)
func (r *Resolver) VerificationSession(ctx context.Context, id uuid.UUID) (*domain.VerificationSession, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	session, err := r.verificationService.GetSession(ctx, id, userUUID)
	if err != nil {
		r.log.Error("failed to get verification session", "session_id", id, "error", err)
		return nil, err
	}

	return session, nil
}

// VerificationAttempts lists attempts for a session (admin or owner)
func (r *Resolver) VerificationAttempts(ctx context.Context, sessionID uuid.UUID) ([]*domain.VerificationAttempt, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	attempts, err := r.verificationService.ListAttempts(ctx, sessionID, userUUID)
	if err != nil {
		r.log.Error("failed to list verification attempts", "session_id", sessionID, "error", err)
		return nil, err
	}

	return attempts, nil
}

// =======================
// Mutations - Phone Verification
// =======================

// CreatePhoneVerification creates a new phone verification session
func (r *Resolver) CreatePhoneVerification(ctx context.Context, input CreatePhoneVerificationInput) (*domain.VerificationSession, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	phoneData := &domain.PhoneData{
		PhoneNumber: input.PhoneNumber,
		CountryCode: input.Country,
	}

	req := service.CreateSessionRequest{
		UserID:  userUUID,
		Type:    domain.VerificationPhone,
		Tier:    domain.TierBasic,
		Country: input.Country,
		Data: domain.VerificationData{
			Phone: phoneData,
		},
	}

	session, err := r.verificationService.CreateSession(ctx, req)
	if err != nil {
		r.log.Error("failed to create phone verification session", "error", err)
		return nil, err
	}

	r.log.Info("phone verification session created", "session_id", session.ID, "user_id", v.UserID)
	return session, nil
}

// GeneratePhoneOTP generates and sends OTP for phone verification
func (r *Resolver) GeneratePhoneOTP(ctx context.Context, sessionID uuid.UUID) (*OTPResponse, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	req := service.GeneratePhoneOTPRequest{
		SessionID: sessionID,
		UserID:    userUUID,
	}

	resp, err := r.verificationService.GeneratePhoneOTP(ctx, req)
	if err != nil {
		r.log.Error("failed to generate phone OTP", "session_id", sessionID, "error", err)
		return nil, err
	}

	return &OTPResponse{
		SessionID:   resp.Session.ID,
		OTPSent:     resp.OTPSent,
		ExpiresAt:   *resp.ExpiresAt,
		SMSProvider: resp.SMSProvider,
		Message:     resp.Message,
	}, nil
}

// VerifyPhoneOTP verifies the OTP code
func (r *Resolver) VerifyPhoneOTP(ctx context.Context, sessionID uuid.UUID, code string) (*OTPVerificationResponse, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	req := service.VerifyPhoneOTPRequest{
		SessionID: sessionID,
		UserID:    userUUID,
		OTPCode:   code,
	}

	resp, err := r.verificationService.VerifyPhoneOTP(ctx, req)
	if err != nil {
		r.log.Error("failed to verify phone OTP", "session_id", sessionID, "error", err)
		return nil, err
	}

	return &OTPVerificationResponse{
		SessionID:         resp.Session.ID,
		Verified:          resp.Verified,
		RemainingAttempts: resp.Remaining,
		Message:           resp.Message,
		Session:           resp.Session,
	}, nil
}

// =======================
// Mutations - Identity Verification (KYC)
// =======================

// CreateIdentityVerification creates a new identity verification session
func (r *Resolver) CreateIdentityVerification(ctx context.Context, input CreateIdentityVerificationInput) (*domain.VerificationSession, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	var dob time.Time
	if input.DateOfBirth != nil {
		dob = *input.DateOfBirth
	}

	identityData := &domain.IdentityData{
		ApplicantInfo: domain.ApplicantInfo{
			FirstName:   input.FirstName,
			LastName:    input.LastName,
			DateOfBirth: dob,
		},
	}

	req := service.CreateSessionRequest{
		UserID:  userUUID,
		Type:    domain.VerificationIdentity,
		Tier:    input.Tier,
		Country: input.Country,
		Data: domain.VerificationData{
			Identity: identityData,
		},
	}

	session, err := r.verificationService.CreateSession(ctx, req)
	if err != nil {
		r.log.Error("failed to create identity verification session", "error", err)
		return nil, err
	}

	r.log.Info("identity verification session created", "session_id", session.ID, "user_id", v.UserID, "tier", input.Tier)
	return session, nil
}

// SubmitIdentityVerification submits identity verification documents
func (r *Resolver) SubmitIdentityVerification(ctx context.Context, input SubmitIdentityVerificationInput) (*VerificationSubmitResponse, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	selfieBytes, err := base64.StdEncoding.DecodeString(input.SelfieImage)
	if err != nil {
		return nil, fmt.Errorf("invalid selfie image encoding")
	}

	documentBytes, err := base64.StdEncoding.DecodeString(input.DocumentImage)
	if err != nil {
		return nil, fmt.Errorf("invalid document image encoding")
	}

	req := service.SubmitIdentityVerificationRequest{
		SessionID:      input.SessionID,
		UserID:         userUUID,
		SelfieImage:    selfieBytes,
		DocumentImage:  documentBytes,
		DocumentType:   input.DocumentType,
		DocumentNumber: input.DocumentNumber,
	}

	resp, err := r.verificationService.SubmitIdentityVerification(ctx, req)
	if err != nil {
		r.log.Error("failed to submit identity verification", "session_id", input.SessionID, "error", err)
		return nil, err
	}

	return &VerificationSubmitResponse{
		SessionID:    resp.Session.ID,
		AttemptID:    resp.Attempt.ID,
		Status:       resp.Status,
		ProviderName: resp.ProviderName,
		Message:      resp.Message,
	}, nil
}

// =======================
// Mutations - Address Verification
// =======================

// CreateAddressVerification creates a new address verification session
func (r *Resolver) CreateAddressVerification(ctx context.Context, input CreateAddressVerificationInput) (*domain.VerificationSession, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	addressData := &domain.AddressData{
		FullAddress: input.Address,
		City:        input.City,
		State:       input.State,
		PostalCode:  *input.ZipCode,
		Country:     input.Country,
	}

	req := service.CreateSessionRequest{
		UserID:  userUUID,
		Type:    domain.VerificationAddress,
		Tier:    domain.TierBasic,
		Country: input.Country,
		Data: domain.VerificationData{
			Address: addressData,
		},
	}

	session, err := r.verificationService.CreateSession(ctx, req)
	if err != nil {
		r.log.Error("failed to create address verification session", "error", err)
		return nil, err
	}

	r.log.Info("address verification session created", "session_id", session.ID, "user_id", v.UserID)
	return session, nil
}

// SubmitAddressVerification submits address verification proof
func (r *Resolver) SubmitAddressVerification(ctx context.Context, input SubmitAddressVerificationInput) (*VerificationSubmitResponse, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	documentBytes, err := base64.StdEncoding.DecodeString(input.ProofDocument)
	if err != nil {
		return nil, fmt.Errorf("invalid proof document encoding")
	}

	req := service.SubmitAddressVerificationRequest{
		SessionID:     input.SessionID,
		UserID:        userUUID,
		ProofDocument: documentBytes,
		DocumentType:  input.DocumentType,
	}

	resp, err := r.verificationService.SubmitAddressVerification(ctx, req)
	if err != nil {
		r.log.Error("failed to submit address verification", "session_id", input.SessionID, "error", err)
		return nil, err
	}

	return &VerificationSubmitResponse{
		SessionID:    resp.Session.ID,
		AttemptID:    resp.Attempt.ID,
		Status:       resp.Status,
		ProviderName: resp.ProviderName,
		Message:      resp.Message,
	}, nil
}

// =======================
// Mutations - Business Verification
// =======================

// CreateBusinessVerification creates a new business verification session
func (r *Resolver) CreateBusinessVerification(ctx context.Context, input CreateBusinessVerificationInput) (*domain.VerificationSession, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	businessData := &domain.BusinessData{
		BusinessName:       input.BusinessName,
		RegistrationNumber: *input.RegistrationNumber,
		Country:            input.Country,
		OwnerUserID:        &input.BusinessID,
	}

	req := service.CreateSessionRequest{
		UserID:  userUUID,
		Type:    domain.VerificationBusiness,
		Tier:    domain.TierBasic,
		Country: input.Country,
		Data: domain.VerificationData{
			Business: businessData,
		},
	}

	session, err := r.verificationService.CreateSession(ctx, req)
	if err != nil {
		r.log.Error("failed to create business verification session", "error", err)
		return nil, err
	}

	r.log.Info("business verification session created", "session_id", session.ID, "user_id", v.UserID, "business_id", input.BusinessID)
	return session, nil
}

// SubmitBusinessVerification submits business verification documents
func (r *Resolver) SubmitBusinessVerification(ctx context.Context, input SubmitBusinessVerificationInput) (*VerificationSubmitResponse, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userUUID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	registrationBytes, err := base64.StdEncoding.DecodeString(input.RegistrationDocument)
	if err != nil {
		return nil, fmt.Errorf("invalid registration document encoding")
	}

	var taxIDBytes *[]byte
	if input.TaxIDDocument != nil {
		decoded, err := base64.StdEncoding.DecodeString(*input.TaxIDDocument)
		if err != nil {
			return nil, fmt.Errorf("invalid tax ID document encoding")
		}
		taxIDBytes = &decoded
	}

	var licenseBytes *[]byte
	if input.BusinessLicenseDocument != nil {
		decoded, err := base64.StdEncoding.DecodeString(*input.BusinessLicenseDocument)
		if err != nil {
			return nil, fmt.Errorf("invalid business license document encoding")
		}
		licenseBytes = &decoded
	}

	req := service.SubmitBusinessVerificationRequest{
		SessionID:               input.SessionID,
		UserID:                  userUUID,
		RegistrationDocument:    registrationBytes,
		TaxIDDocument:           taxIDBytes,
		BusinessLicenseDocument: licenseBytes,
	}

	resp, err := r.verificationService.SubmitBusinessVerification(ctx, req)
	if err != nil {
		r.log.Error("failed to submit business verification", "session_id", input.SessionID, "error", err)
		return nil, err
	}

	return &VerificationSubmitResponse{
		SessionID:    resp.Session.ID,
		AttemptID:    resp.Attempt.ID,
		Status:       resp.Status,
		ProviderName: resp.ProviderName,
		Message:      resp.Message,
	}, nil
}

// =======================
// Response Types
// =======================

type OTPResponse struct {
	SessionID   uuid.UUID
	OTPSent     bool
	ExpiresAt   string
	SMSProvider string
	Message     string
}

type OTPVerificationResponse struct {
	SessionID         uuid.UUID
	Verified          bool
	RemainingAttempts int
	Message           string
	Session           *domain.VerificationSession
}

type VerificationSubmitResponse struct {
	SessionID    uuid.UUID
	AttemptID    uuid.UUID
	Status       string
	ProviderName string
	Message      string
}

// =======================
// Input Types
// =======================

type CreatePhoneVerificationInput struct {
	PhoneNumber string
	Country     string
}

type CreateIdentityVerificationInput struct {
	Tier        domain.VerificationTier
	Country     string
	FirstName   string
	LastName    string
	DateOfBirth *time.Time
}

type SubmitIdentityVerificationInput struct {
	SessionID      uuid.UUID
	SelfieImage    string
	DocumentImage  string
	DocumentType   domain.DocumentType
	DocumentNumber *string
}

type CreateAddressVerificationInput struct {
	Country string
	Address string
	City    string
	State   string
	ZipCode *string
}

type SubmitAddressVerificationInput struct {
	SessionID     uuid.UUID
	ProofDocument string
	DocumentType  string
}

type CreateBusinessVerificationInput struct {
	BusinessID         uuid.UUID
	BusinessName       string
	Country            string
	RegistrationNumber *string
}

type SubmitBusinessVerificationInput struct {
	SessionID               uuid.UUID
	RegistrationDocument    string
	TaxIDDocument           *string
	BusinessLicenseDocument *string
}
