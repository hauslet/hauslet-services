package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// SubmitListingVerification submits listing verification documents
func (s *verificationService) SubmitListingVerification(ctx context.Context, req SubmitListingVerificationRequest) (*SubmitVerificationResponse, error) {
	// 1. Get Session
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// 2. Validate Session State
	if session.Status != domain.SessionPending && session.Status != domain.SessionInProgress {
		return nil, domain.ErrInvalidSessionStatus
	}
	if session.Type != domain.VerificationListing {
		return nil, domain.ErrSessionTypeMismatch
	}

	// 3. Upload Document Evidence
	// Detect MIME type
	mimeType := http.DetectContentType(req.ProofDocument)

	// Determine evidence type based on string input
	var evidenceType domain.EvidenceType
	switch req.DocumentType {
	case "title_deed":
		evidenceType = domain.EvidenceTitleDeed
	case "property_tax":
		evidenceType = domain.EvidencePropertyTax
	case "geo_tagged_photo":
		evidenceType = domain.EvidenceGeoTaggedPhoto
	default:
		// Fallback or error? For now accept as generic document if unknown?
		// Better to restrict.
		// Actually Enums.go has these constants.
		return nil, fmt.Errorf("invalid document type: %s", req.DocumentType)
	}

	evidenceReq := UploadEvidenceRequest{
		SessionID: session.ID,
		UserID:    req.UserID,
		Type:      evidenceType,
		Data:      req.ProofDocument,
		MimeType:  mimeType,
		IPAddress: req.IPAddress,
	}

	_, err = s.UploadEvidence(ctx, evidenceReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload evidence: %w", err)
	}

	// 4. Update Session Data
	listingData, err := session.GetListingData()
	if err != nil {
		// Should not happen if created correctly, but handle it
		return nil, err
	}
	listingData.ProofType = req.DocumentType

	// Update session data in repo
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session data: %w", err)
	}

	// 5. Create Attempt (Manual Review usually for listings)
	attempt := &domain.VerificationAttempt{
		ID:           uuid.New(),
		SessionID:    session.ID,
		Status:       domain.AttemptPending, // Manual review needed
		ProviderName: "manual_review",       // or "hauslet_admin"
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		return nil, fmt.Errorf("failed to create attempt: %w", err)
	}

	// Update session status to pending (or in_progress)
	// If it was pending, it remains pending/in_progress until admin reviews.
	session.Status = domain.SessionInProgress
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session status: %w", err)
	}

	// 6. Notify Admin ??? (Future work)

	return &SubmitVerificationResponse{
		Attempt:      attempt,
		Session:      session,
		ProviderName: "manual_review",
		Status:       string(domain.AttemptPending),
		Message:      "Verification submitted for review",
	}, nil
}
