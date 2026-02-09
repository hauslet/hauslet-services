package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// ManualReviewSubmission contains the common parameters for submitting
// a verification that requires manual review (address, business, listing).
type ManualReviewSubmission struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
	Evidence  []ManualReviewEvidence
	IPAddress *string
}

// ManualReviewEvidence represents a single evidence item to upload.
type ManualReviewEvidence struct {
	Type     domain.EvidenceType
	Data     []byte
	MimeType string
}

// submitManualReview is the shared implementation for verification types
// that require manual review: address, business, and listing.
// It handles: evidence upload → attempt creation → session state transition.
func (s *verificationService) submitManualReview(
	ctx context.Context,
	session *domain.VerificationSession,
	userID uuid.UUID,
	evidenceItems []ManualReviewEvidence,
	ipAddress *string,
) (*SubmitVerificationResponse, error) {
	// Check if session can accept new attempt
	if err := session.CanSubmitAttempt(); err != nil {
		return nil, err
	}

	// Upload all evidence items
	evidenceIDs := make([]uuid.UUID, 0, len(evidenceItems))
	for _, item := range evidenceItems {
		ev, err := s.UploadEvidence(ctx, UploadEvidenceRequest{
			SessionID: session.ID,
			UserID:    userID,
			Type:      item.Type,
			Data:      item.Data,
			MimeType:  item.MimeType,
			IPAddress: ipAddress,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload %s: %w", item.Type, err)
		}
		evidenceIDs = append(evidenceIDs, ev.ID)
	}

	// Create attempt record
	attempt, err := domain.NewVerificationAttempt(session.ID, "manual_review")
	if err != nil {
		return nil, err
	}

	// Add evidence to attempt
	attempt.EvidenceIDs = evidenceIDs

	// Update session state
	if err := session.StartAttempt(); err != nil {
		return nil, err
	}

	// Mark attempt as pending (requires manual review)
	attempt.Status = domain.AttemptPending

	// Save attempt
	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		return nil, fmt.Errorf("failed to save attempt: %w", err)
	}

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	s.logger.Info("manual review verification submitted",
		"session_id", session.ID,
		"attempt_id", attempt.ID,
		"type", session.Type,
		"evidence_count", len(evidenceIDs),
	)

	return &SubmitVerificationResponse{
		Attempt:      attempt,
		Session:      session,
		ProviderName: "manual_review",
		Status:       "pending",
		Message:      fmt.Sprintf("%s verification submitted for review", session.Type),
	}, nil
}
