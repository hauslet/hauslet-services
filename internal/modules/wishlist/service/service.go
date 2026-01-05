package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/wishlist/domain"

	"github.com/google/uuid"
)

// CreateWishlist initializes a new list for a user.
func (s *WishlistServiceImpl) CreateWishlist(ctx context.Context, userID uuid.UUID, name string, description *string, isPrivate bool) (*domain.Wishlist, error) {
	// Handle default naming if name is empty
	if name == "" {
		name = "New Wishlist"
	}

	if s.log != nil {
		s.log.Info("creating wishlist", "user_id", userID, "name", name, "private", isPrivate)
	}

	// Create schema wishlist
	schemaWishlist := domain.CreateSchemaWishlist(userID, name, description, isPrivate)

	// Save to repository
	if err := s.repo.CreateWishlist(ctx, schemaWishlist); err != nil {
		if s.log != nil {
			s.log.Error("failed to create wishlist", "user_id", userID, "error", err)
		}
		return nil, fmt.Errorf("failed to create wishlist: %w", err)
	}

	// Convert to domain entity
	wishlist := domain.MapWishlistFromSchemaToEntity(schemaWishlist)

	// Cache the new wishlist
	s.cacheWishlist(ctx, wishlist)
	s.invalidateUserWishlistsCache(ctx, userID)

	if s.log != nil {
		s.log.Info("created wishlist", "id", wishlist.ID, "user_id", userID)
	}

	return wishlist, nil
}

// GetWishlist retrieves a specific list with permission checks.
func (s *WishlistServiceImpl) GetWishlist(ctx context.Context, wishlistID uuid.UUID, viewerID uuid.UUID) (*domain.Wishlist, error) {
	// Try cache first
	var wishlist *domain.Wishlist
	cached, err := s.getCachedValue(ctx, wishlistCacheKey(wishlistID), &wishlist)
	if err == nil && cached && wishlist != nil {
		if s.log != nil {
			s.log.Debug("cache hit for wishlist", "wishlist_id", wishlistID)
		}
		// Permission check: viewer must be owner OR wishlist must be public
		if !wishlist.IsAccessibleBy(viewerID) {
			if s.log != nil {
				s.log.Warn("access denied to wishlist", "viewer_id", viewerID, "wishlist_id", wishlistID, "reason", "private")
			}
			return nil, fmt.Errorf("access denied: wishlist is private")
		}
		return wishlist, nil
	}

	if s.log != nil {
		s.log.Debug("cache miss for wishlist, fetching from DB", "wishlist_id", wishlistID)
	}

	// Fetch from repository
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get wishlist", "wishlist_id", wishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Warn("wishlist not found", "wishlist_id", wishlistID)
		}
		return nil, fmt.Errorf("wishlist not found")
	}

	// Convert to domain entity
	wishlist = domain.MapWishlistFromSchemaToEntity(schemaWishlist)

	// Cache for future requests
	s.cacheWishlist(ctx, wishlist)

	// Permission check: viewer must be owner OR wishlist must be public
	if !wishlist.IsAccessibleBy(viewerID) {
		if s.log != nil {
			s.log.Warn("access denied to wishlist", "viewer_id", viewerID, "wishlist_id", wishlistID, "reason", "private")
		}
		return nil, fmt.Errorf("access denied: wishlist is private")
	}

	return wishlist, nil
}

// ListUserWishlists returns all lists belonging to a user.
func (s *WishlistServiceImpl) ListUserWishlists(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Wishlist, error) {
	// Try cache first
	var wishlists []*domain.Wishlist
	cached, err := s.getCachedValue(ctx, userWishlistsCacheKey(userID, limit, offset), &wishlists)
	if err == nil && cached {
		if s.log != nil {
			s.log.Debug("cache hit for user wishlists", "user_id", userID, "limit", limit, "offset", offset)
		}
		return wishlists, nil
	}

	if s.log != nil {
		s.log.Debug("cache miss for user wishlists", "user_id", userID)
	}

	// Fetch from repository
	schemaWishlists, err := s.repo.GetWishlistsByUserID(ctx, userID, limit, offset)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to list wishlists", "user_id", userID, "error", err)
		}
		return nil, fmt.Errorf("failed to list wishlists: %w", err)
	}

	// Convert to domain entities
	wishlists = domain.MapWishlistsFromSchemaToEntities(schemaWishlists)

	// Cache for future requests
	s.cacheUserWishlists(ctx, userID, wishlists, limit, offset)

	if s.log != nil {
		s.log.Info("retrieved wishlists", "count", len(wishlists), "user_id", userID)
	}

	return wishlists, nil
}

