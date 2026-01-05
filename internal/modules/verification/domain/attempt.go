package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// VerificationAttempt represents a single KYC submission to a provider
type VerificationAttempt struct {
	ID               uuid.UUID
	SessionID        uuid.UUID
	ProviderName     string // "dojah" or "veriff"
	Status           AttemptStatus
	Result           *VerificationResult
	EvidenceIDs      []uuid.UUID // References to uploaded evidence

	// Timing
	SubmittedAt  time.Time
	CompletedAt  *time.Time
	ProcessingTime *time.Duration // How long provider took

	// Provider tracking
	ProviderSessionID *string // Provider's session/reference ID
	WebhookReceived   bool
	WebhookReceivedAt *time.Time

	// Metadata
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewVerificationAttempt creates a new verification attempt
func NewVerificationAttempt(sessionID uuid.UUID, providerName string) (*VerificationAttempt, error) {
	if sessionID == uuid.Nil {
		return nil, fmt.Errorf("session ID is required")
	}
	if providerName == "" {
		return nil, fmt.Errorf("provider name is required")
	}

	now := time.Now()
	return &VerificationAttempt{
		ID:           uuid.New(),
		SessionID:    sessionID,
		ProviderName: providerName,
		Status:       AttemptPending,
		EvidenceIDs:  []uuid.UUID{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// Submit marks the attempt as submitted to the provider
func (a *VerificationAttempt) Submit(providerSessionID string) error {
	if a.Status != AttemptPending {
		return fmt.Errorf("attempt not in pending state")
	}

	now := time.Now()
	a.Status = AttemptProcessing
	a.SubmittedAt = now
	a.ProviderSessionID = &providerSessionID
	a.UpdatedAt = now

	return nil
}

// Complete marks the attempt as completed with a result
func (a *VerificationAttempt) Complete(result VerificationResult) error {
	if a.Status.IsFinal() {
		return ErrAttemptAlreadyFinal
	}

	now := time.Now()
	a.Result = &result
	a.CompletedAt = &now
	a.UpdatedAt = now

	if result.Success {
		a.Status = AttemptSuccess
	} else {
		a.Status = AttemptFailed
	}

	// Calculate processing time
	if !a.SubmittedAt.IsZero() {
		duration := now.Sub(a.SubmittedAt)
		a.ProcessingTime = &duration
	}

	return nil
}

// MarkWebhookReceived records that a webhook was received for this attempt
func (a *VerificationAttempt) MarkWebhookReceived() {
	now := time.Now()
	a.WebhookReceived = true
	a.WebhookReceivedAt = &now
	a.UpdatedAt = now
}

// AddEvidence associates evidence with this attempt
func (a *VerificationAttempt) AddEvidence(evidenceID uuid.UUID) error {
	if evidenceID == uuid.Nil {
		return fmt.Errorf("invalid evidence ID")
	}

	// Check for duplicates
	for _, id := range a.EvidenceIDs {
		if id == evidenceID {
			return nil // Already added
		}
	}

	a.EvidenceIDs = append(a.EvidenceIDs, evidenceID)
	a.UpdatedAt = time.Now()

	return nil
}

// HasEvidence checks if evidence is attached
func (a *VerificationAttempt) HasEvidence() bool {
	return len(a.EvidenceIDs) > 0
}

// IsSuccess returns true if the attempt succeeded
func (a *VerificationAttempt) IsSuccess() bool {
	return a.Status == AttemptSuccess && a.Result != nil && a.Result.Success
}

// IsFailed returns true if the attempt failed
func (a *VerificationAttempt) IsFailed() bool {
	return a.Status == AttemptFailed
}

// GetRejectionReason returns the rejection reason if the attempt failed
func (a *VerificationAttempt) GetRejectionReason() *RejectionReason {
	if a.Result != nil {
		return a.Result.RejectionReason
	}
	return nil
}
