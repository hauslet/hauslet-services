package repository

import (
	"context"
	"fmt"
	"hauslet/internal/modules/property/repository/schema"
)

// GetListingsWithoutEmbedding finds published and active listings that do not have vector embeddings.
func (r *GormRepository) GetListingsWithoutEmbedding(ctx context.Context, limit int) ([]schema.Listing, error) {
	var listings []schema.Listing
	// We want listings that are active, published, and have no embedding
	err := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("status = ?", schema.StatusActive).
		Where("published = ?", true).
		Where("text_embedding IS NULL").
		Where("deleted_at IS NULL").
		Limit(limit).
		Find(&listings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get listings without embedding: %w", err)
	}
	return listings, nil
}