// UpdateWishlist modifies metadata (Name, Description, Privacy).
func (s *WishlistServiceImpl) UpdateWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, name *string, description *string, isPrivate *bool) (*domain.Wishlist, error) {
	if s.log != nil {
		s.log.Info("updating wishlist", "wishlist_id", wishlistID, "user_id", userID)
	}

	// Fetch existing wishlist
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get wishlist for update", "wishlist_id", wishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Warn("wishlist not found for update", "wishlist_id", wishlistID)
		}
		return nil, fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can update
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Warn("unauthorized update attempt", "wishlist_id", wishlistID, "user_id", userID, "owner_id", schemaWishlist.UserID)
		}
		return nil, fmt.Errorf("access denied: only owner can update wishlist")
	}

	// Apply updates
	domain.ApplyUpdateToSchemaWishlist(schemaWishlist, name, description, isPrivate)

	// Save changes
	if err := s.repo.UpdateWishlist(ctx, schemaWishlist); err != nil {
		if s.log != nil {
			s.log.Error("failed to save wishlist update", "wishlist_id", wishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to update wishlist: %w", err)
	}

	// Convert to domain entity
	wishlist := domain.MapWishlistFromSchemaToEntity(schemaWishlist)

	// Invalidate cache
	s.invalidateWishlistAndRelated(ctx, wishlistID, userID)

	if s.log != nil {
		s.log.Info("updated wishlist", "wishlist_id", wishlistID)
	}

	return wishlist, nil
}

// DeleteWishlist removes the list and cascades the delete to all items.
func (s *WishlistServiceImpl) DeleteWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID) error {
	if s.log != nil {
		s.log.Info("deleting wishlist", "wishlist_id", wishlistID, "user_id", userID)
	}

	// Fetch existing wishlist to check ownership
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get wishlist for deletion", "wishlist_id", wishlistID, "error", err)
		}
		return fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Warn("wishlist not found for deletion", "wishlist_id", wishlistID)
		}
		return fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can delete
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Warn("unauthorized delete attempt", "wishlist_id", wishlistID, "user_id", userID, "owner_id", schemaWishlist.UserID)
		}
		return fmt.Errorf("access denied: only owner can delete wishlist")
	}

	// Delete from repository (cascades to items)
	if err := s.repo.DeleteWishlist(ctx, wishlistID); err != nil {
		if s.log != nil {
			s.log.Error("failed to delete wishlist", "wishlist_id", wishlistID, "error", err)
		}
		return fmt.Errorf("failed to delete wishlist: %w", err)
	}

	// Invalidate all related cache
	s.invalidateWishlistAndRelated(ctx, wishlistID, userID)

	if s.log != nil {
		s.log.Info("deleted wishlist", "wishlist_id", wishlistID)
	}

	return nil
}

// AddItem adds a listing to a wishlist.
func (s *WishlistServiceImpl) AddItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID, source domain.WishlistItemSource) (*domain.WishlistItem, error) {
	if s.log != nil {
		s.log.Info("adding item to wishlist", "listing_id", listingID, "wishlist_id", wishlistID, "user_id", userID, "source", source)
	}

	// Check if listing can be added via hooks
	canAdd, err := s.listingHooks.CanAddListingToWishlist(ctx, listingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("listing hook check failed", "listing_id", listingID, "error", err)
		}
		return nil, err
	}
	if !canAdd {
		if s.log != nil {
			s.log.Warn("listing cannot be added to wishlists", "listing_id", listingID)
		}
		return nil, fmt.Errorf("listing cannot be added to wishlist")
	}

	// Fetch wishlist to check ownership
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get wishlist for add item", "wishlist_id", wishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Warn("wishlist not found for add item", "wishlist_id", wishlistID)
		}
		return nil, fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can add items
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Warn("unauthorized add item attempt", "wishlist_id", wishlistID, "user_id", userID, "owner_id", schemaWishlist.UserID)
		}
		return nil, fmt.Errorf("access denied: only owner can add items")
	}

	// Create schema item
	schemaItem := domain.CreateSchemaWishlistItem(wishlistID, listingID, source)

	// Add to repository
	if err := s.repo.AddItem(ctx, schemaItem); err != nil {
		if s.log != nil {
			s.log.Error("failed to add item", "listing_id", listingID, "wishlist_id", wishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	// Invalidate cache
	s.invalidateWishlistItemsCache(ctx, wishlistID)
	s.invalidateListingExistsCache(ctx, wishlistID, listingID)

	if s.log != nil {
		s.log.Info("added item to wishlist", "item_id", schemaItem.ID, "wishlist_id", wishlistID)
	}

	// Convert to domain entity and return
	return domain.MapWishlistItemFromSchemaToEntity(schemaItem), nil
}

// RemoveItem removes a listing from a wishlist.
func (s *WishlistServiceImpl) RemoveItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID) error {
	if s.log != nil {
		s.log.Info("removing item from wishlist", "listing_id", listingID, "wishlist_id", wishlistID, "user_id", userID)
	}

	// Fetch wishlist to check ownership
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get wishlist for remove item", "wishlist_id", wishlistID, "error", err)
		}
		return fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Warn("wishlist not found for remove item", "wishlist_id", wishlistID)
		}
		return fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can remove items
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Warn("unauthorized remove item attempt", "wishlist_id", wishlistID, "user_id", userID, "owner_id", schemaWishlist.UserID)
		}
		return fmt.Errorf("access denied: only owner can remove items")
	}

	// Remove from repository
	if err := s.repo.RemoveItem(ctx, wishlistID, listingID); err != nil {
		if s.log != nil {
			s.log.Error("failed to remove item", "listing_id", listingID, "wishlist_id", wishlistID, "error", err)
		}
		return fmt.Errorf("failed to remove item: %w", err)
	}

	// Invalidate cache
	s.invalidateWishlistItemsCache(ctx, wishlistID)
	s.invalidateListingExistsCache(ctx, wishlistID, listingID)

	if s.log != nil {
		s.log.Info("removed item from wishlist", "listing_id", listingID, "wishlist_id", wishlistID)
	}

	return nil
}

