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
		s.log.Logf("INFO creating wishlist for user=%s name=%s private=%t", userID, name, isPrivate)
	}

	// Create schema wishlist
	schemaWishlist := domain.CreateSchemaWishlist(userID, name, description, isPrivate)

	// Save to repository
	if err := s.repo.CreateWishlist(ctx, schemaWishlist); err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to create wishlist for user=%s: %v", userID, err)
		}
		return nil, fmt.Errorf("failed to create wishlist: %w", err)
	}

	// Convert to domain entity
	wishlist := domain.MapWishlistFromSchemaToEntity(schemaWishlist)

	// Cache the new wishlist
	s.cacheWishlist(ctx, wishlist)
	s.invalidateUserWishlistsCache(ctx, userID)

	if s.log != nil {
		s.log.Logf("INFO created wishlist id=%s for user=%s", wishlist.ID, userID)
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
			s.log.Logf("DEBUG cache hit for wishlist id=%s", wishlistID)
		}
		// Permission check: viewer must be owner OR wishlist must be public
		if !wishlist.IsAccessibleBy(viewerID) {
			if s.log != nil {
				s.log.Logf("WARN access denied for viewer=%s to wishlist=%s (private)", viewerID, wishlistID)
			}
			return nil, fmt.Errorf("access denied: wishlist is private")
		}
		return wishlist, nil
	}

	if s.log != nil {
		s.log.Logf("DEBUG cache miss for wishlist id=%s, fetching from DB", wishlistID)
	}

	// Fetch from repository
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get wishlist id=%s: %v", wishlistID, err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN wishlist not found id=%s", wishlistID)
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
			s.log.Logf("WARN access denied for viewer=%s to wishlist=%s (private)", viewerID, wishlistID)
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
			s.log.Logf("DEBUG cache hit for user wishlists user=%s limit=%d offset=%d", userID, limit, offset)
		}
		return wishlists, nil
	}

	if s.log != nil {
		s.log.Logf("DEBUG cache miss for user wishlists user=%s, fetching from DB", userID)
	}

	// Fetch from repository
	schemaWishlists, err := s.repo.GetWishlistsByUserID(ctx, userID, limit, offset)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to list wishlists for user=%s: %v", userID, err)
		}
		return nil, fmt.Errorf("failed to list wishlists: %w", err)
	}

	// Convert to domain entities
	wishlists = domain.MapWishlistsFromSchemaToEntities(schemaWishlists)

	// Cache for future requests
	s.cacheUserWishlists(ctx, userID, wishlists, limit, offset)

	if s.log != nil {
		s.log.Logf("INFO retrieved %d wishlists for user=%s", len(wishlists), userID)
	}

	return wishlists, nil
}

// UpdateWishlist modifies metadata (Name, Description, Privacy).
func (s *WishlistServiceImpl) UpdateWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, name *string, description *string, isPrivate *bool) (*domain.Wishlist, error) {
	if s.log != nil {
		s.log.Logf("INFO updating wishlist id=%s by user=%s", wishlistID, userID)
	}

	// Fetch existing wishlist
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get wishlist id=%s for update: %v", wishlistID, err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN wishlist not found for update id=%s", wishlistID)
		}
		return nil, fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can update
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Logf("WARN unauthorized update attempt on wishlist=%s by user=%s (owner=%s)", wishlistID, userID, schemaWishlist.UserID)
		}
		return nil, fmt.Errorf("access denied: only owner can update wishlist")
	}

	// Apply updates
	domain.ApplyUpdateToSchemaWishlist(schemaWishlist, name, description, isPrivate)

	// Save changes
	if err := s.repo.UpdateWishlist(ctx, schemaWishlist); err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to save wishlist update id=%s: %v", wishlistID, err)
		}
		return nil, fmt.Errorf("failed to update wishlist: %w", err)
	}

	// Convert to domain entity
	wishlist := domain.MapWishlistFromSchemaToEntity(schemaWishlist)

	// Invalidate cache
	s.invalidateWishlistAndRelated(ctx, wishlistID, userID)

	if s.log != nil {
		s.log.Logf("INFO updated wishlist id=%s", wishlistID)
	}

	return wishlist, nil
}

