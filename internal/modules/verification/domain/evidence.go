package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Evidence represents a stored piece of verification evidence
// This is a domain wrapper around platform/evidence for verification context
type Evidence struct {
	ID         uuid.UUID
	SessionID  uuid.UUID
	AttemptID  *uuid.UUID // Optional: which attempt this evidence belongs to
	Type       EvidenceType
	Metadata   EvidenceMetadata

	// Verification
	Verified     bool       // Whether integrity check passed
	VerifiedAt   *time.Time

	// Lifecycle
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time // Soft delete for compliance
}

// NewEvidence creates a new evidence record
func NewEvidence(
	sessionID uuid.UUID,
	evidenceType EvidenceType,
	metadata EvidenceMetadata,
) (*Evidence, error) {
	if sessionID == uuid.Nil {
		return nil, fmt.Errorf("session ID is required")
	}
	if !evidenceType.IsValid() {
		return nil, ErrInvalidEvidenceType
	}
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid evidence metadata: %w", err)
	}

	now := time.Now()
	return &Evidence{
		ID:        uuid.New(),
		SessionID: sessionID,
		Type:      evidenceType,
		Metadata:  metadata,
		Verified:  false, // Will be verified after integrity check
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// MarkVerified marks the evidence as integrity-verified
func (e *Evidence) MarkVerified() {
	now := time.Now()
	e.Verified = true
	e.VerifiedAt = &now
	e.UpdatedAt = now
}

// MarkVerificationFailed marks the evidence integrity check as failed
func (e *Evidence) MarkVerificationFailed() {
	e.Verified = false
	e.VerifiedAt = nil
	e.UpdatedAt = time.Now()
}

// AssociateAttempt links this evidence to a specific attempt
func (e *Evidence) AssociateAttempt(attemptID uuid.UUID) error {
	if attemptID == uuid.Nil {
		return fmt.Errorf("invalid attempt ID")
	}
	e.AttemptID = &attemptID
	e.UpdatedAt = time.Now()
	return nil
}

// SoftDelete marks the evidence as deleted (for compliance/audit trail)
func (e *Evidence) SoftDelete() {
	now := time.Now()
	e.DeletedAt = &now
	e.UpdatedAt = now
}

// IsDeleted checks if the evidence has been soft-deleted
func (e *Evidence) IsDeleted() bool {
	return e.DeletedAt != nil
}

// GetURL returns the storage URL from metadata
func (e *Evidence) GetURL() string {
	return e.Metadata.URL
}

// GetHash returns the integrity hash from metadata
func (e *Evidence) GetHash() string {
	return e.Metadata.Hash
}
