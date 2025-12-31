package service

import (
	"context"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/moderation/domain"
	"hauslet/internal/modules/moderation/repository"
	"hauslet/internal/modules/moderation/repository/schema"
	aimoderation "hauslet/internal/platform/ai/moderation"
	platformQueue "hauslet/internal/platform/queue"
	moderationjobs "hauslet/internal/queue/jobs/moderation"

	"github.com/google/uuid"
)

const defaultMaxAttempts = 3

// ModerationServiceImpl implements ModerationService.
type ModerationServiceImpl struct {
	repo              repository.ModerationRepository
	aiClient          *aimoderation.Client
	queue             *platformQueue.Client
	aiSubject         string
	maxAIAttemptCount int
	propertyHooks     PropertyHooks
	profileHooks      ProfileHooks
	reviewHooks       ReviewHooks
	log               *slog.Logger
}

// NewModerationService constructs a moderation service.
func NewModerationService(repo repository.ModerationRepository,
	aiClient *aimoderation.Client,
	queue *platformQueue.Client,
	aiSubject string,
	propertyHooks PropertyHooks,
	profileHooks ProfileHooks,
	reviewHooks ReviewHooks,
	log *slog.Logger) ModerationService {
	return &ModerationServiceImpl{
		repo:              repo,
		aiClient:          aiClient,
		queue:             queue,
		aiSubject:         aiSubject,
		maxAIAttemptCount: defaultMaxAttempts,
		propertyHooks:     propertyHooks,
		profileHooks:      profileHooks,
		reviewHooks:       reviewHooks,
		log:               log,
	}
}

// EnqueueAIModeration records a moderation entry and publishes an AI job.
func (s *ModerationServiceImpl) EnqueueAIModeration(ctx context.Context, req CreateModerationRequest) (*domain.Moderation, error) {
	if req.ContentType == "" {
		return nil, fmt.Errorf("content type is required")
	}
	if req.TargetID == uuid.Nil {
		return nil, fmt.Errorf("target id is required")
	}
	if req.Payload == "" {
		return nil, fmt.Errorf("payload is required")
	}

	maxAttempts := req.MaxAIAttemptCount
	if maxAttempts <= 0 {
		maxAttempts = s.maxAIAttemptCount
	}

	reviewerType := req.ReviewerType
	if reviewerType == "" {
		reviewerType = domain.ReviewerTypeAI
	}

	record := &schema.Moderation{
		ID:                uuid.New(),
		ContentType:       schema.ContentType(req.ContentType),
		ContentID:         req.TargetID,
		MaxAIAttemptCount: maxAttempts,
		CurrentAttempt:    0,
		Status:            schema.ModerationStatusPending,
		ReviewerType:      schema.ReviewerType(reviewerType),
		Reason:            "",
		Metadata:          "{}",
	}

	if err := s.repo.Create(ctx, record); err != nil {
		s.log.Error("failed to create moderation record for target", "target_id", req.TargetID, "error", err)
		return nil, err
	}

	if s.queue != nil && s.aiSubject != "" {
		job := moderationjobs.AIModerationJob{
			ModerationID: record.ID,
			TargetID:     req.TargetID,
			ContentType:  moderationjobs.ContentType(req.ContentType),
			Payload:      req.Payload,
		}
		if err := job.Validate(); err != nil {
			s.log.Error("invalid AI moderation job for target", "target_id", req.TargetID, "error", err)
			return nil, err
		}
		if err := s.queue.Publish(ctx, s.aiSubject, job); err != nil {
			s.log.Error("failed to publish AI moderation job for target", "target_id", req.TargetID, "error", err)
			return nil, err
		}
		s.log.Info("enqueued AI moderation job for target", "target_id", req.TargetID, "content_type", req.ContentType)
	} else {
		s.log.Warn("queue or AI subject not configured, moderation created but not enqueued", "moderation_id", record.ID)
	}

	return mapToDomain(record), nil
}

