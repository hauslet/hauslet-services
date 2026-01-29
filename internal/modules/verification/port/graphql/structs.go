package graphql

import (
	"hauslet/internal/modules/verification/domain"
	"time"

	"github.com/google/uuid"
)

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

type CreateListingVerificationInput struct {
	ListingID uuid.UUID
	Tier      domain.VerificationTier
	Country   string
}

type SubmitListingVerificationInput struct {
	SessionID     uuid.UUID
	ProofDocument string
	DocumentType  string
}
