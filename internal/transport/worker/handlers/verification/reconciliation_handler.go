package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	profileService "hauslet/internal/modules/profile/service"
	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/modules/verification/repository"
	verificationJob "hauslet/internal/queue/jobs/verification"
)

type ReconciliationHandler struct {
	repo       repository.VerificationRepository
	profileSvc profileService.ProfileService
	log        *slog.Logger
	subject    string
}

func NewReconciliationHandler(
	repo repository.VerificationRepository,
	profileSvc profileService.ProfileService,
	log *slog.Logger,
	subject string,
) *ReconciliationHandler {
	return &ReconciliationHandler{
		repo:       repo,
		profileSvc: profileSvc,
		log:        log,
		subject:    subject,
	}
}

func (h *ReconciliationHandler) JobType() string {
	return verificationJob.ReconciliationJobType
}

func (h *ReconciliationHandler) Subject() string {
	return h.subject
}

func (h *ReconciliationHandler) Handle(ctx context.Context, data []byte) error {
	var job verificationJob.ReconciliationJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal reconciliation job: %w", err)
	}

	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid reconciliation job: %w", err)
	}

	h.log.Info("starting verification reconciliation", "batch_size", job.BatchSize)

	sessions, err := h.repo.ListExpiredSessions(ctx, job.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	reconciledCount := 0
	errorCount := 0

	for _, session := range sessions {
		if session.Status == domain.SessionApproved {
			if err := h.reconcileSession(ctx, session); err != nil {
				h.log.Error("reconciliation failed for session",
					"session_id", session.ID,
					"error", err,
				)
				errorCount++
				continue
			}
			reconciledCount++
		}
	}

	h.log.Info("reconciliation completed",
		"total_sessions", len(sessions),
		"reconciled", reconciledCount,
		"errors", errorCount,
	)

	return nil
}

func (h *ReconciliationHandler) reconcileSession(ctx context.Context, session *domain.VerificationSession) error {
	profileType := h.mapSessionTypeToProfileType(session.Type)

	if err := h.profileSvc.SetVerificationStatus(ctx, session.UserID.String(), profileType, true, nil); err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	h.log.Info("session reconciled with profile",
		"session_id", session.ID,
		"user_id", session.UserID,
		"type", session.Type,
	)

	return nil
}

func (h *ReconciliationHandler) mapSessionTypeToProfileType(sessionType domain.VerificationType) string {
	switch sessionType {
	case domain.VerificationIdentity:
		return "identity"
	case domain.VerificationAddress:
		return "address"
	case domain.VerificationPhone:
		return "phone"
	case domain.VerificationBusiness:
		return "business"
	default:
		return "unknown"
	}
}
