package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/wishlist/domain"

	"github.com/google/uuid"
)

// ImportItems copies specific items from a source wishlist to a target wishlist.
func (s *WishlistServiceImpl) ImportItems(ctx context.Context, targetWishlistID uuid.UUID, userID uuid.UUID, sourceWishlistID uuid.UUID, listingIDs []uuid.UUID) (imported int, skipped int, err error) {
	if s.log != nil {
		s.log.Info("importing items", "source_wishlist_id", sourceWishlistID, "target_wishlist_id", targetWishlistID, "user_id", userID)
	}

	// Verify target wishlist exists and user owns it
	targetWishlist, err := s.repo.GetWishlistByID(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get target wishlist for import", "wishlist_id", targetWishlistID, "error", err)
		}
		return 0, 0, fmt.Errorf("failed to get target wishlist: %w", err)
	}

	if targetWishlist == nil {
		if s.log != nil {
			s.log.Warn("target wishlist not found", "wishlist_id", targetWishlistID)
		}
		return 0, 0, fmt.Errorf("target wishlist not found")
	}

	if targetWishlist.UserID != userID {
		if s.log != nil {
			s.log.Warn("unauthorized import attempt", "wishlist_id", targetWishlistID, "user_id", userID, "owner_id", targetWishlist.UserID)
		}
		return 0, 0, fmt.Errorf("access denied: only owner can import items")
	}

	// Verify source wishlist exists
	sourceWishlist, err := s.repo.GetWishlistByID(ctx, sourceWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get source wishlist", "wishlist_id", sourceWishlistID, "error", err)
		}
		return 0, 0, fmt.Errorf("failed to get source wishlist: %w", err)
	}

	if sourceWishlist == nil {
		if s.log != nil {
			s.log.Warn("source wishlist not found", "wishlist_id", sourceWishlistID)
		}
		return 0, 0, fmt.Errorf("source wishlist not found")
	}

	// Convert to domain entity for permission check
	sourceDomain := domain.MapWishlistFromSchemaToEntity(sourceWishlist)
	if !sourceDomain.IsAccessibleBy(userID) {
		if s.log != nil {
			s.log.Warn("access denied to source wishlist", "wishlist_id", sourceWishlistID, "user_id", userID, "reason", "private")
		}
		return 0, 0, fmt.Errorf("access denied: source wishlist is private")
	}

	// Get items from source wishlist
	var sourceItems []*domain.WishlistItem
	if len(listingIDs) == 0 {
		// Import all items
		schemaItems, err := s.repo.GetWishlistItems(ctx, sourceWishlistID, 0, 0)
		if err != nil {
			if s.log != nil {
				s.log.Error("failed to get source items", "wishlist_id", sourceWishlistID, "error", err)
			}
			return 0, 0, fmt.Errorf("failed to get source items: %w", err)
		}
		sourceItems = domain.MapWishlistItemsFromSchemaToEntities(schemaItems)
	} else {
		// Import specific items
		sourceItems = make([]*domain.WishlistItem, 0, len(listingIDs))
		for _, listingID := range listingIDs {
			// Verify each listing is in the source wishlist
			exists, err := s.repo.IsListingInWishlist(ctx, sourceWishlistID, listingID)
			if err != nil {
				if s.log != nil {
					s.log.Error("failed to check listing", "listing_id", listingID, "wishlist_id", sourceWishlistID, "error", err)
				}
				return 0, 0, fmt.Errorf("failed to check listing: %w", err)
			}
			if exists {
				sourceItems = append(sourceItems, &domain.WishlistItem{
					ListingID: listingID,
					Source:    domain.SourceImported,
				})
			}
		}
	}

	if len(sourceItems) == 0 {
		if s.log != nil {
			s.log.Info("no items to import", "source_wishlist_id", sourceWishlistID, "target_wishlist_id", targetWishlistID)
		}
		return 0, 0, nil
	}

	// Count existing items before import
	beforeCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to count items before import", "error", err)
		}
		return 0, 0, fmt.Errorf("failed to count items before import: %w", err)
	}

	// Create bulk items for import
	bulkItems := domain.CreateBulkSchemaWishlistItems(
		targetWishlistID,
		extractListingIDs(sourceItems),
		domain.SourceImported,
	)

	// Perform bulk insert (duplicates are skipped via ON CONFLICT DO NOTHING)
	if err := s.repo.CopyItemsBulk(ctx, bulkItems); err != nil {
		if s.log != nil {
			s.log.Error("failed to bulk import items", "wishlist_id", targetWishlistID, "error", err)
		}
		return 0, 0, fmt.Errorf("failed to import items: %w", err)
	}

	// Count items after import to calculate imported vs skipped
	afterCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to count items after import", "error", err)
		}
		return 0, 0, fmt.Errorf("failed to count items after import: %w", err)
	}

	imported = int(afterCount - beforeCount)
	skipped = len(sourceItems) - imported

	// Invalidate cache for target wishlist
	s.invalidateWishlistItemsCache(ctx, targetWishlistID)

	if s.log != nil {
		s.log.Info("imported items", "imported", imported, "skipped", skipped, "source_wishlist_id", sourceWishlistID, "target_wishlist_id", targetWishlistID)
	}

	return imported, skipped, nil
}

