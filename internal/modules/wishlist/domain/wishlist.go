package domain

import (
	"time"

	"github.com/google/uuid"
)

// WishlistItemSource represents where an item was added from
type WishlistItemSource string

const (
	SourceUserAdded WishlistItemSource = "user_added"
	SourceImported  WishlistItemSource = "imported"
	SourceRecents   WishlistItemSource = "recents"
)

// Wishlist represents a wishlist in the domain (business logic layer)
type Wishlist struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	IsPrivate   bool           `json:"is_private"`
	Items       []WishlistItem `json:"items,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// WishlistItem represents an item in a wishlist (domain model)
type WishlistItem struct {
	ID         uuid.UUID          `json:"id"`
	WishlistID uuid.UUID          `json:"wishlist_id"`
	ListingID  uuid.UUID          `json:"listing_id"`
	AddedAt    time.Time          `json:"added_at"`
	Source     WishlistItemSource `json:"source"`
}

// IsOwnedBy checks if the wishlist belongs to the given user
func (w *Wishlist) IsOwnedBy(userID uuid.UUID) bool {
	return w.UserID == userID
}

// IsAccessibleBy checks if a user can view this wishlist
// A wishlist is accessible if it's owned by the user or if it's public
func (w *Wishlist) IsAccessibleBy(userID uuid.UUID) bool {
	return w.IsOwnedBy(userID) || !w.IsPrivate
}

// ItemCount returns the number of items in the wishlist
func (w *Wishlist) ItemCount() int {
	return len(w.Items)
}

// HasItem checks if a specific listing is in the wishlist
func (w *Wishlist) HasItem(listingID uuid.UUID) bool {
	for _, item := range w.Items {
		if item.ListingID == listingID {
			return true
		}
	}
	return false
}
