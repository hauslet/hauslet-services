package kyc

import (
	"encoding/json"
	"errors"
	"time"
)

// DocumentType represents the type of identity document
type DocumentType string

const (
	DocumentPassport        DocumentType = "passport"
	DocumentDriversLicense  DocumentType = "drivers_license"
	DocumentNationalID      DocumentType = "national_id"
	DocumentResidencePermit DocumentType = "residence_permit"
	// Nigeria-specific document types
	DocumentVotersCard DocumentType = "voters_card"
	DocumentNIN        DocumentType = "NIN" // National Identification Number
	DocumentBVN        DocumentType = "BVN" // Bank Verification Number
	DocumentVIN        DocumentType = "VIN" // Voter Identification Number
	DocumentCAC        DocumentType = "CAC" // Corporate Affairs Commission (Business)
)

// IsValid checks if document type is supported
func (d DocumentType) IsValid() bool {
	switch d {
	case DocumentPassport, DocumentDriversLicense, DocumentNationalID, DocumentVotersCard,
		DocumentNIN, DocumentBVN, DocumentVIN, DocumentResidencePermit, DocumentCAC:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (d DocumentType) String() string {
	return string(d)
}

// VerificationStatus represents the current state of verification
type VerificationStatus string

const (
	StatusPending     VerificationStatus = "pending"
	StatusInReview    VerificationStatus = "in_review"
	StatusApproved    VerificationStatus = "approved"
	StatusRejected    VerificationStatus = "rejected"
	StatusExpired     VerificationStatus = "expired"
	StatusResubmitted VerificationStatus = "resubmitted"
)

// String returns the string representation
func (s VerificationStatus) String() string {
	return string(s)
}

// FailureCode represents machine-readable failure reasons
type FailureCode string

const (
	FailureImageBlurry     FailureCode = "image_blurry"
	FailureNoFaceDetected  FailureCode = "no_face_detected"
	FailureDocExpired      FailureCode = "document_expired"
	FailureFaceMismatch    FailureCode = "face_mismatch"
	FailureProviderTimeout FailureCode = "provider_timeout"
	FailureProviderError   FailureCode = "provider_error"
	FailureInvalidDocument FailureCode = "invalid_document"
	FailurePoorQuality     FailureCode = "poor_image_quality"
	FailureUnderage        FailureCode = "underage"
	FailureSuspectedFraud  FailureCode = "suspected_fraud"
	FailureDocNotFound     FailureCode = "document_not_found"
)

// String returns the string representation
func (f FailureCode) String() string {
	return string(f)
}

// VerificationRequest is the unified input for KYC verification
type VerificationRequest struct {
	UserID         string            // Internal user identifier
	Country        string            // ISO 3166-1 alpha-2 country code (e.g., "NG", "US")
	DocumentType   DocumentType      // Type of document being verified
	SelfieImage    []byte            // Base64 encoded selfie/liveness image
	DocumentImage  []byte            // Base64 encoded document image (optional for some flows)
	FirstName      string            // Expected first name on document
	LastName       string            // Expected last name on document
	DateOfBirth    *time.Time        // Expected date of birth (optional)
	DocumentNumber string            // Document number for additional validation (optional)
	Metadata       map[string]string // Additional context (session_id, attempt_number, etc.)
}

// Validate checks if verification request has required fields.
// Supports two modes:
//   - Photo ID mode: requires SelfieImage + DocumentImage + name fields
//   - Govt Data Lookup mode: requires only DocumentNumber
func (r *VerificationRequest) Validate() error {
	if r.UserID == "" {
		return errors.New("user_id is required")
	}
	if r.Country == "" {
		return errors.New("country is required")
	}
	if !r.DocumentType.IsValid() {
		return errors.New("invalid document type")
	}

	// Determine mode and validate accordingly
	hasImages := len(r.SelfieImage) > 0 || len(r.DocumentImage) > 0
	hasDocNumber := r.DocumentNumber != ""

	if !hasImages && !hasDocNumber {
		return errors.New("either (selfie_image + document_image) or document_number is required")
	}

	// Photo ID mode: needs both images and name fields
	if hasImages {
		if len(r.SelfieImage) == 0 {
			return errors.New("selfie image is required for photo ID verification")
		}
		if len(r.DocumentImage) == 0 {
			return errors.New("document image is required for photo ID verification")
		}
		if r.FirstName == "" {
			return errors.New("first_name is required for photo ID verification")
		}
		if r.LastName == "" {
			return errors.New("last_name is required for photo ID verification")
		}
	}

	// Govt Data Lookup mode: document_number is sufficient (already checked above)
	return nil
}

// VerificationResponse contains the result of a verification submission
type VerificationResponse struct {
	Success     bool               // Whether the request was processed successfully
	ProviderRef string             // External provider's transaction ID
	Provider    string             // Provider name (dojah, veriff)
	Status      VerificationStatus // Current verification status

	// Quality & Cost Tracking
	QualityScore   *float64      // 0-1 confidence score from provider
	EstimatedCost  float64       // Estimated cost in USD
	ProcessingTime time.Duration // Time taken to process request

	// Additional Data
	ExtractedData map[string]string // Extracted document data (name, DOB, etc.)
	Message       string            // Status message for client

	// Failure details (populated on rejection)
	FailureReason *string      // Human-readable failure reason
	FailureCode   *FailureCode // Machine-readable failure code
}

// WebhookEvent represents a normalized webhook event from providers
type WebhookEvent struct {
	ProviderRef   string             // External provider's transaction ID
	Provider      string             // Provider name
	Status        VerificationStatus // Final verification status
	FailureReason *string            // Human-readable failure reason (if rejected)
	FailureCode   *FailureCode       // Machine-readable failure code (if rejected)
	ExtractedData map[string]string  // Extracted document data
	RawPayload    json.RawMessage    // Original provider payload for debugging
	ReceivedAt    time.Time          // When webhook was received
}

// ProviderStats contains provider performance metrics
type ProviderStats struct {
	Name               string
	SuccessRate        float64       // Last 24 hours
	AvgLatency         time.Duration // Average response time
	AvgCost            float64       // Average cost per verification
	SupportedCountries []string      // List of supported country codes
	LastUpdated        time.Time
}

// =======================
// Business Verification (KYB) Types
// =======================

// BusinessVerificationResponse contains the result of a CAC/KYB lookup
type BusinessVerificationResponse struct {
	Found              bool           `json:"found"`
	CompanyName        string         `json:"company_name,omitempty"`
	RCNumber           string         `json:"rc_number,omitempty"`
	CompanyType        string         `json:"company_type,omitempty"`
	Address            string         `json:"address,omitempty"`
	Status             string         `json:"status,omitempty"`
	DateOfRegistration string         `json:"date_of_registration,omitempty"`
	State              string         `json:"state,omitempty"`
	City               string         `json:"city,omitempty"`
	LGA                string         `json:"lga,omitempty"`
	Email              string         `json:"email,omitempty"`
	Affiliates         []CACAffiliate `json:"affiliates,omitempty"`
	Message            string         `json:"message"`
}

// CACEntity represents the entity returned from Dojah's CAC advance endpoint
type CACEntity struct {
	CompanyName        string         `json:"company_name"`
	RCNumber           string         `json:"rc_number"`
	TypeOfCompany      string         `json:"type_of_company"`
	Address            string         `json:"address"`
	Status             string         `json:"status"`
	DateOfRegistration string         `json:"date_of_registration"`
	State              string         `json:"state"`
	City               string         `json:"city"`
	LGA                string         `json:"lga"`
	Email              string         `json:"email"`
	NatureOfBusiness   *string        `json:"nature_of_business"`
	ShareCapital       *string        `json:"share_capital"`
	Affiliates         []CACAffiliate `json:"affiliates"`
}

// CACAffiliate represents a director/proprietor from the CAC lookup
type CACAffiliate struct {
	FirstName             string  `json:"first_name"`
	LastName              string  `json:"last_name"`
	Email                 string  `json:"email"`
	Address               string  `json:"address"`
	State                 string  `json:"state"`
	City                  string  `json:"city"`
	LGA                   string  `json:"lga"`
	Occupation            *string `json:"occupation"`
	PhoneNumber           string  `json:"phone_number"`
	Gender                string  `json:"gender"`
	DateOfBirth           string  `json:"date_of_birth"`
	Nationality           string  `json:"nationality"`
	AffiliateType         string  `json:"affiliate_type"`          // PROPRIETOR, DIRECTOR, etc.
	AffiliateCategoryType string  `json:"affiliate_category_type"` // individual_proprietor, etc.
	Country               string  `json:"country"`
}