// DeleteWishlist removes the list and cascades the delete to all items.
func (s *WishlistServiceImpl) DeleteWishlist(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID) error {
	if s.log != nil {
		s.log.Logf("INFO deleting wishlist id=%s by user=%s", wishlistID, userID)
	}

	// Fetch existing wishlist to check ownership
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get wishlist id=%s for deletion: %v", wishlistID, err)
		}
		return fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN wishlist not found for deletion id=%s", wishlistID)
		}
		return fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can delete
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Logf("WARN unauthorized delete attempt on wishlist=%s by user=%s (owner=%s)", wishlistID, userID, schemaWishlist.UserID)
		}
		return fmt.Errorf("access denied: only owner can delete wishlist")
	}

	// Delete from repository (cascades to items)
	if err := s.repo.DeleteWishlist(ctx, wishlistID); err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to delete wishlist id=%s: %v", wishlistID, err)
		}
		return fmt.Errorf("failed to delete wishlist: %w", err)
	}

	// Invalidate all related cache
	s.invalidateWishlistAndRelated(ctx, wishlistID, userID)

	if s.log != nil {
		s.log.Logf("INFO deleted wishlist id=%s", wishlistID)
	}

	return nil
}

// AddItem adds a listing to a wishlist.
func (s *WishlistServiceImpl) AddItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID, source domain.WishlistItemSource) (*domain.WishlistItem, error) {
	if s.log != nil {
		s.log.Logf("INFO adding item listing=%s to wishlist=%s by user=%s source=%s", listingID, wishlistID, userID, source)
	}

	// Check if listing can be added via hooks
	canAdd, err := s.listingHooks.CanAddListingToWishlist(ctx, listingID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR listing hook check failed for listing=%s: %v", listingID, err)
		}
		return nil, err
	}
	if !canAdd {
		if s.log != nil {
			s.log.Logf("WARN listing=%s cannot be added to wishlists per hooks", listingID)
		}
		return nil, fmt.Errorf("listing cannot be added to wishlist")
	}

	// Fetch wishlist to check ownership
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get wishlist id=%s for add item: %v", wishlistID, err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN wishlist not found for add item id=%s", wishlistID)
		}
		return nil, fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can add items
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Logf("WARN unauthorized add item attempt on wishlist=%s by user=%s (owner=%s)", wishlistID, userID, schemaWishlist.UserID)
		}
		return nil, fmt.Errorf("access denied: only owner can add items")
	}

	// Create schema item
	schemaItem := domain.CreateSchemaWishlistItem(wishlistID, listingID, source)

	// Add to repository
	if err := s.repo.AddItem(ctx, schemaItem); err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to add item listing=%s to wishlist=%s: %v", listingID, wishlistID, err)
		}
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	// Invalidate cache
	s.invalidateWishlistItemsCache(ctx, wishlistID)
	s.invalidateListingExistsCache(ctx, wishlistID, listingID)

	if s.log != nil {
		s.log.Logf("INFO added item id=%s to wishlist=%s", schemaItem.ID, wishlistID)
	}

	// Convert to domain entity and return
	return domain.MapWishlistItemFromSchemaToEntity(schemaItem), nil
}

