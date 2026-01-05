package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// VerificationData is a union type for type-specific verification data
// Only one field should be populated based on VerificationType
type VerificationData struct {
	Identity *IdentityData `json:"identity,omitempty"`
	Phone    *PhoneData    `json:"phone,omitempty"`
	Address  *AddressData  `json:"address,omitempty"`
	Business *BusinessData `json:"business,omitempty"`
}

// ValidateForType ensures data matches the verification type
func (v *VerificationData) ValidateForType(vType VerificationType) error {
	switch vType {
	case VerificationIdentity:
		if v.Identity == nil {
			return fmt.Errorf("identity data required for identity verification")
		}
		return v.Identity.Validate()
	case VerificationPhone:
		if v.Phone == nil {
			return fmt.Errorf("phone data required for phone verification")
		}
		return v.Phone.Validate()
	case VerificationAddress:
		if v.Address == nil {
			return fmt.Errorf("address data required for address verification")
		}
		return v.Address.Validate()
	case VerificationBusiness:
		if v.Business == nil {
			return fmt.Errorf("business data required for business verification")
		}
		return v.Business.Validate()
	default:
		return fmt.Errorf("unknown verification type: %s", vType)
	}
}

// IdentityData contains identity verification specific information
type IdentityData struct {
	ApplicantInfo ApplicantInfo `json:"applicant_info"`
	DocumentInfo  DocumentInfo  `json:"document_info"`
}

func (i *IdentityData) Validate() error {
	if err := i.ApplicantInfo.Validate(); err != nil {
		return fmt.Errorf("invalid applicant info: %w", err)
	}
	if err := i.DocumentInfo.Validate(); err != nil {
		return fmt.Errorf("invalid document info: %w", err)
	}
	return nil
}

// PhoneData contains phone verification specific information
// Note: OTP code is stored in Redis with TTL, not in this struct
type PhoneData struct {
	PhoneNumber     string     `json:"phone_number"`      // E.164 format
	CountryCode     string     `json:"country_code"`      // ISO 3166-1 alpha-2
	OTPGeneratedAt  *time.Time `json:"otp_generated_at,omitempty"`
	OTPExpiresAt    *time.Time `json:"otp_expires_at,omitempty"`
	OTPAttempts     int        `json:"otp_attempts"`       // Number of failed OTP verification attempts
	MaxOTPAttempts  int        `json:"max_otp_attempts"`   // Maximum allowed attempts
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	SMSProvider     *string    `json:"sms_provider,omitempty"` // Which SMS provider was used
}

func (p *PhoneData) Validate() error {
	if p.PhoneNumber == "" {
		return fmt.Errorf("phone number is required")
	}
	if p.CountryCode == "" {
		return fmt.Errorf("country code is required")
	}
	if len(p.CountryCode) != 2 {
		return fmt.Errorf("country code must be 2 characters (ISO 3166-1 alpha-2)")
	}
	if p.MaxOTPAttempts <= 0 {
		p.MaxOTPAttempts = 3 // Default
	}
	return nil
}

// CanRetryOTP checks if more OTP attempts are allowed
func (p *PhoneData) CanRetryOTP() bool {
	return p.OTPAttempts < p.MaxOTPAttempts
}

// IsOTPExpired checks if the OTP has expired
func (p *PhoneData) IsOTPExpired() bool {
	if p.OTPExpiresAt == nil {
		return true
	}
	return time.Now().After(*p.OTPExpiresAt)
}

// AddressData contains address verification specific information
type AddressData struct {
	FullAddress    string     `json:"full_address"`
	Street         string     `json:"street"`
	City           string     `json:"city"`
	State          string     `json:"state"`
	PostalCode     string     `json:"postal_code"`
	Country        string     `json:"country"`         // ISO 3166-1 alpha-2
	DocumentType   string     `json:"document_type"`   // utility_bill, bank_statement, lease, etc.
	IssueDate      *time.Time `json:"issue_date,omitempty"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	MatchScore     *float64   `json:"match_score,omitempty"` // Confidence score from verification service
}

func (a *AddressData) Validate() error {
	if a.FullAddress == "" && a.Street == "" {
		return fmt.Errorf("address information is required")
	}
	if a.Country == "" {
		return fmt.Errorf("country is required")
	}
	if len(a.Country) != 2 {
		return fmt.Errorf("country must be 2 characters (ISO 3166-1 alpha-2)")
	}
	if a.DocumentType == "" {
		return fmt.Errorf("document type is required")
	}
	return nil
}

// BusinessData contains business verification specific information
type BusinessData struct {
	BusinessName        string     `json:"business_name"`
	RegistrationNumber  string     `json:"registration_number"`
	TaxID               *string    `json:"tax_id,omitempty"`
	BusinessType        string     `json:"business_type"` // llc, corporation, sole_proprietorship, etc.
	IncorporationDate   *time.Time `json:"incorporation_date,omitempty"`
	Country             string     `json:"country"` // ISO 3166-1 alpha-2
	BusinessAddress     Address    `json:"business_address"`

	// Ownership/Directors
	OwnerUserID         *uuid.UUID `json:"owner_user_id,omitempty"`
	BeneficialOwners    []string   `json:"beneficial_owners,omitempty"` // Names
	Directors           []string   `json:"directors,omitempty"`         // Names

	VerifiedAt          *time.Time `json:"verified_at,omitempty"`
	VerificationProvider *string   `json:"verification_provider,omitempty"`
}

func (b *BusinessData) Validate() error {
	if b.BusinessName == "" {
		return fmt.Errorf("business name is required")
	}
	if b.RegistrationNumber == "" {
		return fmt.Errorf("registration number is required")
	}
	if b.BusinessType == "" {
		return fmt.Errorf("business type is required")
	}
	if b.Country == "" {
		return fmt.Errorf("country is required")
	}
	if len(b.Country) != 2 {
		return fmt.Errorf("country must be 2 characters (ISO 3166-1 alpha-2)")
	}
	if err := b.BusinessAddress.Validate(); err != nil {
		return fmt.Errorf("invalid business address: %w", err)
	}
	return nil
}
