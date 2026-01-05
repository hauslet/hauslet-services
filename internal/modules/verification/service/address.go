package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// =======================
// Address Verification
// =======================

func (s *verificationService) SubmitAddressVerification(ctx context.Context, req SubmitAddressVerificationRequest) (*SubmitVerificationResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Ensure session is address verification type
	if session.Type != domain.VerificationAddress {
		return nil, fmt.Errorf("session is not address verification type")
	}

	// Check if session can accept new attempt
	if err := session.CanSubmitAttempt(); err != nil {
		return nil, err
	}

	// Upload proof document as evidence
	evidenceReq := UploadEvidenceRequest{
		SessionID: session.ID,
		UserID:    req.UserID,
		Type:      s.mapAddressDocTypeToEvidence(req.DocumentType),
		Data:      req.ProofDocument,
		MimeType:  "application/pdf", // Default to PDF, could be enhanced
		IPAddress: req.IPAddress,
	}

	evidence, err := s.UploadEvidence(ctx, evidenceReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload address proof: %w", err)
	}

	// Create attempt record
	attempt, err := domain.NewVerificationAttempt(session.ID, "manual_review")
	if err != nil {
		return nil, err
	}

	// Add evidence to attempt
	attempt.EvidenceIDs = []uuid.UUID{evidence.ID}

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

	s.logger.Info("address verification submitted",
		"session_id", session.ID,
		"attempt_id", attempt.ID,
		"evidence_id", evidence.ID,
	)

	return &SubmitVerificationResponse{
		Attempt:      attempt,
		Session:      session,
		ProviderName: "manual_review",
		Status:       "pending",
		Message:      "Address verification submitted for review",
	}, nil
}
