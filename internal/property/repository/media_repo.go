package repository

import (
	"context"
	"fmt"
	"hauslet/internal/property/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReplaceListingMedia swaps the media collection for a listing in a transaction.
func (r *GormRepository) ReplaceListingMedia(ctx context.Context, id uuid.UUID, media []schema.ListingMedia) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("listing_id = ?", id).Delete(&schema.ListingMedia{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing media: %w", err)
		}
		if len(media) == 0 {
			return nil
		}
		for i := range media {
			media[i].ListingID = id
		}
		if err := tx.CreateInBatches(&media, BatchInsertSize).Error; err != nil {
			return fmt.Errorf("failed to insert new media: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to replace listing media: %w", err)
	}
	return nil
}

// AddListingMedia appends media to a listing.
func (r *GormRepository) AddListingMedia(ctx context.Context, listingID uuid.UUID, media []schema.ListingMedia) error {
	if len(media) == 0 {
		return nil
	}
	for i := range media {
		media[i].ListingID = listingID
	}
	if err := r.db.WithContext(ctx).CreateInBatches(&media, BatchInsertSize).Error; err != nil {
		return fmt.Errorf("failed to add listing media: %w", err)
	}
	return nil
}

// DeleteListingMedia removes media by IDs for a listing. If mediaIDs is empty, no-op.
func (r *GormRepository) DeleteListingMedia(ctx context.Context, listingID uuid.UUID, mediaIDs []uuid.UUID) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Where("listing_id = ? AND id IN ?", listingID, mediaIDs).Delete(&schema.ListingMedia{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete listing media: %w", result.Error)
	}
	return nil
}

// ListListingMedia fetches media for a listing ordered by "order" then created_at.
func (r *GormRepository) ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]schema.ListingMedia, error) {
	var media []schema.ListingMedia
	if err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Order("order ASC").
		Order("created_at ASC").
		Find(&media).Error; err != nil {
		return nil, fmt.Errorf("failed to list listing media: %w", err)
	}
	return media, nil
}

// UpdateListingMedia applies partial updates to a media record scoped to a listing.
func (r *GormRepository) UpdateListingMedia(ctx context.Context, listingID uuid.UUID, mediaID uuid.UUID, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&schema.ListingMedia{}).
		Where("listing_id = ? AND id = ?", listingID, mediaID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update listing media: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing media not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}
