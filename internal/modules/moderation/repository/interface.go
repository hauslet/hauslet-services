package repository

import (
	"context"

	"hauslet/internal/modules/moderation/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ModerationRepository interface {
	Create(ctx context.Context, moderation *schema.Moderation) error
	Update(ctx context.Context, moderation *schema.Moderation) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Moderation, error)

	// GetLatestByTarget fetches the current active moderation status for a specific entity.
	GetLatestByTarget(ctx context.Context, contentType schema.ContentType, targetID uuid.UUID) (*schema.Moderation, error)

	// GetAllByTarget fetches moderation history for an entity (e.g. was this rejected before?).
	GetAllByTarget(ctx context.Context, targetID uuid.UUID) ([]*schema.Moderation, error)

	// Worker / Queue Methods
	// GetPendingItems fetches items needing AI or Human review.
	GetPendingItems(ctx context.Context, contentType schema.ContentType, limit, offset int) ([]*schema.Moderation, error)

	// GetEscalatedItems fetches items specifically flagged for human admins.
	GetEscalatedItems(ctx context.Context, limit, offset int) ([]*schema.Moderation, error)

	// Admin Tools
	// BatchUpdateStatus allows approving/rejecting multiple items at once.
	BatchUpdateStatus(ctx context.Context, ids []uuid.UUID, status schema.ModerationStatus, reviewerID *uuid.UUID) error

	// Aggregation
	AggregateByTarget(ctx context.Context, targetID uuid.UUID) (*AggregatedCounts, error)
}

type ModerationRepositoryImpl struct {
	db *gorm.DB
}

func NewModerationRepository(db *gorm.DB) ModerationRepository {
	return &ModerationRepositoryImpl{db: db}
}
