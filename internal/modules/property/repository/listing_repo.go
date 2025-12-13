package repository

import (
	"context"
	"errors"
	"fmt"
	"hauslet/internal/modules/property/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateListing inserts a new listing record.
func (r *GormRepository) CreateListing(ctx context.Context, listing *schema.Listing) error {
	if listing == nil {
		return fmt.Errorf("listing cannot be nil")
	}

	// Ensure property exists before creating the listing.
	exists, err := r.PropertyExists(ctx, listing.PropertyID)
	if err != nil {
		return fmt.Errorf("failed to verify property for listing: %w", err)
	}
	if !exists {
		return fmt.Errorf("property %s not found: %w", listing.PropertyID, gorm.ErrRecordNotFound)
	}

	// Enforce one listing per property at the application layer (DB unique index also exists).
	var existing int64
	if err := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("property_id = ?", listing.PropertyID).
		Count(&existing).Error; err != nil {
		return fmt.Errorf("failed to check existing listing for property: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("listing already exists for property %s", listing.PropertyID)
	}

	if err := r.db.WithContext(ctx).Create(listing).Error; err != nil {
		return fmt.Errorf("failed to create listing: %w", err)
	}
	return nil
}

// CreateListingTx inserts a new listing record within a transaction.
func (r *GormRepository) CreateListingTx(ctx context.Context, tx *gorm.DB, listing *schema.Listing) error {
	if listing == nil {
		return fmt.Errorf("listing cannot be nil")
	}

	// Ensure property exists before creating the listing.
	var propertyCount int64
	if err := tx.WithContext(ctx).Model(&schema.Property{}).Where("id = ?", listing.PropertyID).Count(&propertyCount).Error; err != nil {
		return fmt.Errorf("failed to verify property for listing: %w", err)
	}
	if propertyCount == 0 {
		return fmt.Errorf("property %s not found: %w", listing.PropertyID, gorm.ErrRecordNotFound)
	}

	// Enforce one listing per property at the application layer (DB unique index also exists).
	var existing int64
	if err := tx.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("property_id = ?", listing.PropertyID).
		Count(&existing).Error; err != nil {
		return fmt.Errorf("failed to check existing listing for property: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("listing already exists for property %s", listing.PropertyID)
	}

	if err := tx.WithContext(ctx).Create(listing).Error; err != nil {
		return fmt.Errorf("failed to create listing: %w", err)
	}
	return nil
}

// UpdateListing updates all fields on an existing listing.
func (r *GormRepository) UpdateListing(ctx context.Context, listing *schema.Listing) error {
	result := r.db.WithContext(ctx).Save(listing)
	if result.Error != nil {
		return fmt.Errorf("failed to update listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// UpdateListingTx updates all fields on an existing listing within a transaction.
func (r *GormRepository) UpdateListingTx(ctx context.Context, tx *gorm.DB, listing *schema.Listing) error {
	result := tx.WithContext(ctx).Save(listing)
	if result.Error != nil {
		return fmt.Errorf("failed to update listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// PatchListing applies partial updates to a listing by ID.
func (r *GormRepository) PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to patch listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// PatchListingTx applies partial updates to a listing by ID within a transaction.
func (r *GormRepository) PatchListingTx(ctx context.Context, tx *gorm.DB, id uuid.UUID, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := tx.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to patch listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// GetListingByID fetches a listing by primary key.
func (r *GormRepository) GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*schema.Listing, error) {
	var listing schema.Listing
	query := r.db.WithContext(ctx)
	if preloadMedia {
		query = query.Preload("Media")
	}
	if err := query.First(&listing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("listing not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get listing: %w", err)
	}
	return &listing, nil
}

// GetListingBySlug fetches a listing by slug.
func (r *GormRepository) GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*schema.Listing, error) {
	var listing schema.Listing
	query := r.db.WithContext(ctx)
	if preloadMedia {
		query = query.Preload("Media")
	}
	if err := query.First(&listing, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("listing not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get listing by slug: %w", err)
	}
	return &listing, nil
}

// GetListingByPublicID fetches a listing by the associated property's public ID.
func (r *GormRepository) GetListingByPublicID(ctx context.Context, publicID string, preloadMedia bool) (*schema.Listing, error) {
	var listing schema.Listing

	query := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Joins("JOIN properties ON properties.id = listings.property_id").
		Where("properties.public_id = ?", publicID)
		// Preload("Property")

	if preloadMedia {
		query = query.Preload("Media")
	}

	if err := query.Select("listings.*").First(&listing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("listing not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get listing by public id: %w", err)
	}

	return &listing, nil
}

// ListListings returns listings that match the provided filter and pagination.
func (r *GormRepository) ListListings(ctx context.Context, filter ListingFilter, page Pagination) (*PaginatedResult[schema.Listing], error) {
	var listings []schema.Listing
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).Model(&schema.Listing{})
	if !filter.IncludeDeleted {
		baseQuery = baseQuery.Where("deleted_at IS NULL")
	}
	baseQuery = applyListingFilter(baseQuery, filter)

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count listings: %w", err)
	}

	// Apply sorting and pagination
	query := baseQuery
	query = applyListingSort(query, filter.SortBy, filter.SortOrder)
	query = applyPagination(query, page)

	if err := query.Preload("Media").Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to list listings: %w", err)
	}

	return &PaginatedResult[schema.Listing]{
		Items:      listings,
		TotalCount: totalCount,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}

// ListListingsByPropertyID returns all listings for a specific property.
func (r *GormRepository) ListListingsByPropertyID(ctx context.Context, propertyID uuid.UUID, page Pagination) (*PaginatedResult[schema.Listing], error) {
	var listings []schema.Listing
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).Model(&schema.Listing{}).Where("property_id = ?", propertyID)

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count listings by property: %w", err)
	}

	// Apply pagination
	query := baseQuery
	query = applyPagination(query, page)
	query = query.Order("created_at DESC")

	if err := query.Preload("Media").Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to list listings by property: %w", err)
	}

	return &PaginatedResult[schema.Listing]{
		Items:      listings,
		TotalCount: totalCount,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}

// CountListings returns the count of listings matching the filter.
func (r *GormRepository) CountListings(ctx context.Context, filter ListingFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&schema.Listing{})
	if !filter.IncludeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	query = applyListingFilter(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count listings: %w", err)
	}
	return count, nil
}

// ListingExists checks if a listing exists by ID.
func (r *GormRepository) ListingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&schema.Listing{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check listing existence: %w", err)
	}
	return count > 0, nil
}

// GetListingsByIDs fetches multiple listings by their IDs.
func (r *GormRepository) GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]schema.Listing, error) {
	if len(ids) == 0 {
		return []schema.Listing{}, nil
	}

	var listings []schema.Listing
	query := r.db.WithContext(ctx).Where("id IN ?", ids)
	if preloadMedia {
		query = query.Preload("Media")
	}

	if err := query.Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to get listings by IDs: %w", err)
	}

	return listings, nil
}

