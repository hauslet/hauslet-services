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
		s.log.Logf("INFO importing items from source=%s to target=%s by user=%s", sourceWishlistID, targetWishlistID, userID)
	}

	// Verify target wishlist exists and user owns it
	targetWishlist, err := s.repo.GetWishlistByID(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get target wishlist id=%s for import: %v", targetWishlistID, err)
		}
		return 0, 0, fmt.Errorf("failed to get target wishlist: %w", err)
	}

	if targetWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN target wishlist not found id=%s", targetWishlistID)
		}
		return 0, 0, fmt.Errorf("target wishlist not found")
	}

	if targetWishlist.UserID != userID {
		if s.log != nil {
			s.log.Logf("WARN unauthorized import attempt on wishlist=%s by user=%s (owner=%s)", targetWishlistID, userID, targetWishlist.UserID)
		}
		return 0, 0, fmt.Errorf("access denied: only owner can import items")
	}

	// Verify source wishlist exists
	sourceWishlist, err := s.repo.GetWishlistByID(ctx, sourceWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get source wishlist id=%s: %v", sourceWishlistID, err)
		}
		return 0, 0, fmt.Errorf("failed to get source wishlist: %w", err)
	}

	if sourceWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN source wishlist not found id=%s", sourceWishlistID)
		}
		return 0, 0, fmt.Errorf("source wishlist not found")
	}

	// Convert to domain entity for permission check
	sourceDomain := domain.MapWishlistFromSchemaToEntity(sourceWishlist)
	if !sourceDomain.IsAccessibleBy(userID) {
		if s.log != nil {
			s.log.Logf("WARN access denied to source wishlist=%s by user=%s (private)", sourceWishlistID, userID)
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
				s.log.Logf("ERROR failed to get source items from wishlist=%s: %v", sourceWishlistID, err)
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
					s.log.Logf("ERROR failed to check listing=%s in wishlist=%s: %v", listingID, sourceWishlistID, err)
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
			s.log.Logf("INFO no items to import from source=%s to target=%s", sourceWishlistID, targetWishlistID)
		}
		return 0, 0, nil
	}

	// Count existing items before import
	beforeCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to count items before import: %v", err)
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
			s.log.Logf("ERROR failed to bulk import items to wishlist=%s: %v", targetWishlistID, err)
		}
		return 0, 0, fmt.Errorf("failed to import items: %w", err)
	}

	// Count items after import to calculate imported vs skipped
	afterCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to count items after import: %v", err)
		}
		return 0, 0, fmt.Errorf("failed to count items after import: %w", err)
	}

	imported = int(afterCount - beforeCount)
	skipped = len(sourceItems) - imported

	// Invalidate cache for target wishlist
	s.invalidateWishlistItemsCache(ctx, targetWishlistID)

	if s.log != nil {
		s.log.Logf("INFO imported %d items (skipped %d duplicates) from source=%s to target=%s", imported, skipped, sourceWishlistID, targetWishlistID)
	}

	return imported, skipped, nil
}

// CloneWishlist creates a new wishlist for targetUserID and copies all items from sourceWishlistID.
func (s *WishlistServiceImpl) CloneWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetUserID uuid.UUID, newName string) (*domain.Wishlist, error) {
	if s.log != nil {
		s.log.Logf("INFO cloning wishlist source=%s for user=%s", sourceWishlistID, targetUserID)
	}

	// Fetch source wishlist
	sourceWishlist, err := s.repo.GetWishlistByID(ctx, sourceWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get source wishlist id=%s for clone: %v", sourceWishlistID, err)
		}
		return nil, fmt.Errorf("failed to get source wishlist: %w", err)
	}

	if sourceWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN source wishlist not found for clone id=%s", sourceWishlistID)
		}
		return nil, fmt.Errorf("source wishlist not found")
	}

	// Convert to domain entity for permission check
	sourceDomain := domain.MapWishlistFromSchemaToEntity(sourceWishlist)
	if !sourceDomain.IsAccessibleBy(targetUserID) {
		if s.log != nil {
			s.log.Logf("WARN access denied to clone wishlist=%s by user=%s (private)", sourceWishlistID, targetUserID)
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
			s.log.Logf("ERROR failed to create cloned wishlist for user=%s: %v", targetUserID, err)
		}
		return nil, fmt.Errorf("failed to create cloned wishlist: %w", err)
	}

	// Get all items from source wishlist
	sourceItems, err := s.repo.GetWishlistItems(ctx, sourceWishlistID, 0, 0)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get items from source wishlist=%s for clone: %v", sourceWishlistID, err)
		}
		return nil, fmt.Errorf("failed to get source items: %w", err)
	}

	// Copy items if source has any
	if len(sourceItems) > 0 {
		bulkItems := domain.MapItemsForImport(sourceItems, newWishlist.ID)
		if err := s.repo.CopyItemsBulk(ctx, bulkItems); err != nil {
			if s.log != nil {
				s.log.Logf("ERROR failed to copy items to cloned wishlist=%s: %v", newWishlist.ID, err)
			}
			return nil, fmt.Errorf("failed to copy items: %w", err)
		}
		if s.log != nil {
			s.log.Logf("INFO copied %d items to cloned wishlist=%s", len(sourceItems), newWishlist.ID)
		}
	}

	// Fetch the complete new wishlist with items
	clonedWishlist, err := s.repo.GetWishlistByID(ctx, newWishlist.ID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to fetch cloned wishlist id=%s: %v", newWishlist.ID, err)
		}
		return nil, fmt.Errorf("failed to fetch cloned wishlist: %w", err)
	}

	wishlist := domain.MapWishlistFromSchemaToEntity(clonedWishlist)

	// Cache the new wishlist and invalidate user's list cache
	s.cacheWishlist(ctx, wishlist)
	s.invalidateUserWishlistsCache(ctx, targetUserID)

	if s.log != nil {
		s.log.Logf("INFO cloned wishlist source=%s to new=%s for user=%s", sourceWishlistID, newWishlist.ID, targetUserID)
	}

	return wishlist, nil
}

