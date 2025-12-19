package service

import (
	"context"

	"hauslet/internal/modules/moderation/domain"
	moderationjobs "hauslet/internal/queue/jobs/moderation"

	"github.com/google/uuid"
)

// ModerationService defines business operations for AI and human moderation.
// It orchestrates repository writes, AI calls, and job scheduling.
type ModerationService interface {
	// EnqueueAIModeration records a moderation entry and publishes an AI job.
	EnqueueAIModeration(ctx context.Context, req CreateModerationRequest) (*domain.Moderation, error)

	// HandleAIJob processes an AI moderation job and updates the moderation record.
	HandleAIJob(ctx context.Context, job moderationjobs.AIModerationJob) (*domain.Moderation, error)

	// Query helpers
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Moderation, error)
	GetLatestByTarget(ctx context.Context, contentType domain.ContentType, targetID uuid.UUID) (*domain.Moderation, error)
	GetHistoryByTarget(ctx context.Context, targetID uuid.UUID) ([]domain.Moderation, error)
	ListPending(ctx context.Context, contentType domain.ContentType, limit, offset int) ([]domain.Moderation, error)
	ListEscalated(ctx context.Context, limit, offset int) ([]domain.Moderation, error)

	// Administrative actions
	BatchUpdateStatus(ctx context.Context, ids []uuid.UUID, status domain.ModerationStatus, reviewerID *uuid.UUID) error

	// Integration
	RegisterEnqueueHooks(hooks PropertyHooks)
}

type PropertyHooks interface {
	OnModerationCompleted(ctx context.Context, aggregate AggregatedModeration) error
}

// AggregatedModeration summarizes moderation state for a target content ID.
type AggregatedModeration struct {
	TargetID uuid.UUID

	Pending   int64
	Accepted  int64
	Rejected  int64
	Escalated int64

	ContentTypes []domain.ContentType
	Reasons      []string
}

// FinalStatus returns the terminal moderation status for the aggregate.
func (a AggregatedModeration) FinalStatus() domain.ModerationStatus {
	if a.Pending > 0 {
		return domain.ModerationStatusPending
	}
	if a.Rejected > 0 {
		return domain.ModerationStatusRejected
	}
	if a.Escalated > 0 {
		return domain.ModerationStatusEscalated
	}
	if a.Accepted > 0 {
		return domain.ModerationStatusAccepted
	}
	return domain.ModerationStatusPending
}

// CreateModerationRequest captures inputs for scheduling AI moderation.
// Payload can be raw text or a storage key/URL depending on content type.
type CreateModerationRequest struct {
	ContentType       domain.ContentType
	TargetID          uuid.UUID
	Payload           string
	MaxAIAttemptCount int
	ReviewerType      domain.ReviewerType // default to AI if omitted in implementation
}