// GetListingsByPropertyIDs fetches all listings for multiple properties.
func (r *GormRepository) GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]schema.Listing, error) {
	if len(propertyIDs) == 0 {
		return []schema.Listing{}, nil
	}

	var listings []schema.Listing
	if err := r.db.WithContext(ctx).
		Where("property_id IN ?", propertyIDs).
		Order("created_at DESC").
		Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to get listings by property IDs: %w", err)
	}

	return listings, nil
}

// BulkCreateListings inserts multiple listings in batches.
func (r *GormRepository) BulkCreateListings(ctx context.Context, listings []schema.Listing) error {
	if len(listings) == 0 {
		return nil
	}

	// Collect property IDs and ensure no duplicates within the batch.
	propertyIDs := make([]uuid.UUID, 0, len(listings))
	seenProps := make(map[uuid.UUID]struct{})
	for _, l := range listings {
		if l.PropertyID == uuid.Nil {
			return fmt.Errorf("listing property_id cannot be nil")
		}
		if _, ok := seenProps[l.PropertyID]; ok {
			return fmt.Errorf("duplicate property_id %s in bulk listings", l.PropertyID)
		}
		seenProps[l.PropertyID] = struct{}{}
		propertyIDs = append(propertyIDs, l.PropertyID)
	}

	// Ensure all referenced properties exist.
	var propsCount int64
	if err := r.db.WithContext(ctx).
		Model(&schema.Property{}).
		Where("id IN ?", propertyIDs).
		Count(&propsCount).Error; err != nil {
		return fmt.Errorf("failed to verify properties for listings: %w", err)
	}
	if propsCount != int64(len(propertyIDs)) {
		return fmt.Errorf("one or more properties referenced by listings do not exist")
	}

	// Ensure no existing listings already use these properties.
	var existing int64
	if err := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("property_id IN ?", propertyIDs).
		Count(&existing).Error; err != nil {
		return fmt.Errorf("failed to check existing listings for properties: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("one or more properties already have listings; bulk create aborted")
	}

	if err := r.db.WithContext(ctx).CreateInBatches(listings, BatchInsertSize).Error; err != nil {
		return fmt.Errorf("failed to bulk create listings: %w", err)
	}
	return nil
}

// BulkDeleteListings deletes multiple listings by IDs.
func (r *GormRepository) BulkDeleteListings(ctx context.Context, ids []uuid.UUID, hard bool) error {
	if len(ids) == 0 {
		return nil
	}

	query := r.db.WithContext(ctx)
	if hard {
		query = query.Unscoped()
	}

	result := query.Delete(&schema.Listing{}, "id IN ?", ids)
	if result.Error != nil {
		return fmt.Errorf("failed to bulk delete listings: %w", result.Error)
	}

	return nil
}

// UpdateListingStatus updates the status, reason, and audit fields for a listing.
func (r *GormRepository) UpdateListingStatus(ctx context.Context, id uuid.UUID, status schema.ListingStatus, reason string, changedBy *uuid.UUID) error {
	updates := map[string]any{
		"status":            status,
		"change_reason":     reason,
		"status_changed_at": time.Now(),
	}
	if changedBy != nil {
		updates["updated_by"] = *changedBy
	}
	result := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update listing status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// UpdatePublishState toggles publish state and timestamp.
func (r *GormRepository) UpdatePublishState(ctx context.Context, id uuid.UUID, published bool, publishedAt *time.Time) error {
	updates := map[string]any{
		"published": published,
	}
	if published {
		if publishedAt != nil {
			updates["published_at"] = *publishedAt
		} else {
			updates["published_at"] = time.Now()
		}
	} else {
		updates["published_at"] = nil
	}
	result := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update publish state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// SoftDeleteListing marks a listing as deleted.
func (r *GormRepository) SoftDeleteListing(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&schema.Listing{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// HardDeleteListing permanently deletes a listing.
func (r *GormRepository) HardDeleteListing(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&schema.Listing{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to hard delete listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// IncrementView increments the view counter and updates the last viewed timestamp.
func (r *GormRepository) IncrementView(ctx context.Context, id uuid.UUID, viewedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&schema.Listing{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"view_count":     gorm.Expr("view_count + 1"),
			"last_viewed_at": viewedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to increment view count: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}