// GetItems retrieves the actual contents of a wishlist with pagination.
func (s *WishlistServiceImpl) GetItems(ctx context.Context, wishlistID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]*domain.WishlistItem, error) {
	// Fetch wishlist to check permissions (not cached to ensure fresh permission check)
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get wishlist for items fetch", "wishlist_id", wishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Warn("wishlist not found for items fetch", "wishlist_id", wishlistID)
		}
		return nil, fmt.Errorf("wishlist not found")
	}

	// Convert to domain entity for permission check
	wishlist := domain.MapWishlistFromSchemaToEntity(schemaWishlist)

	// Permission check: viewer must have access
	if !wishlist.IsAccessibleBy(viewerID) {
		if s.log != nil {
			s.log.Warn("access denied to wishlist items", "viewer_id", viewerID, "wishlist_id", wishlistID, "reason", "private")
		}
		return nil, fmt.Errorf("access denied: wishlist is private")
	}

	// Try cache first
	var items []*domain.WishlistItem
	cached, err := s.getCachedValue(ctx, wishlistItemsCacheKey(wishlistID, limit, offset), &items)
	if err == nil && cached {
		if s.log != nil {
			s.log.Debug("cache hit for wishlist items", "wishlist_id", wishlistID, "limit", limit, "offset", offset)
		}
		return items, nil
	}

	if s.log != nil {
		s.log.Debug("cache miss for wishlist items", "wishlist_id", wishlistID)
	}

	// Fetch items from repository
	schemaItems, err := s.repo.GetWishlistItems(ctx, wishlistID, limit, offset)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get wishlist items", "wishlist_id", wishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to get wishlist items: %w", err)
	}

	// Convert to domain entities
	items = domain.MapWishlistItemsFromSchemaToEntities(schemaItems)

	// Cache for future requests
	s.cacheWishlistItems(ctx, wishlistID, items, limit, offset)

	if s.log != nil {
		s.log.Info("retrieved wishlist items", "count", len(items), "wishlist_id", wishlistID)
	}

	return items, nil
}

// IsListed checks if a listing is in a specific wishlist (for UI state).
func (s *WishlistServiceImpl) IsListed(ctx context.Context, wishlistID uuid.UUID, listingID uuid.UUID) (bool, error) {
	// Try cache first
	var exists bool
	cached, err := s.getCachedValue(ctx, listingInWishlistCacheKey(wishlistID, listingID), &exists)
	if err == nil && cached {
		if s.log != nil {
			s.log.Debug("cache hit for listing check", "wishlist_id", wishlistID, "listing_id", listingID)
		}
		return exists, nil
	}

	// Check repository
	exists, err = s.repo.IsListingInWishlist(ctx, wishlistID, listingID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to check listing", "wishlist_id", wishlistID, "listing_id", listingID, "error", err)
		}
		return false, fmt.Errorf("failed to check if listing is in wishlist: %w", err)
	}

	// Cache the result
	s.cacheListingExists(ctx, wishlistID, listingID, exists)

	return exists, nil
}

// CountItems returns the total number of items in a wishlist.
func (s *WishlistServiceImpl) CountItems(ctx context.Context, wishlistID uuid.UUID) (int64, error) {
	// Try cache first
	var count int64
	cached, err := s.getCachedValue(ctx, wishlistItemCountCacheKey(wishlistID), &count)
	if err == nil && cached {
		if s.log != nil {
			s.log.Debug("cache hit for item count", "wishlist_id", wishlistID)
		}
		return count, nil
	}

	// Fetch from repository
	count, err = s.repo.CountItems(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to count items", "wishlist_id", wishlistID, "error", err)
		}
		return 0, fmt.Errorf("failed to count items: %w", err)
	}

	// Cache the count
	s.cacheItemCount(ctx, wishlistID, count)

	return count, nil
}
