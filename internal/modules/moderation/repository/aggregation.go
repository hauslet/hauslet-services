package repository

import (
	"context"
	"fmt"

	"hauslet/internal/modules/moderation/repository/schema"

	"github.com/google/uuid"
)

// AggregatedCounts captures per-status counts for a moderation target.
type AggregatedCounts struct {
	TargetID uuid.UUID

	Pending   int64
	Accepted  int64
	Rejected  int64
	Escalated int64

	ContentTypes []schema.ContentType
	Reasons      []string
}

// AggregateByTarget summarizes moderation state for a given target.
func (r *ModerationRepositoryImpl) AggregateByTarget(ctx context.Context, targetID uuid.UUID) (*AggregatedCounts, error) {
	if targetID == uuid.Nil {
		return nil, fmt.Errorf("targetID cannot be empty")
	}

	result := &AggregatedCounts{TargetID: targetID}

	// Fetch all moderation records for the target ordered by latest update.
	var moderations []schema.Moderation
	if err := r.db.WithContext(ctx).
		Model(&schema.Moderation{}).
		Where("content_id = ?", targetID).
		Order("updated_at DESC").
		Find(&moderations).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate moderation statuses: %w", err)
	}

	// Pick the latest record per content type to avoid counting stale runs.
	latestByType := make(map[schema.ContentType]schema.Moderation)
	for _, m := range moderations {
		if _, exists := latestByType[m.ContentType]; exists {
			continue
		}
		latestByType[m.ContentType] = m
	}

	// Aggregate counts from the latest records only.
	for _, m := range latestByType {
		switch m.Status {
		case schema.ModerationStatusPending:
			result.Pending++
		case schema.ModerationStatusAccepted:
			result.Accepted++
		case schema.ModerationStatusRejected:
			result.Rejected++
		case schema.ModerationStatusEscalated:
			result.Escalated++
		}
		result.ContentTypes = append(result.ContentTypes, m.ContentType)
		if m.Status == schema.ModerationStatusRejected || m.Status == schema.ModerationStatusEscalated {
			if m.Reason != "" {
				result.Reasons = append(result.Reasons, m.Reason)
			}
		}
	}

	return result, nil
}
