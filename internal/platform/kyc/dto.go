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
)

// IsValid checks if document type is supported
func (d DocumentType) IsValid() bool {
	switch d {
	case DocumentPassport, DocumentDriversLicense, DocumentNationalID, DocumentVotersCard,
		DocumentNIN, DocumentBVN, DocumentVIN:
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

// Validate checks if verification request has required fields
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
	if len(r.SelfieImage) == 0 {
		return errors.New("selfie image is required")
	}
	if r.FirstName == "" {
		return errors.New("first_name is required")
	}
	if r.LastName == "" {
		return errors.New("last_name is required")
	}
	return nil
}

// VerificationResponse contains the result of a verification submission
type VerificationResponse struct {
	Success       bool               // Whether the request was processed successfully
	ProviderRef   string             // External provider's transaction ID
	Provider      string             // Provider name (dojah, veriff)
	Status        VerificationStatus // Current verification status
	FailureReason *string            // Human-readable failure reason
	FailureCode   *FailureCode       // Machine-readable failure code

	// Quality & Cost Tracking
	QualityScore   *float64      // 0-1 confidence score from provider
	EstimatedCost  float64       // Estimated cost in USD
	ProcessingTime time.Duration // Time taken to process request

	// Additional Data
	ExtractedData map[string]string // Extracted document data (name, DOB, etc.)
	Message       string            // Status message for client
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
