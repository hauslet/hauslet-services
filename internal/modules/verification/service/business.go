package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// =======================
// Business Verification
// =======================

func (s *verificationService) SubmitBusinessVerification(ctx context.Context, req SubmitBusinessVerificationRequest) (*SubmitVerificationResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Ensure session is business verification type
	if session.Type != domain.VerificationBusiness {
		return nil, fmt.Errorf("session is not business verification type")
	}

	// Check if session can accept new attempt
	if err := session.CanSubmitAttempt(); err != nil {
		return nil, err
	}

	// Upload registration document
	evidenceIDs := []uuid.UUID{}
	regEvidence, err := s.UploadEvidence(ctx, UploadEvidenceRequest{
		SessionID: session.ID,
		UserID:    req.UserID,
		Type:      domain.EvidenceBusinessRegistration,
		Data:      req.RegistrationDocument,
		MimeType:  "application/pdf",
		IPAddress: req.IPAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload registration document: %w", err)
	}
	evidenceIDs = append(evidenceIDs, regEvidence.ID)

	// Upload tax ID document if provided
	if req.TaxIDDocument != nil {
		taxEvidence, err := s.UploadEvidence(ctx, UploadEvidenceRequest{
			SessionID: session.ID,
			UserID:    req.UserID,
			Type:      domain.EvidenceBusinessTaxID,
			Data:      *req.TaxIDDocument,
			MimeType:  "application/pdf",
			IPAddress: req.IPAddress,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload tax ID document: %w", err)
		}
		evidenceIDs = append(evidenceIDs, taxEvidence.ID)
	}

	// Upload business license if provided
	if req.BusinessLicenseDocument != nil {
		licEvidence, err := s.UploadEvidence(ctx, UploadEvidenceRequest{
			SessionID: session.ID,
			UserID:    req.UserID,
			Type:      domain.EvidenceBusinessLicense,
			Data:      *req.BusinessLicenseDocument,
			MimeType:  "application/pdf",
			IPAddress: req.IPAddress,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload business license: %w", err)
		}
		evidenceIDs = append(evidenceIDs, licEvidence.ID)
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

	s.logger.Info("business verification submitted",
		"session_id", session.ID,
		"attempt_id", attempt.ID,
		"evidence_count", len(evidenceIDs),
	)

	return &SubmitVerificationResponse{
		Attempt:      attempt,
		Session:      session,
		ProviderName: "manual_review",
		Status:       "pending",
		Message:      "Business verification submitted for review",
	}, nil
}
