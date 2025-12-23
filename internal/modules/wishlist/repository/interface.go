package repository

import (
	"context"
	"hauslet/internal/modules/wishlist/repository/schema" // Assumed path

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WishlistRepository interface {
	// --- Core Wishlist CRUD ---
	CreateWishlist(ctx context.Context, wishlist *schema.Wishlist) error
	GetWishlistByID(ctx context.Context, id uuid.UUID) (*schema.Wishlist, error)

	// GetWishlistsByUserID fetches all lists for a specific user (e.g., "My Lists")
	GetWishlistsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*schema.Wishlist, error)

	UpdateWishlist(ctx context.Context, wishlist *schema.Wishlist) error
	DeleteWishlist(ctx context.Context, id uuid.UUID) error // Hard delete (cascades to items)

	// Check if a wishlist exists (useful for validation before import/sharing)
	WishlistExists(ctx context.Context, id uuid.UUID) (bool, error)

	// --- Item Management (Single) ---
	// AddItem adds a single listing. Implementation handles "ON CONFLICT" errors nicely.
	AddItem(ctx context.Context, item *schema.WishlistItem) error

	// RemoveItem removes a specific listing from a specific wishlist
	RemoveItem(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) error

	// GetWishlistItems retrieves paginated items for a list
	GetWishlistItems(ctx context.Context, wishlistID uuid.UUID, limit, offset int) ([]*schema.WishlistItem, error)

	// IsListingInWishlist checks if a specific listing is already saved (for UI "heart" toggles)
	IsListingInWishlist(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) (bool, error)

	// --- Import & Bulk Operations ---
	// CopyItemsBulk is the engine for the "Import" feature.
	// It should handle "ON CONFLICT DO NOTHING" to skip duplicates safely.
	CopyItemsBulk(ctx context.Context, items []schema.WishlistItem) error

	// CountItems helps with limits (e.g., "Max 50 items per list")
	CountItems(ctx context.Context, wishlistID uuid.UUID) (int64, error)
}

type WishlistRepositoryImpl struct {
	db *gorm.DB
}

func NewWishlistRepository(db *gorm.DB) WishlistRepository {
	return &WishlistRepositoryImpl{db: db}
}