// MergeWishlist copies items from sourceWishlistID into an existing targetWishlistID.
func (s *WishlistServiceImpl) MergeWishlist(ctx context.Context, sourceWishlistID uuid.UUID, targetWishlistID uuid.UUID, userID uuid.UUID) (int, error) {
	if s.log != nil {
		s.log.Logf("INFO merging wishlist source=%s into target=%s by user=%s", sourceWishlistID, targetWishlistID, userID)
	}

	// Verify target wishlist exists and user owns it
	targetWishlist, err := s.repo.GetWishlistByID(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get target wishlist id=%s for merge: %v", targetWishlistID, err)
		}
		return 0, fmt.Errorf("failed to get target wishlist: %w", err)
	}

	if targetWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN target wishlist not found for merge id=%s", targetWishlistID)
		}
		return 0, fmt.Errorf("target wishlist not found")
	}

	if targetWishlist.UserID != userID {
		if s.log != nil {
			s.log.Logf("WARN unauthorized merge attempt on wishlist=%s by user=%s (owner=%s)", targetWishlistID, userID, targetWishlist.UserID)
		}
		return 0, fmt.Errorf("access denied: only owner can merge into wishlist")
	}

	// Verify source wishlist exists and is accessible
	sourceWishlist, err := s.repo.GetWishlistByID(ctx, sourceWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get source wishlist id=%s for merge: %v", sourceWishlistID, err)
		}
		return 0, fmt.Errorf("failed to get source wishlist: %w", err)
	}

	if sourceWishlist == nil {
		if s.log != nil {
			s.log.Logf("WARN source wishlist not found for merge id=%s", sourceWishlistID)
		}
		return 0, fmt.Errorf("source wishlist not found")
	}

	// Convert to domain entity for permission check
	sourceDomain := domain.MapWishlistFromSchemaToEntity(sourceWishlist)
	if !sourceDomain.IsAccessibleBy(userID) {
		if s.log != nil {
			s.log.Logf("WARN access denied to merge from wishlist=%s by user=%s (private)", sourceWishlistID, userID)
		}
		return 0, fmt.Errorf("access denied: source wishlist is private")
	}

	// Prevent merging a wishlist into itself
	if sourceWishlistID == targetWishlistID {
		if s.log != nil {
			s.log.Logf("WARN attempt to merge wishlist into itself id=%s", sourceWishlistID)
		}
		return 0, fmt.Errorf("cannot merge a wishlist into itself")
	}

	// Count items before merge
	beforeCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to count items before merge: %v", err)
		}
		return 0, fmt.Errorf("failed to count items before merge: %w", err)
	}

	// Get all items from source wishlist
	sourceItems, err := s.repo.GetWishlistItems(ctx, sourceWishlistID, 0, 0)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to get items from source wishlist=%s for merge: %v", sourceWishlistID, err)
		}
		return 0, fmt.Errorf("failed to get source items: %w", err)
	}

	if len(sourceItems) == 0 {
		if s.log != nil {
			s.log.Logf("INFO no items to merge from source=%s", sourceWishlistID)
		}
		return 0, nil // Nothing to merge
	}

	// Prepare items for bulk insert
	bulkItems := domain.MapItemsForImport(sourceItems, targetWishlistID)

	// Perform bulk insert (duplicates are skipped)
	if err := s.repo.CopyItemsBulk(ctx, bulkItems); err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to bulk merge items to wishlist=%s: %v", targetWishlistID, err)
		}
		return 0, fmt.Errorf("failed to merge items: %w", err)
	}

	// Count items after merge to calculate merged count
	afterCount, err := s.repo.CountItems(ctx, targetWishlistID)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to count items after merge: %v", err)
		}
		return 0, fmt.Errorf("failed to count items after merge: %w", err)
	}

	mergedCount := int(afterCount - beforeCount)

	// Invalidate cache for target wishlist
	s.invalidateWishlistItemsCache(ctx, targetWishlistID)

	if s.log != nil {
		s.log.Logf("INFO merged %d items from source=%s to target=%s (skipped %d duplicates)", mergedCount, sourceWishlistID, targetWishlistID, len(sourceItems)-mergedCount)
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