// CloneWishlist creates a new wishlist for targetUserID and copies all items from sourceWishlistID.
func (s *WishlistServiceImpl) CloneWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetUserID uuid.UUID, newName string) (*domain.Wishlist, error) {
	if s.log != nil {
		s.log.Info("cloning wishlist", "source_wishlist_id", sourceWishlistID, "target_user_id", targetUserID)
	}

	// Fetch source wishlist
	sourceWishlist, err := s.repo.GetWishlistByID(ctx, sourceWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get source wishlist for clone", "wishlist_id", sourceWishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to get source wishlist: %w", err)
	}

	if sourceWishlist == nil {
		if s.log != nil {
			s.log.Warn("source wishlist not found for clone", "wishlist_id", sourceWishlistID)
		}
		return nil, fmt.Errorf("source wishlist not found")
	}

	// Convert to domain entity for permission check
	sourceDomain := domain.MapWishlistFromSchemaToEntity(sourceWishlist)
	if !sourceDomain.IsAccessibleBy(targetUserID) {
		if s.log != nil {
			s.log.Warn("access denied to clone wishlist", "wishlist_id", sourceWishlistID, "user_id", targetUserID, "reason", "private")
		}
		return nil, fmt.Errorf("access denied: source wishlist is private")
	}

	// Use source name if no new name provided
	if newName == "" {
		newName = sourceWishlist.Name + " (Copy)"
	}

	// Create new wishlist for target user
	newWishlist := domain.CreateSchemaWishlist(
		targetUserID,
		newName,
		sourceWishlist.Description,
		false, // Cloned wishlists are public by default
	)

	if err := s.repo.CreateWishlist(ctx, newWishlist); err != nil {
		if s.log != nil {
			s.log.Error("failed to create cloned wishlist", "user_id", targetUserID, "error", err)
		}
		return nil, fmt.Errorf("failed to create cloned wishlist: %w", err)
	}

	// Get all items from source wishlist
	sourceItems, err := s.repo.GetWishlistItems(ctx, sourceWishlistID, 0, 0)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get items from source wishlist for clone", "wishlist_id", sourceWishlistID, "error", err)
		}
		return nil, fmt.Errorf("failed to get source items: %w", err)
	}

	// Copy items if source has any
	if len(sourceItems) > 0 {
		bulkItems := domain.MapItemsForImport(sourceItems, newWishlist.ID)
		if err := s.repo.CopyItemsBulk(ctx, bulkItems); err != nil {
			if s.log != nil {
				s.log.Error("failed to copy items to cloned wishlist", "wishlist_id", newWishlist.ID, "error", err)
			}
			return nil, fmt.Errorf("failed to copy items: %w", err)
		}
		if s.log != nil {
			s.log.Info("copied items to cloned wishlist", "count", len(sourceItems), "wishlist_id", newWishlist.ID)
		}
	}

	// Fetch the complete new wishlist with items
	clonedWishlist, err := s.repo.GetWishlistByID(ctx, newWishlist.ID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to fetch cloned wishlist", "wishlist_id", newWishlist.ID, "error", err)
		}
		return nil, fmt.Errorf("failed to fetch cloned wishlist: %w", err)
	}

	wishlist := domain.MapWishlistFromSchemaToEntity(clonedWishlist)

	// Cache the new wishlist and invalidate user's list cache
	s.cacheWishlist(ctx, wishlist)
	s.invalidateUserWishlistsCache(ctx, targetUserID)

	if s.log != nil {
		s.log.Info("cloned wishlist", "source_wishlist_id", sourceWishlistID, "new_wishlist_id", newWishlist.ID, "target_user_id", targetUserID)
	}

	return wishlist, nil
}

