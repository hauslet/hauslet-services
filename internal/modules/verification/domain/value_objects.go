package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ApplicantInfo contains the personal information submitted for verification
type ApplicantInfo struct {
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Nationality string // ISO 3166-1 alpha-2 country code
	Email       *string
	PhoneNumber *string
	Address     *Address
}

// Validate checks if the applicant info is complete and valid
func (a *ApplicantInfo) Validate() error {
	if strings.TrimSpace(a.FirstName) == "" {
		return fmt.Errorf("first name is required")
	}
	if strings.TrimSpace(a.LastName) == "" {
		return fmt.Errorf("last name is required")
	}
	if a.DateOfBirth.IsZero() {
		return fmt.Errorf("date of birth is required")
	}
	if a.Nationality == "" || len(a.Nationality) != 2 {
		return ErrInvalidCountryCode
	}

	// Check minimum age (18 years)
	minAge := time.Now().AddDate(-18, 0, 0)
	if a.DateOfBirth.After(minAge) {
		return ErrUnderAge
	}

	return nil
}

// FullName returns the complete name
func (a *ApplicantInfo) FullName() string {
	return fmt.Sprintf("%s %s", a.FirstName, a.LastName)
}

// Age calculates the current age
func (a *ApplicantInfo) Age() int {
	now := time.Now()
	age := now.Year() - a.DateOfBirth.Year()
	if now.YearDay() < a.DateOfBirth.YearDay() {
		age--
	}
	return age
}

// Address represents a physical address
type Address struct {
	Line1      string
	Line2      *string
	City       string
	State      *string
	PostalCode string
	Country    string // ISO 3166-1 alpha-2
}

// Validate checks if address has required fields
func (a *Address) Validate() error {
	if strings.TrimSpace(a.Line1) == "" {
		return fmt.Errorf("address line 1 is required")
	}
	if strings.TrimSpace(a.City) == "" {
		return fmt.Errorf("city is required")
	}
	if a.Country == "" || len(a.Country) != 2 {
		return ErrInvalidCountryCode
	}
	return nil
}

// VerificationResult contains the outcome of a verification attempt
type VerificationResult struct {
	Success         bool
	ProviderName    string
	ProviderRefID   string
	Score           *float64 // Confidence score (0-100)
	RejectionReason *RejectionReason
	RejectionNotes  *string
	ProcessedAt     time.Time
	RawResponse     map[string]any // Provider's raw response
}

// IsRejected returns true if the verification was rejected
func (r *VerificationResult) IsRejected() bool {
	return !r.Success && r.RejectionReason != nil
}

// HasHighConfidence returns true if score >= 80
func (r *VerificationResult) HasHighConfidence() bool {
	if r.Score == nil {
		return false
	}
	return *r.Score >= 80.0
}

// DocumentInfo contains metadata about the submitted document
type DocumentInfo struct {
	Type           DocumentType
	Number         *string    // Document number (if applicable)
	IssuingCountry string     // ISO 3166-1 alpha-2
	IssueDate      *time.Time
	ExpiryDate     *time.Time
}

// IsExpired checks if the document has expired
func (d *DocumentInfo) IsExpired() bool {
	if d.ExpiryDate == nil {
		return false
	}
	return d.ExpiryDate.Before(time.Now())
}

// Validate checks document info validity
func (d *DocumentInfo) Validate() error {
	if !d.Type.IsValid() {
		return ErrInvalidDocumentType
	}
	if d.IssuingCountry == "" || len(d.IssuingCountry) != 2 {
		return ErrInvalidCountryCode
	}
	if d.IsExpired() {
		return ErrSessionExpired
	}
	return nil
}

// EvidenceMetadata contains information about stored evidence
type EvidenceMetadata struct {
	EvidenceID   uuid.UUID
	Type         EvidenceType
	URL          string
	Hash         string
	Size         int64
	MimeType     string
	UploadedAt   time.Time
	UploadedByIP *string
}

// Validate checks evidence metadata validity
func (e *EvidenceMetadata) Validate() error {
	if e.EvidenceID == uuid.Nil {
		return fmt.Errorf("evidence ID is required")
	}
	if !e.Type.IsValid() {
		return ErrInvalidEvidenceType
	}
	if e.URL == "" {
		return fmt.Errorf("evidence URL is required")
	}
	if e.Hash == "" {
		return fmt.Errorf("evidence hash is required")
	}
	return nil
}
