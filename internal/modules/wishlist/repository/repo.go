package repository

import (
	"context"
	"errors"
	"fmt"

	"hauslet/internal/modules/wishlist/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateWishlist creates a new wishlist
func (r *WishlistRepositoryImpl) CreateWishlist(ctx context.Context, wishlist *schema.Wishlist) error {
	if wishlist == nil {
		return fmt.Errorf("wishlist cannot be nil")
	}

	if err := r.db.WithContext(ctx).Create(wishlist).Error; err != nil {
		return fmt.Errorf("failed to create wishlist: %w", err)
	}

	return nil
}

// GetWishlistByID retrieves a wishlist by its ID
func (r *WishlistRepositoryImpl) GetWishlistByID(ctx context.Context, id uuid.UUID) (*schema.Wishlist, error) {
	var wishlist schema.Wishlist

	if err := r.db.WithContext(ctx).First(&wishlist, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get wishlist by id: %w", err)
	}

	return &wishlist, nil
}

// GetWishlistsByUserID fetches all lists for a specific user
func (r *WishlistRepositoryImpl) GetWishlistsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*schema.Wishlist, error) {
	var wishlists []*schema.Wishlist

	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&wishlists).Error; err != nil {
		return nil, fmt.Errorf("failed to get wishlists for user %s: %w", userID, err)
	}

	return wishlists, nil
}

// UpdateWishlist updates an existing wishlist
func (r *WishlistRepositoryImpl) UpdateWishlist(ctx context.Context, wishlist *schema.Wishlist) error {
	if wishlist == nil {
		return fmt.Errorf("wishlist cannot be nil")
	}

	result := r.db.WithContext(ctx).Save(wishlist)
	if result.Error != nil {
		return fmt.Errorf("failed to update wishlist: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("wishlist not found: %w", gorm.ErrRecordNotFound)
	}

	return nil
}

// DeleteWishlist performs a hard delete on a wishlist (cascades to items)
func (r *WishlistRepositoryImpl) DeleteWishlist(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&schema.Wishlist{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete wishlist: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("wishlist not found: %w", gorm.ErrRecordNotFound)
	}

	return nil
}

// WishlistExists checks if a wishlist exists by ID
func (r *WishlistRepositoryImpl) WishlistExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).Model(&schema.Wishlist{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check wishlist existence: %w", err)
	}

	return count > 0, nil
}

// AddItem adds a single listing to a wishlist
func (r *WishlistRepositoryImpl) AddItem(ctx context.Context, item *schema.WishlistItem) error {
	if item == nil {
		return fmt.Errorf("wishlist item cannot be nil")
	}

	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		// Check for unique constraint violation (duplicate item)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("item already exists in wishlist: %w", err)
		}
		return fmt.Errorf("failed to add item to wishlist: %w", err)
	}

	return nil
}

// RemoveItem removes a specific listing from a wishlist
func (r *WishlistRepositoryImpl) RemoveItem(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("wishlist_id = ? AND listing_id = ?", wishlistID, listingID).
		Delete(&schema.WishlistItem{})

	if result.Error != nil {
		return fmt.Errorf("failed to remove item from wishlist: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("item not found in wishlist: %w", gorm.ErrRecordNotFound)
	}

	return nil
}

// GetWishlistItems retrieves paginated items for a wishlist
func (r *WishlistRepositoryImpl) GetWishlistItems(ctx context.Context, wishlistID uuid.UUID, limit, offset int) ([]*schema.WishlistItem, error) {
	var items []*schema.WishlistItem

	query := r.db.WithContext(ctx).
		Where("wishlist_id = ?", wishlistID).
		Order("added_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get wishlist items: %w", err)
	}

	return items, nil
}

// IsListingInWishlist checks if a specific listing is in a wishlist
func (r *WishlistRepositoryImpl) IsListingInWishlist(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&schema.WishlistItem{}).
		Where("wishlist_id = ? AND listing_id = ?", wishlistID, listingID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check if listing is in wishlist: %w", err)
	}

	return count > 0, nil
}

// CountItems returns the total number of items in a wishlist
func (r *WishlistRepositoryImpl) CountItems(ctx context.Context, wishlistID uuid.UUID) (int64, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&schema.WishlistItem{}).
		Where("wishlist_id = ?", wishlistID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count wishlist items: %w", err)
	}

	return count, nil
}

func (r *WishlistRepositoryImpl) CopyItemsBulk(ctx context.Context, items []schema.WishlistItem) error {
	if len(items) == 0 {
		return nil
	}

	// GORM Bulk Insert with "On Conflict Do Nothing"
	// This relies on the composite unique index: idx_wishlist_listing
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			// Identify the conflict by the unique columns
			Columns: []clause.Column{
				{Name: "wishlist_id"},
				{Name: "listing_id"},
			},
			// If a conflict occurs (item already exists), do nothing (skip)
			DoNothing: true,
		}).
		Create(&items).Error
}