// MergeWishlist copies items from sourceWishlistID into an existing targetWishlistID.
func (s *WishlistServiceImpl) MergeWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetWishlistID uuid.UUID, userID uuid.UUID) (int, error) {
	if s.log != nil {
		s.log.Info("merging wishlist", "source_wishlist_id", sourceWishlistID, "target_wishlist_id", targetWishlistID, "user_id", userID)
	}

	// Verify target wishlist exists and user owns it
	targetWishlist, err := s.repo.GetWishlistByID(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get target wishlist for merge", "wishlist_id", targetWishlistID, "error", err)
		}
		return 0, fmt.Errorf("failed to get target wishlist: %w", err)
	}

	if targetWishlist == nil {
		if s.log != nil {
			s.log.Warn("target wishlist not found for merge", "wishlist_id", targetWishlistID)
		}
		return 0, fmt.Errorf("target wishlist not found")
	}

	if targetWishlist.UserID != userID {
		if s.log != nil {
			s.log.Warn("unauthorized merge attempt", "wishlist_id", targetWishlistID, "user_id", userID, "owner_id", targetWishlist.UserID)
		}
		return 0, fmt.Errorf("access denied: only owner can merge into wishlist")
	}

	// Verify source wishlist exists and is accessible
	sourceWishlist, err := s.repo.GetWishlistByID(ctx, sourceWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get source wishlist for merge", "wishlist_id", sourceWishlistID, "error", err)
		}
		return 0, fmt.Errorf("failed to get source wishlist: %w", err)
	}

	if sourceWishlist == nil {
		if s.log != nil {
			s.log.Warn("source wishlist not found for merge", "wishlist_id", sourceWishlistID)
		}
		return 0, fmt.Errorf("source wishlist not found")
	}

	// Convert to domain entity for permission check
	sourceDomain := domain.MapWishlistFromSchemaToEntity(sourceWishlist)
	if !sourceDomain.IsAccessibleBy(userID) {
		if s.log != nil {
			s.log.Warn("access denied to merge from wishlist", "wishlist_id", sourceWishlistID, "user_id", userID, "reason", "private")
		}
		return 0, fmt.Errorf("access denied: source wishlist is private")
	}

	// Prevent merging a wishlist into itself
	if sourceWishlistID == targetWishlistID {
		if s.log != nil {
			s.log.Warn("attempt to merge wishlist into itself", "wishlist_id", sourceWishlistID)
		}
		return 0, fmt.Errorf("cannot merge a wishlist into itself")
	}

	// Count items before merge
	beforeCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to count items before merge", "error", err)
		}
		return 0, fmt.Errorf("failed to count items before merge: %w", err)
	}

	// Get all items from source wishlist
	sourceItems, err := s.repo.GetWishlistItems(ctx, sourceWishlistID, 0, 0)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get items from source wishlist for merge", "wishlist_id", sourceWishlistID, "error", err)
		}
		return 0, fmt.Errorf("failed to get source items: %w", err)
	}

	if len(sourceItems) == 0 {
		if s.log != nil {
			s.log.Info("no items to merge", "source_wishlist_id", sourceWishlistID)
		}
		return 0, nil // Nothing to merge
	}

	// Prepare items for bulk insert
	bulkItems := domain.MapItemsForImport(sourceItems, targetWishlistID)

	// Perform bulk insert (duplicates are skipped)
	if err := s.repo.CopyItemsBulk(ctx, bulkItems); err != nil {
		if s.log != nil {
			s.log.Error("failed to bulk merge items", "wishlist_id", targetWishlistID, "error", err)
		}
		return 0, fmt.Errorf("failed to merge items: %w", err)
	}

	// Count items after merge to calculate merged count
	afterCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to count items after merge", "error", err)
		}
		return 0, fmt.Errorf("failed to count items after merge: %w", err)
	}

	mergedCount := int(afterCount - beforeCount)

	// Invalidate cache for target wishlist
	s.invalidateWishlistItemsCache(ctx, targetWishlistID)

	if s.log != nil {
		s.log.Info("merged items", "merged", mergedCount, "source_wishlist_id", sourceWishlistID, "target_wishlist_id", targetWishlistID, "skipped", len(sourceItems)-mergedCount)
	}

	return mergedCount, nil
}

// Helper function to extract listing IDs from wishlist items
func extractListingIDs(items []*domain.WishlistItem) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ListingID)
	}
	return ids
}