// HandleAIJob processes an AI moderation job and updates the moderation record.
func (s *ModerationServiceImpl) HandleAIJob(ctx context.Context, job moderationjobs.AIModerationJob) (*domain.Moderation, error) {
	if s.aiClient == nil {
		s.log.Error("AI client not configured for job", "moderation_id", job.ModerationID)
		return nil, fmt.Errorf("ai client is not configured")
	}
	if err := job.Validate(); err != nil {
		s.log.Error("invalid AI job for moderation", "moderation_id", job.ModerationID, "error", err)
		return nil, err
	}

	record, err := s.repo.GetByID(ctx, job.ModerationID)
	if err != nil {
		s.log.Error("failed to fetch moderation", "moderation_id", job.ModerationID, "error", err)
		return nil, err
	}
	if record == nil {
		s.log.Warn("moderation not found", "moderation_id", job.ModerationID)
		return nil, fmt.Errorf("moderation %s not found", job.ModerationID)
	}

	input := aimoderation.AIModerationInput{
		ModerationID:  job.ModerationID,
		ContentType:   schema.ContentType(job.ContentType),
		ContentID:     job.TargetID,
		AttemptNumber: record.CurrentAttempt + 1,
		MaxAttempts:   record.MaxAIAttemptCount,
	}

	switch job.ContentType {
	case moderationjobs.ContentTypeListingImage, moderationjobs.ContentTypeProfileImage, moderationjobs.ContentTypeTravelCompImage, moderationjobs.ContentTypeReviewImage:
		input.Image = &aimoderation.ImagePayload{Key: job.Payload}
	case moderationjobs.ContentTypeListingVideo:
		input.Video = &aimoderation.VideoPayload{Key: job.Payload}
	default:
		text := job.Payload
		input.Text = &text
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	s.log.Info("processing AI moderation for target", "target_id", job.TargetID, "attempt", input.AttemptNumber, "max_attempts", input.MaxAttempts)

	result, err := s.aiClient.Moderate(ctx, input)
	if err != nil {
		s.log.Error("AI moderation failed for target", "target_id", job.TargetID, "attempt", input.AttemptNumber, "max_attempts", input.MaxAttempts, "error", err)
		// Persist the attempt count so subsequent retries see the incremented AttemptNumber.
		record.CurrentAttempt = input.AttemptNumber
		_ = s.repo.Update(ctx, record) // best-effort; moderation error takes precedence
		return nil, err
	}

	record.Status = schema.ModerationStatus(result.Status)
	record.ReviewerType = schema.ReviewerTypeAI
	record.AIConfidence = &result.Confidence
	record.Reason = result.Reason
	record.CurrentAttempt = input.AttemptNumber

	s.log.Info("AI moderation completed for target", "target_id", job.TargetID, "status", result.Status, "confidence", result.Confidence)

	if err := s.repo.Update(ctx, record); err != nil {
		s.log.Error("failed to update moderation record", "moderation_id", job.ModerationID, "error", err)
		return nil, err
	}

	// Aggregate moderation state for the target and, if terminal, notify property hooks.
	aggregate, err := s.repo.AggregateByTarget(ctx, record.ContentID)
	if err != nil {
		s.log.Error("failed to aggregate moderation for target", "target_id", record.ContentID, "error", err)
		return nil, err
	}

	serviceAggregate := mapAggregateToService(aggregate)

	if serviceAggregate.FinalStatus() != domain.ModerationStatusPending &&
		s.propertyHooks != nil &&
		containsListingContent(serviceAggregate.ContentTypes) {
		s.log.Info("invoking property hooks for target", "target_id", record.ContentID, "final_status", serviceAggregate.FinalStatus())
		if err := s.propertyHooks.OnModerationCompleted(ctx, serviceAggregate); err != nil {
			s.log.Error("property hooks failed for target", "target_id", record.ContentID, "error", err)
			return nil, err
		}
	}

	if serviceAggregate.FinalStatus() != domain.ModerationStatusPending &&
		s.profileHooks != nil &&
		containsProfileContent(serviceAggregate.ContentTypes) {
		s.log.Info("invoking profile hooks for target", "target_id", record.ContentID, "final_status", serviceAggregate.FinalStatus())
		if err := s.profileHooks.OnModerationCompleted(ctx, serviceAggregate); err != nil {
			s.log.Error("profile hooks failed for target", "target_id", record.ContentID, "error", err)
			return nil, err
		}
	}

	if serviceAggregate.FinalStatus() != domain.ModerationStatusPending &&
		s.reviewHooks != nil &&
		containsReviewContent(serviceAggregate.ContentTypes) {
		s.log.Info("invoking review hooks for target", "target_id", record.ContentID, "final_status", serviceAggregate.FinalStatus())
		if err := s.reviewHooks.OnModerationCompleted(ctx, serviceAggregate); err != nil {
			s.log.Error("review hooks failed for target", "target_id", record.ContentID, "error", err)
			return nil, err
		}
	}

	return mapToDomain(record), nil
}

func (s *ModerationServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.Moderation, error) {
	rec, err := s.repo.GetByID(ctx, id)
	if err != nil || rec == nil {
		return mapToDomain(rec), err
	}
	return mapToDomain(rec), nil
}

func (s *ModerationServiceImpl) GetLatestByTarget(ctx context.Context, contentType domain.ContentType, targetID uuid.UUID) (*domain.Moderation, error) {
	rec, err := s.repo.GetLatestByTarget(ctx, schema.ContentType(contentType), targetID)
	if err != nil || rec == nil {
		return mapToDomain(rec), err
	}
	return mapToDomain(rec), nil
}

func (s *ModerationServiceImpl) GetHistoryByTarget(ctx context.Context, targetID uuid.UUID) ([]domain.Moderation, error) {
	recs, err := s.repo.GetAllByTarget(ctx, targetID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Moderation, 0, len(recs))
	for _, r := range recs {
		if mapped := mapToDomain(r); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result, nil
}

func (s *ModerationServiceImpl) ListPending(ctx context.Context, contentType domain.ContentType, limit, offset int) ([]domain.Moderation, error) {
	recs, err := s.repo.GetPendingItems(ctx, schema.ContentType(contentType), limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Moderation, 0, len(recs))
	for _, r := range recs {
		if mapped := mapToDomain(r); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result, nil
}

func (s *ModerationServiceImpl) ListEscalated(ctx context.Context, limit, offset int) ([]domain.Moderation, error) {
	recs, err := s.repo.GetEscalatedItems(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Moderation, 0, len(recs))
	for _, r := range recs {
		if mapped := mapToDomain(r); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result, nil
}

func (s *ModerationServiceImpl) BatchUpdateStatus(ctx context.Context, ids []uuid.UUID, status domain.ModerationStatus, reviewerID *uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return s.repo.BatchUpdateStatus(ctx, ids, schema.ModerationStatus(status), reviewerID)
}

// RegisterEnqueueHooks allows wiring a property callback after construction.
func (s *ModerationServiceImpl) RegisterEnqueueHooks(hooks PropertyHooks) {
	s.propertyHooks = hooks
}

// mapToDomain converts schema model to domain model.
func mapToDomain(m *schema.Moderation) *domain.Moderation {
	if m == nil {
		return nil
	}

	return &domain.Moderation{
		ID:             m.ID,
		ContentType:    domain.ContentType(m.ContentType),
		TargetID:       m.ContentID,
		AttemptCount:   m.MaxAIAttemptCount,
		CurrentAttempt: m.CurrentAttempt,
		Status:         domain.ModerationStatus(m.Status),
		ReviewerType:   domain.ReviewerType(m.ReviewerType),
		ReviewerID:     m.ReviewerID,
		AIConfidence:   m.AIConfidence,
		Reason:         m.Reason,
		UpdatedAt:      m.UpdatedAt,
	}
}

func mapAggregateToService(agg *repository.AggregatedCounts) AggregatedModeration {
	if agg == nil {
		return AggregatedModeration{}
	}

	contentTypes := make([]domain.ContentType, 0, len(agg.ContentTypes))
	for _, ct := range agg.ContentTypes {
		contentTypes = append(contentTypes, domain.ContentType(ct))
	}

	return AggregatedModeration{
		TargetID:     agg.TargetID,
		Pending:      agg.Pending,
		Accepted:     agg.Accepted,
		Rejected:     agg.Rejected,
		Escalated:    agg.Escalated,
		ContentTypes: contentTypes,
		Reasons:      agg.Reasons,
	}
}

func containsListingContent(contentTypes []domain.ContentType) bool {
	for _, ct := range contentTypes {
		if ct.IsListingContent() {
			return true
		}
	}
	return false
}

// containsProfileContent checks if any content type relates to profiles.
func containsProfileContent(contentTypes []domain.ContentType) bool {
	for _, ct := range contentTypes {
		if ct.IsProfileContent() {
			return true
		}
	}
	return false
}

// containsReviewContent checks if any content type relates to reviews.
func containsReviewContent(contentTypes []domain.ContentType) bool {
	for _, ct := range contentTypes {
		if ct.IsReviewContent() {
			return true
		}
	}
	return false
}
