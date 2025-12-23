package service

import (
	"context"
	"hauslet/internal/modules/wishlist/domain"

	"github.com/google/uuid"
)

type WishlistService interface {
	// --- Lifecycle & Management ---

	// CreateWishlist initializes a new list for a user.
	// If name is empty, it should handle default naming (e.g., "New Wishlist").
	CreateWishlist(ctx context.Context, userID uuid.UUID, name string, description *string, isPrivate bool) (*domain.Wishlist, error)

	// GetWishlist retrieves a specific list.
	// It must check if the user has permission to view it (Owner or !IsPrivate).
	GetWishlist(ctx context.Context, wishlistID uuid.UUID, viewerID uuid.UUID) (*domain.Wishlist, error)

	// ListUserWishlists returns all lists belonging to a user (e.g., for the "Saved" tab).
	ListUserWishlists(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Wishlist, error)

	// UpdateWishlist modifies metadata (Name, Description, Privacy).
	// Only the owner can update a wishlist.
	UpdateWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, name *string, description *string, isPrivate *bool) (*domain.Wishlist, error)

	// DeleteWishlist removes the list and cascades the delete to all items.
	// Only the owner can delete a wishlist.
	DeleteWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID) error

	// --- Item Operations ---

	// AddItem adds a listing to a wishlist.
	// Returns the created WishlistItem.
	AddItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID, source domain.WishlistItemSource) (*domain.WishlistItem, error)

	// RemoveItem removes a listing from a wishlist.
	// Only the owner can remove items.
	RemoveItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID) error

	// GetItems retrieves the actual contents of a wishlist with pagination.
	// Viewer must have permission to view the wishlist.
	GetItems(ctx context.Context, wishlistID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]*domain.WishlistItem, error)

	// IsListed checks if a listing is in a specific wishlist (for UI state).
	IsListed(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) (bool, error)

	// CountItems returns the total number of items in a wishlist.
	CountItems(ctx context.Context, wishlistID uuid.UUID) (int64, error)

	// --- Sharing & Importing ---

	// ImportItems copies specific items from a source wishlist to a target wishlist.
	// If listingIDs is empty, imports all items from the source.
	// Returns the number of items imported and skipped (duplicates).
	ImportItems(ctx context.Context, targetWishlistID uuid.UUID, userID uuid.UUID, sourceWishlistID uuid.UUID, listingIDs []uuid.UUID) (imported int, skipped int, err error)

	// CloneWishlist creates a new wishlist for 'targetUserID' and copies all items
	// from 'sourceWishlistID' into it. (The "Snapshot" feature).
	// The source wishlist must be accessible to the target user.
	CloneWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetUserID uuid.UUID, newName string) (*domain.Wishlist, error)

	// MergeWishlist copies items from 'sourceWishlistID' into an *existing* 'targetWishlistID'.
	// Useful if a user wants to combine a shared list into their main list.
	// Returns the number of items merged.
	MergeWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetWishlistID uuid.UUID, userID uuid.UUID) (int, error)
}
