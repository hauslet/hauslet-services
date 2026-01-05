package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/modules/verification/repository"
	"hauslet/internal/modules/verification/service"
	"hauslet/internal/platform/evidence"
	"hauslet/internal/platform/kyc"
	verificationJob "hauslet/internal/queue/jobs/verification"

	"github.com/google/uuid"
)

type SubmissionHandler struct {
	verificationSvc service.VerificationService
	repo            repository.VerificationRepository
	kycClient       *kyc.Client
	evidenceStore   evidence.Store
	log             *slog.Logger
	subject         string
}

func NewSubmissionHandler(
	verificationSvc service.VerificationService,
	repo repository.VerificationRepository,
	kycClient *kyc.Client,
	evidenceStore evidence.Store,
	log *slog.Logger,
	subject string,
) *SubmissionHandler {
	return &SubmissionHandler{
		verificationSvc: verificationSvc,
		repo:            repo,
		kycClient:       kycClient,
		evidenceStore:   evidenceStore,
		log:             log,
		subject:         subject,
	}
}

func (h *SubmissionHandler) JobType() string {
	return verificationJob.VerificationSubmissionJobType
}

func (h *SubmissionHandler) Subject() string {
	return h.subject
}

func (h *SubmissionHandler) Handle(ctx context.Context, data []byte) error {
	var job verificationJob.VerificationSubmissionJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal verification submission job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid verification submission job: %w", err)
	}

	sessionID, err := uuid.Parse(job.SessionID)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	h.log.Info("processing verification submission",
		"session_id", sessionID,
		"evidence_url", job.EvidenceURL,
		"priority", job.Priority,
		"retry_attempt", job.RetryAttempt,
	)

	session, err := h.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != domain.SessionInProgress && session.Status != domain.SessionPending {
		h.log.Warn("session already processed", "status", session.Status)
		return nil
	}

	if err := h.verifyEvidenceIntegrity(ctx, job.EvidenceURL, job.EvidenceHash); err != nil {
		h.log.Error("evidence integrity check failed", "error", err)
		return h.failSession(ctx, session, "evidence_integrity_failed", err.Error())
	}

	h.log.Info("skipping AI quality check (not implemented)")

	if err := h.submitToProvider(ctx, session); err != nil {
		h.log.Error("provider submission failed", "error", err)
		return h.handleProviderError(ctx, session, err, job.RetryAttempt)
	}

	h.log.Info("✅ verification submitted to provider successfully", "session_id", sessionID)
	return nil
}

func (h *SubmissionHandler) verifyEvidenceIntegrity(ctx context.Context, evidenceURL, expectedHash string) error {
	downloadResp, err := h.evidenceStore.Download(ctx, evidenceURL)
	if err != nil {
		return fmt.Errorf("failed to download evidence: %w", err)
	}

	hasher := sha256.New()
	hasher.Write(downloadResp.Data)
	actualHash := hex.EncodeToString(hasher.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func (h *SubmissionHandler) submitToProvider(ctx context.Context, session *domain.VerificationSession) error {
	if session.Data.Identity == nil {
		return fmt.Errorf("identity data is required")
	}

	identityData := session.Data.Identity

	kycReq := kyc.VerificationRequest{
		Country:      session.Country,
		DocumentType: kyc.DocumentType(identityData.DocumentInfo.Type),
		FirstName:    identityData.ApplicantInfo.FirstName,
		LastName:     identityData.ApplicantInfo.LastName,
		DateOfBirth:  &identityData.ApplicantInfo.DateOfBirth,
	}

	startTime := time.Now()
	resp, err := h.kycClient.SubmitVerification(ctx, kycReq)
	if err != nil {
		return fmt.Errorf("kyc submission failed: %w", err)
	}

	session.Status = domain.SessionInProgress
	session.UpdatedAt = time.Now()

	if err := h.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	h.log.Info("provider submission completed",
		"provider", resp.Provider,
		"provider_ref", resp.ProviderRef,
		"latency_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

func (h *SubmissionHandler) failSession(ctx context.Context, session *domain.VerificationSession, failureCode, reason string) error {
	session.Status = domain.SessionRejected
	notes := reason
	session.RejectionNotes = &notes
	now := time.Now()
	session.RejectedAt = &now
	session.CompletedAt = &now

	if err := h.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update failed session: %w", err)
	}

	return fmt.Errorf("session failed: %s", reason)
}

func (h *SubmissionHandler) handleProviderError(ctx context.Context, session *domain.VerificationSession, providerErr error, retryAttempt int) error {
	maxRetries := 3
	if retryAttempt >= maxRetries {
		return h.failSession(ctx, session, "provider_error", fmt.Sprintf("max retries exceeded: %v", providerErr))
	}

	h.log.Info("will retry verification submission",
		"session_id", session.ID,
		"retry_attempt", retryAttempt+1,
	)

	return h.failSession(ctx, session, "provider_error", providerErr.Error())
}
