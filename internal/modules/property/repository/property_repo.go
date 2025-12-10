package repository

import (
	"context"
	"errors"
	"fmt"

	"hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormRepository implements PropertyRepository and ListingRepository.
type GormRepository struct {
	db *gorm.DB
}

// NewPropertyRepository constructs a Repository backed by GORM.
func NewPropertyRepository(db *gorm.DB) Repository {
	return &GormRepository{db: db}
}

// CreateProperty inserts a new property record.
func (r *GormRepository) CreateProperty(ctx context.Context, property *schema.Property) error {
	if err := r.db.WithContext(ctx).Create(property).Error; err != nil {
		return fmt.Errorf("failed to create property: %w", err)
	}
	return nil
}

// CreatePropertyTx inserts a new property record within a transaction.
func (r *GormRepository) CreatePropertyTx(ctx context.Context, tx *gorm.DB, property *schema.Property) error {
	if err := tx.WithContext(ctx).Create(property).Error; err != nil {
		return fmt.Errorf("failed to create property: %w", err)
	}
	return nil
}

// UpdateProperty updates all fields on an existing property.
func (r *GormRepository) UpdateProperty(ctx context.Context, property *schema.Property) error {
	result := r.db.WithContext(ctx).Save(property)
	if result.Error != nil {
		return fmt.Errorf("failed to update property: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("property not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// UpdatePropertyTx updates all fields on an existing property within a transaction.
func (r *GormRepository) UpdatePropertyTx(ctx context.Context, tx *gorm.DB, property *schema.Property) error {
	result := tx.WithContext(ctx).Save(property)
	if result.Error != nil {
		return fmt.Errorf("failed to update property: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("property not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// PatchProperty applies partial updates to a property by ID.
func (r *GormRepository) PatchProperty(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&schema.Property{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to patch property: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("property not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// PatchPropertyTx applies partial updates to a property by ID within a transaction.
func (r *GormRepository) PatchPropertyTx(ctx context.Context, tx *gorm.DB, id uuid.UUID, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := tx.WithContext(ctx).
		Model(&schema.Property{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to patch property: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("property not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// GetPropertyByID fetches a property by primary key.
func (r *GormRepository) GetPropertyByID(ctx context.Context, id uuid.UUID) (*schema.Property, error) {
	var property schema.Property
	if err := r.db.WithContext(ctx).First(&property, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("property not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get property: %w", err)
	}
	return &property, nil
}

// GetPropertyByPublicID fetches a property by public ID.
func (r *GormRepository) GetPropertyByPublicID(ctx context.Context, publicID string) (*schema.Property, error) {
	var property schema.Property
	if err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&property).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("property not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get property: %w", err)
	}
	return &property, nil
}

// ListProperties returns properties that match the provided filter and pagination.
func (r *GormRepository) ListProperties(ctx context.Context, filter PropertyFilter, page Pagination) (*PaginatedResult[schema.Property], error) {
	var properties []schema.Property
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).Model(&schema.Property{})
	if !filter.IncludeDeleted {
		baseQuery = baseQuery.Where("deleted_at IS NULL")
	}
	baseQuery = applyPropertyFilter(baseQuery, filter)

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count properties: %w", err)
	}

	// Apply sorting and pagination
	query := baseQuery
	query = applyPropertySort(query, filter.SortBy, filter.SortOrder)
	query = applyPagination(query, page)

	if err := query.Find(&properties).Error; err != nil {
		return nil, fmt.Errorf("failed to list properties: %w", err)
	}

	return &PaginatedResult[schema.Property]{
		Items:      properties,
		TotalCount: totalCount,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}

// CountProperties returns the count of properties matching the filter.
func (r *GormRepository) CountProperties(ctx context.Context, filter PropertyFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&schema.Property{})
	if !filter.IncludeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	query = applyPropertyFilter(query, filter)

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count properties: %w", err)
	}
	return count, nil
}

// PropertyExists checks if a property exists by ID.
func (r *GormRepository) PropertyExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&schema.Property{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check property existence: %w", err)
	}
	return count > 0, nil
}

// GetPropertiesByIDs fetches multiple properties by their IDs.
func (r *GormRepository) GetPropertiesByIDs(ctx context.Context, ids []uuid.UUID) ([]schema.Property, error) {
	if len(ids) == 0 {
		return []schema.Property{}, nil
	}

	var properties []schema.Property
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&properties).Error; err != nil {
		return nil, fmt.Errorf("failed to get properties by IDs: %w", err)
	}

	return properties, nil
}

// BulkCreateProperties inserts multiple properties in batches.
func (r *GormRepository) BulkCreateProperties(ctx context.Context, properties []schema.Property) error {
	if len(properties) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).CreateInBatches(properties, BatchInsertSize).Error; err != nil {
		return fmt.Errorf("failed to bulk create properties: %w", err)
	}
	return nil
}

// BulkDeleteProperties deletes multiple properties by IDs.
func (r *GormRepository) BulkDeleteProperties(ctx context.Context, ids []uuid.UUID, hard bool) error {
	if len(ids) == 0 {
		return nil
	}

	query := r.db.WithContext(ctx)
	if hard {
		query = query.Unscoped()
	}

	result := query.Delete(&schema.Property{}, "id IN ?", ids)
	if result.Error != nil {
		return fmt.Errorf("failed to bulk delete properties: %w", result.Error)
	}

	return nil
}

// SoftDeleteProperty marks a property as deleted.
func (r *GormRepository) SoftDeleteProperty(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&schema.Property{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete property: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("property not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// HardDeleteProperty permanently deletes a property.
func (r *GormRepository) HardDeleteProperty(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&schema.Property{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to hard delete property: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("property not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// Transaction executes a function within a database transaction.
func (r *GormRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
