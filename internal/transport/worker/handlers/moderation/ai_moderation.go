package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/moderation/service"
	moderationjobs "hauslet/internal/queue/jobs/moderation"
)

// AIModerationHandler processes AI moderation jobs.
type AIModerationHandler struct {
	service service.ModerationService
	log     *slog.Logger
	subject string
}

// NewAIModerationHandler constructs an AI moderation handler.
func NewAIModerationHandler(service service.ModerationService, log *slog.Logger, subject string) *AIModerationHandler {
	return &AIModerationHandler{
		service: service,
		log:     log,
		subject: subject,
	}
}

func (h *AIModerationHandler) JobType() string {
	return moderationjobs.AIModerationJobType
}

func (h *AIModerationHandler) Subject() string {
	return h.subject
}

func (h *AIModerationHandler) Handle(ctx context.Context, data []byte) error {
	var job moderationjobs.AIModerationJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal ai moderation job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid ai moderation job: %w", err)
	}

	result, err := h.service.HandleAIJob(ctx, job)
	if err != nil {
		return fmt.Errorf("process ai moderation job: %w", err)
	}

	if result != nil {
		h.log.Info("moderation updated	",
			"id", result.ID,
			"status", result.Status,
			"content_type", result.ContentType,
			"reviewer", result.ReviewerType,
			"current_attempt", result.CurrentAttempt,
			"attempt_count", result.AttemptCount,
		)
	}
	return nil
}
