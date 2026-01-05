package verification

import (
	"fmt"
	"math"
	"time"
)

const VerificationSubmissionJobType = "verification_submission"

// VerificationSubmissionJob represents a job to process a verification submission
type VerificationSubmissionJob struct {
	SessionID    string    `json:"session_id"`
	EvidenceURL  string    `json:"evidence_url"`
	EvidenceHash string    `json:"evidence_hash"`
	Priority     string    `json:"priority"`      // high, normal, low
	RetryAttempt int       `json:"retry_attempt"` // 0 for first attempt
	ScheduledFor time.Time `json:"scheduled_for,omitempty"`
	TraceID      string    `json:"trace_id,omitempty"`
}

// Type returns the job type identifier
func (j *VerificationSubmissionJob) Type() string {
	return VerificationSubmissionJobType
}

// Validate checks if the job is valid
func (j *VerificationSubmissionJob) Validate() error {
	if j.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if j.EvidenceURL == "" {
		return fmt.Errorf("evidence_url is required")
	}
	if j.EvidenceHash == "" {
		return fmt.Errorf("evidence_hash is required")
	}
	if j.Priority == "" {
		j.Priority = "normal"
	}
	if j.Priority != "high" && j.Priority != "normal" && j.Priority != "low" {
		return fmt.Errorf("priority must be one of: high, normal, low")
	}
	return nil
}

// CalculateDelay calculates exponential backoff with jitter for retries
func (j *VerificationSubmissionJob) CalculateDelay() time.Duration {
	if j.RetryAttempt == 0 {
		return 0
	}
	// Exponential backoff: 2^n minutes + random jitter up to 30 seconds
	baseDelay := time.Duration(math.Pow(2, float64(j.RetryAttempt))) * time.Minute
	jitter := time.Duration(time.Now().UnixNano()%30) * time.Second
	return baseDelay + jitter
}