// RemoveItem removes a listing from a wishlist.
func (s *WishlistServiceImpl) RemoveItem(ctx context.Context, wishlistID uuid.UUID, userID uuid.UUID, listingID uuid.UUID) error {
	if s.log != nil {
		s.log.Logf("INFO removing item listing=%s from wishlist=%s by user=%s", listingID, wishlistID, userID)
	}

	// Fetch wishlist to check ownership
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get wishlist id=%s for remove item: %v", wishlistID, err)
		}
		return fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN wishlist not found for remove item id=%s", wishlistID)
		}
		return fmt.Errorf("wishlist not found")
	}

	// Permission check: only owner can remove items
	if schemaWishlist.UserID != userID {
		if s.log != nil {
			s.log.Logf("WARN unauthorized remove item attempt on wishlist=%s by user=%s (owner=%s)", wishlistID, userID, schemaWishlist.UserID)
		}
		return fmt.Errorf("access denied: only owner can remove items")
	}

	// Remove from repository
	if err := s.repo.RemoveItem(ctx, wishlistID, listingID); err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to remove item listing=%s from wishlist=%s: %v", listingID, wishlistID, err)
		}
		return fmt.Errorf("failed to remove item: %w", err)
	}

	// Invalidate cache
	s.invalidateWishlistItemsCache(ctx, wishlistID)
	s.invalidateListingExistsCache(ctx, wishlistID, listingID)

	if s.log != nil {
		s.log.Logf("INFO removed item listing=%s from wishlist=%s", listingID, wishlistID)
	}

	return nil
}

// GetItems retrieves the actual contents of a wishlist with pagination.
func (s *WishlistServiceImpl) GetItems(ctx context.Context, wishlistID uuid.UUID, viewerID uuid.UUID, limit, offset int) ([]*domain.WishlistItem, error) {
	// Fetch wishlist to check permissions (not cached to ensure fresh permission check)
	schemaWishlist, err := s.repo.GetWishlistByID(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get wishlist id=%s for items fetch: %v", wishlistID, err)
		}
		return nil, fmt.Errorf("failed to get wishlist: %w", err)
	}

	if schemaWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN wishlist not found for items fetch id=%s", wishlistID)
		}
		return nil, fmt.Errorf("wishlist not found")
	}

	// Convert to domain entity for permission check
	wishlist := domain.MapWishlistFromSchemaToEntity(schemaWishlist)

	// Permission check: viewer must have access
	if !wishlist.IsAccessibleBy(viewerID) {
		if s.log != nil {
			s.log.Logf("WARN access denied for viewer=%s to wishlist=%s items (private)", viewerID, wishlistID)
		}
		return nil, fmt.Errorf("access denied: wishlist is private")
	}

	// Try cache first
	var items []*domain.WishlistItem
	cached, err := s.getCachedValue(ctx, wishlistItemsCacheKey(wishlistID, limit, offset), &items)
	if err == nil && cached {
		if s.log != nil {
			s.log.Logf("DEBUG cache hit for wishlist items id=%s limit=%d offset=%d", wishlistID, limit, offset)
		}
		return items, nil
	}

	if s.log != nil {
		s.log.Logf("DEBUG cache miss for wishlist items id=%s, fetching from DB", wishlistID)
	}

	// Fetch items from repository
	schemaItems, err := s.repo.GetWishlistItems(ctx, wishlistID, limit, offset)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get wishlist items id=%s: %v", wishlistID, err)
		}
		return nil, fmt.Errorf("failed to get wishlist items: %w", err)
	}

	// Convert to domain entities
	items = domain.MapWishlistItemsFromSchemaToEntities(schemaItems)

	// Cache for future requests
	s.cacheWishlistItems(ctx, wishlistID, items, limit, offset)

	if s.log != nil {
		s.log.Logf("INFO retrieved %d items from wishlist=%s", len(items), wishlistID)
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
			s.log.Logf("DEBUG cache hit for listing check wishlist=%s listing=%s", wishlistID, listingID)
		}
		return exists, nil
	}

	// Check repository
	exists, err = s.repo.IsListingInWishlist(ctx, wishlistID, listingID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to check listing wishlist=%s listing=%s: %v", wishlistID, listingID, err)
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
			s.log.Logf("DEBUG cache hit for item count wishlist=%s", wishlistID)
		}
		return count, nil
	}

	// Fetch from repository
	count, err = s.repo.CountItems(ctx, wishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to count items wishlist=%s: %v", wishlistID, err)
		}
		return 0, fmt.Errorf("failed to count items: %w", err)
	}

	// Cache the count
	s.cacheItemCount(ctx, wishlistID, count)

	return count, nil
}
