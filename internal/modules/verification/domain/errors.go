package domain

import "errors"

var (
	// Session errors
	ErrSessionNotFound      = errors.New("verification session not found")
	ErrSessionAlreadyExists = errors.New("verification session already exists for this user")
	ErrSessionExpired       = errors.New("verification session has expired")
	ErrSessionAlreadyFinal  = errors.New("verification session already in final state")
	ErrInvalidSessionStatus = errors.New("invalid session status transition")
	ErrSessionTypeMismatch  = errors.New("session type mismatch")

	// Attempt errors
	ErrAttemptNotFound     = errors.New("verification attempt not found")
	ErrMaxAttemptsExceeded = errors.New("maximum verification attempts exceeded")
	ErrAttemptAlreadyFinal = errors.New("attempt already in final state")

	// Evidence errors
	ErrEvidenceNotFound        = errors.New("evidence not found")
	ErrEvidenceIntegrityFailed = errors.New("evidence integrity check failed")
	ErrInvalidEvidenceType     = errors.New("invalid evidence type")
	ErrMissingRequiredEvidence = errors.New("missing required evidence")

	// Validation errors
	ErrInvalidTier          = errors.New("invalid verification tier")
	ErrInvalidDocumentType  = errors.New("invalid document type")
	ErrInvalidCountryCode   = errors.New("invalid country code")
	ErrUnderAge             = errors.New("applicant is under minimum age requirement")
	ErrMissingApplicantInfo = errors.New("missing required applicant information")

	// Authorization errors
	ErrUnauthorized     = errors.New("unauthorized to access this verification")
	ErrPermissionDenied = errors.New("permission denied")

	// Rate limiting errors
	ErrRateLimitExceeded = errors.New("rate limit exceeded for verification requests")
)
