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

	// 1) Status counts
	type statusRow struct {
		Status schema.ModerationStatus
		Count  int64
	}
	var rows []statusRow
	if err := r.db.WithContext(ctx).
		Model(&schema.Moderation{}).
		Select("status, COUNT(*) as count").
		Where("content_id = ?", targetID).
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate moderation statuses: %w", err)
	}

	for _, row := range rows {
		switch row.Status {
		case schema.ModerationStatusPending:
			result.Pending = row.Count
		case schema.ModerationStatusAccepted:
			result.Accepted = row.Count
		case schema.ModerationStatusRejected:
			result.Rejected = row.Count
		case schema.ModerationStatusEscalated:
			result.Escalated = row.Count
		}
	}

	// 2) Distinct content types for this target
	var contentTypes []string
	if err := r.db.WithContext(ctx).
		Model(&schema.Moderation{}).
		Distinct().
		Where("content_id = ?", targetID).
		Pluck("content_type", &contentTypes).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate moderation content types: %w", err)
	}
	result.ContentTypes = make([]schema.ContentType, 0, len(contentTypes))
	for _, ct := range contentTypes {
		result.ContentTypes = append(result.ContentTypes, schema.ContentType(ct))
	}

	// 3) Collect reasons for rejected/escalated entries (for notifications/labels)
	var reasons []string
	if err := r.db.WithContext(ctx).
		Model(&schema.Moderation{}).
		Where("content_id = ? AND status IN ?", targetID,
			[]schema.ModerationStatus{schema.ModerationStatusRejected, schema.ModerationStatusEscalated}).
		Pluck("reason", &reasons).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate moderation reasons: %w", err)
	}
	result.Reasons = reasons

	return result, nil
}
