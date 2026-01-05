package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/platform/evidence"
	verificationJob "hauslet/internal/queue/jobs/verification"

	"github.com/google/uuid"
)

// =======================
// Evidence Management
// =======================

func (s *verificationService) UploadEvidence(ctx context.Context, req UploadEvidenceRequest) (*domain.Evidence, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Upload to evidence store
	uploadReq := evidence.UploadRequest{
		SessionID: req.SessionID,
		Data:      req.Data,
		MimeType:  req.MimeType,
		Metadata: map[string]string{
			"user_id": req.UserID.String(),
			"type":    req.Type.String(),
		},
	}
	if req.IPAddress != nil {
		uploadReq.Metadata["uploaded_by_ip"] = *req.IPAddress
	}

	evidenceResp, err := s.evidenceStore.Upload(ctx, uploadReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload evidence: %w", err)
	}

	// Create evidence domain entity
	metadata := domain.EvidenceMetadata{
		EvidenceID:   uuid.New(),
		Type:         req.Type,
		URL:          evidenceResp.URL,
		Hash:         evidenceResp.Hash,
		Size:         evidenceResp.Size,
		MimeType:     evidenceResp.MimeType,
		UploadedAt:   evidenceResp.UploadedAt,
		UploadedByIP: req.IPAddress,
	}

	ev, err := domain.NewEvidence(req.SessionID, req.Type, metadata)
	if err != nil {
		return nil, err
	}

	// Verify integrity
	integrityCheck, err := s.evidenceStore.VerifyIntegrity(ctx, evidenceResp.URL, evidenceResp.Hash)
	if err == nil && integrityCheck.Verified {
		ev.MarkVerified()
	}

	// Save to repository
	if err := s.repo.CreateEvidence(ctx, ev); err != nil {
		return nil, fmt.Errorf("failed to save evidence: %w", err)
	}

	s.logger.Info("evidence uploaded",
		"evidence_id", ev.ID,
		"session_id", session.ID,
		"type", req.Type,
		"size", evidenceResp.Size,
	)

	// Enqueue verification submission job for async processing
	if session.Type == domain.VerificationIdentity && s.queueClient != nil {
		submissionJob := verificationJob.VerificationSubmissionJob{
			SessionID:    session.ID.String(),
			EvidenceURL:  evidenceResp.URL,
			EvidenceHash: evidenceResp.Hash,
			Priority:     s.getPriorityForTier(session.Tier),
			RetryAttempt: 0,
		}

		if err := s.queueClient.Publish(ctx, "verification_submission", submissionJob); err != nil {
			s.logger.Error("failed to enqueue verification submission",
				"session_id", session.ID,
				"error", err,
			)
			// Don't fail the upload if queue fails - job can be retried
		}
	}

	return ev, nil
}

func (s *verificationService) GetEvidence(ctx context.Context, evidenceID uuid.UUID, requesterID uuid.UUID) (*domain.Evidence, error) {
	ev, err := s.repo.GetEvidenceByID(ctx, evidenceID)
	if err != nil {
		return nil, err
	}

	// Authorize: verify requester owns the session
	session, err := s.repo.GetSessionByID(ctx, ev.SessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != requesterID {
		return nil, domain.ErrUnauthorized
	}

	return ev, nil
}

func (s *verificationService) ListSessionEvidence(ctx context.Context, sessionID uuid.UUID, requesterID uuid.UUID) ([]*domain.Evidence, error) {
	// Authorize: verify requester owns the session
	session, err := s.GetSession(ctx, sessionID, requesterID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListEvidenceBySessionID(ctx, session.ID)
}

func (s *verificationService) GenerateEvidenceSignedURL(ctx context.Context, evidenceID uuid.UUID, requesterID uuid.UUID) (string, error) {
	ev, err := s.GetEvidence(ctx, evidenceID, requesterID)
	if err != nil {
		return "", err
	}

	// Generate 15-minute signed URL
	signedURLResp, err := s.evidenceStore.GenerateSignedURL(ctx, ev.GetURL(), 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return signedURLResp.URL, nil
}
