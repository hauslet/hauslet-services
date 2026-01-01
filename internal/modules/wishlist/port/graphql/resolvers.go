package graphql

import (
	"context"
	"fmt"
	"log/slog"

	propertydomain "hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/wishlist/domain"
	"hauslet/internal/modules/wishlist/service"
	"hauslet/internal/transport/graph/loaders"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles wishlist-specific GraphQL fields.
type Resolver struct {
	wishlistService service.WishlistService
	log             *slog.Logger
}

func NewResolver(wishlistService service.WishlistService, log *slog.Logger) *Resolver {
	return &Resolver{wishlistService: wishlistService, log: log}
}

// ===========================
// QUERY RESOLVERS
// ===========================

// Wishlist retrieves a wishlist by ID with access control enforced in the service layer.
func (r *Resolver) Wishlist(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
	viewerID := r.optionalViewerID(ctx)
	wishlist, err := r.wishlistService.GetWishlist(ctx, id, viewerID)
	if err != nil {
		r.log.Error("failed to get wishlist", "wishlist_id", id, "error", err)
		return nil, err
	}
	return wishlist, nil
}

// MyWishlists lists wishlists belonging to the authenticated user.
func (r *Resolver) MyWishlists(ctx context.Context, limit *int, offset *int) ([]*domain.Wishlist, error) {
	userID, err := r.requireViewerID(ctx)
	if err != nil {
		return nil, err
	}

	l, o := r.normalizePagination(limit, offset)
	wishlists, err := r.wishlistService.ListUserWishlists(ctx, userID, l, o)
	if err != nil {
		r.log.Error("failed to list wishlists for user", "user_id", userID, "error", err)
		return nil, err
	}
	return wishlists, nil
}

// WishlistItems returns items for a wishlist, respecting privacy rules via the service.
func (r *Resolver) WishlistItems(ctx context.Context, wishlistID uuid.UUID, limit *int, offset *int) ([]*domain.WishlistItem, error) {
	viewerID := r.optionalViewerID(ctx)
	l, o := r.normalizePagination(limit, offset)

	items, err := r.wishlistService.GetItems(ctx, wishlistID, viewerID, l, o)
	if err != nil {
		r.log.Error("failed to list items for wishlist", "wishlist_id", wishlistID, "error", err)
		return nil, err
	}
	return items, nil
}

// IsListingInWishlist checks if a listing exists in a wishlist. Access enforced via wishlist lookup.
func (r *Resolver) IsListingInWishlist(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) (bool, error) {
	viewerID := r.optionalViewerID(ctx)
	if _, err := r.wishlistService.GetWishlist(ctx, wishlistID, viewerID); err != nil {
		return false, err
	}

	exists, err := r.wishlistService.IsListed(ctx, wishlistID, listingID)
	if err != nil {
		r.log.Error("failed to check listing in wishlist", "listing_id", listingID, "wishlist_id", wishlistID, "error", err)
		return false, err
	}
	return exists, nil
}

// ===========================
// MUTATION RESOLVERS
// ===========================

// CreateWishlist creates a new wishlist for the authenticated user.
func (r *Resolver) CreateWishlist(ctx context.Context, input model.CreateWishlistInput) (*domain.Wishlist, error) {
	userID, err := r.requireViewerID(ctx)
	if err != nil {
		return nil, err
	}

	isPrivate := false
	if input.IsPrivate != nil {
		isPrivate = *input.IsPrivate
	}

	wishlist, err := r.wishlistService.CreateWishlist(ctx, userID, input.Name, input.Description, isPrivate)
	if err != nil {
		r.log.Error("failed to create wishlist for user", "user_id", userID, "error", err)
		return nil, err
	}
	return wishlist, nil
}

// UpdateWishlist updates wishlist metadata for the authenticated owner.
func (r *Resolver) UpdateWishlist(ctx context.Context, id uuid.UUID, input model.UpdateWishlistInput) (*domain.Wishlist, error) {
	userID, err := r.requireViewerID(ctx)
	if err != nil {
		return nil, err
	}

	wishlist, err := r.wishlistService.UpdateWishlist(ctx, id, userID, input.Name, input.Description, input.IsPrivate)
	if err != nil {
		r.log.Error("failed to update wishlist", "wishlist_id", id, "user_id", userID, "error", err)
		return nil, err
	}
	return wishlist, nil
}

// DeleteWishlist deletes a wishlist owned by the authenticated user.
func (r *Resolver) DeleteWishlist(ctx context.Context, id uuid.UUID) (bool, error) {
	userID, err := r.requireViewerID(ctx)
	if err != nil {
		return false, err
	}

	if err := r.wishlistService.DeleteWishlist(ctx, id, userID); err != nil {
		r.log.Error("failed to delete wishlist", "wishlist_id", id, "user_id", userID, "error", err)
		return false, err
	}
	return true, nil
}

// AddWishlistItem adds a listing to a wishlist owned by the authenticated user.
func (r *Resolver) AddWishlistItem(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID, source *domain.WishlistItemSource) (*domain.WishlistItem, error) {
	userID, err := r.requireViewerID(ctx)
	if err != nil {
		return nil, err
	}

	src := domain.SourceUserAdded
	if source != nil {
		src = *source
	}

	item, err := r.wishlistService.AddItem(ctx, wishlistID, userID, listingID, src)
	if err != nil {
		r.log.Error("failed to add listing to wishlist", "listing_id", listingID, "wishlist_id", wishlistID, "user_id", userID, "error", err)
		return nil, err
	}
	return item, nil
}

// RemoveWishlistItem removes a listing from a wishlist owned by the authenticated user.
func (r *Resolver) RemoveWishlistItem(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) (bool, error) {
	userID, err := r.requireViewerID(ctx)
	if err != nil {
		return false, err
	}

	if err := r.wishlistService.RemoveItem(ctx, wishlistID, userID, listingID); err != nil {
		r.log.Error("failed to remove listing from wishlist", "listing_id", listingID, "wishlist_id", wishlistID, "user_id", userID, "error", err)
		return false, err
	}
	return true, nil
}

// ImportWishlist clones a source wishlist into a new wishlist for the authenticated user.
func (r *Resolver) ImportWishlist(ctx context.Context, sourceWishlistID uuid.UUID, newName *string) (*domain.Wishlist, error) {
	userID, err := r.requireViewerID(ctx)
	if err != nil {
		return nil, err
	}

	name := ""
	if newName != nil {
		name = *newName
	}

	wishlist, err := r.wishlistService.CloneWishlist(ctx, sourceWishlistID, userID, name)
	if err != nil {
		r.log.Error("failed to import wishlist", "source_wishlist_id", sourceWishlistID, "user_id", userID, "error", err)
		return nil, err
	}
	return wishlist, nil
}

// ===========================
// FIELD RESOLVERS
// ===========================

// Items resolves wishlist items with pagination.
func (r *Resolver) WishlistItemsField(ctx context.Context, obj *domain.Wishlist, limit *int, offset *int) ([]*domain.WishlistItem, error) {
	return r.WishlistItems(ctx, obj.ID, limit, offset)
}

// ItemCount resolves the number of items in a wishlist.
func (r *Resolver) WishlistItemCount(ctx context.Context, obj *domain.Wishlist) (int, error) {
	// Use in-memory count if present to avoid extra call
	if len(obj.Items) > 0 {
		return len(obj.Items), nil
	}

	count, err := r.wishlistService.CountItems(ctx, obj.ID)
	if err != nil {
		r.log.Error("failed to count items for wishlist", "wishlist_id", obj.ID, "error", err)
		return 0, err
	}
	return int(count), nil
}

// Listing resolves the listing for a wishlist item using the loader to avoid N+1 queries.
func (r *Resolver) WishlistItemListing(ctx context.Context, obj *domain.WishlistItem) (*propertydomain.Listing, error) {
	if obj == nil {
		return nil, nil
	}
	if l := loaders.For(ctx); l != nil && l.Listing != nil {
		listing, err := l.Listing.LoadWithMedia(ctx, obj.ListingID)
		if err != nil {
			r.log.Error("failed to load listing for wishlist item", "listing_id", obj.ListingID, "item_id", obj.ID, "error", err)
			return nil, err
		}
		return listing, nil
	}
	return nil, fmt.Errorf("listing loader unavailable")
}

// ===========================
// HELPERS
// ===========================

func (r *Resolver) normalizePagination(limit *int, offset *int) (int, int) {
	l := 20
	o := 0
	if limit != nil && *limit >= 0 {
		l = *limit
	}
	if offset != nil && *offset > 0 {
		o = *offset
	}
	return l, o
}

func (r *Resolver) optionalViewerID(ctx context.Context) uuid.UUID {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(v.UserID)
	if err != nil {
		if r.log != nil {
			r.log.Warn("invalid viewer user ID", "user_id", v.UserID, "error", err)
		}
		return uuid.Nil
	}
	return id
}

func (r *Resolver) requireViewerID(ctx context.Context) (uuid.UUID, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return uuid.Nil, fmt.Errorf("unauthenticated")
	}

	id, err := uuid.Parse(v.UserID)
	if err != nil {
		if r.log != nil {
			r.log.Error("invalid viewer user ID", "user_id", v.UserID, "error", err)
		}
		return uuid.Nil, fmt.Errorf("invalid user id")
	}
	return id, nil
}
