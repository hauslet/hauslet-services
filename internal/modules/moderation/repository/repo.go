package repository

import (
	"context"
	"errors"
	"fmt"

	"hauslet/internal/modules/moderation/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *ModerationRepositoryImpl) Create(ctx context.Context, moderation *schema.Moderation) error {
	if moderation == nil {
		return fmt.Errorf("moderation cannot be nil")
	}

	if err := r.db.WithContext(ctx).Create(moderation).Error; err != nil {
		return fmt.Errorf("failed to create moderation: %w", err)
	}

	return nil
}

func (r *ModerationRepositoryImpl) Update(ctx context.Context, moderation *schema.Moderation) error {
	if moderation == nil {
		return fmt.Errorf("moderation cannot be nil")
	}
	if moderation.ID == uuid.Nil {
		return fmt.Errorf("moderation ID cannot be empty")
	}

	result := r.db.WithContext(ctx).Save(moderation)
	if result.Error != nil {
		return fmt.Errorf("failed to update moderation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("moderation not found: %w", gorm.ErrRecordNotFound)
	}

	return nil
}

func (r *ModerationRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*schema.Moderation, error) {
	var moderation schema.Moderation

	if err := r.db.WithContext(ctx).First(&moderation, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get moderation by id: %w", err)
	}

	return &moderation, nil
}

func (r *ModerationRepositoryImpl) GetLatestByTarget(ctx context.Context, contentType schema.ContentType, targetID uuid.UUID) (*schema.Moderation, error) {
	var moderation schema.Moderation

	query := r.db.WithContext(ctx).
		Where("content_type = ? AND content_id = ?", contentType, targetID).
		Order("updated_at DESC")

	if err := query.First(&moderation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest moderation for target %s: %w", targetID, err)
	}

	return &moderation, nil
}

func (r *ModerationRepositoryImpl) GetAllByTarget(ctx context.Context, targetID uuid.UUID) ([]*schema.Moderation, error) {
	var moderations []*schema.Moderation

	if err := r.db.WithContext(ctx).
		Where("content_id = ?", targetID).
		Order("updated_at DESC").
		Find(&moderations).Error; err != nil {
		return nil, fmt.Errorf("failed to get moderations for target %s: %w", targetID, err)
	}

	return moderations, nil
}

func (r *ModerationRepositoryImpl) GetPendingItems(ctx context.Context, contentType schema.ContentType, limit, offset int) ([]*schema.Moderation, error) {
	var moderations []*schema.Moderation

	query := r.db.WithContext(ctx).
		Where("status = ?", schema.ModerationStatusPending).
		Order("updated_at ASC")

	if contentType != "" {
		query = query.Where("content_type = ?", contentType)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&moderations).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending moderations: %w", err)
	}

	return moderations, nil
}

func (r *ModerationRepositoryImpl) GetEscalatedItems(ctx context.Context, limit, offset int) ([]*schema.Moderation, error) {
	var moderations []*schema.Moderation

	query := r.db.WithContext(ctx).
		Where("status = ?", schema.ModerationStatusEscalated).
		Order("updated_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&moderations).Error; err != nil {
		return nil, fmt.Errorf("failed to get escalated moderations: %w", err)
	}

	return moderations, nil
}

func (r *ModerationRepositoryImpl) BatchUpdateStatus(ctx context.Context, ids []uuid.UUID, status schema.ModerationStatus, reviewerID *uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	updates := map[string]any{
		"status":      status,
		"reviewer_id": reviewerID,
	}

	result := r.db.WithContext(ctx).
		Model(&schema.Moderation{}).
		Where("id IN ?", ids).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to batch update moderation status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no moderations updated: %w", gorm.ErrRecordNotFound)
	}

	return nil
}
