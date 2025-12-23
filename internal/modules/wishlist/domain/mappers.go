package domain

import (
	"hauslet/internal/modules/wishlist/repository/schema"

	"github.com/google/uuid"
)

// --- Schema to Domain Entity Mappers ---

// MapWishlistFromSchemaToEntity converts repository schema.Wishlist to domain Wishlist entity
func MapWishlistFromSchemaToEntity(schemaWishlist *schema.Wishlist) *Wishlist {
	if schemaWishlist == nil {
		return nil
	}

	wishlist := &Wishlist{
		ID:          schemaWishlist.ID,
		UserID:      schemaWishlist.UserID,
		Name:        schemaWishlist.Name,
		Description: schemaWishlist.Description,
		IsPrivate:   schemaWishlist.IsPrivate,
		CreatedAt:   schemaWishlist.CreatedAt,
		UpdatedAt:   schemaWishlist.UpdatedAt,
	}

	// Map items if present
	if len(schemaWishlist.Items) > 0 {
		wishlist.Items = make([]WishlistItem, 0, len(schemaWishlist.Items))
		for _, item := range schemaWishlist.Items {
			if mapped := MapWishlistItemFromSchemaToEntity(&item); mapped != nil {
				wishlist.Items = append(wishlist.Items, *mapped)
			}
		}
	}

	return wishlist
}

// MapWishlistToSchemaEntity converts domain Wishlist entity to repository schema.Wishlist
func MapWishlistToSchemaEntity(wishlist *Wishlist) *schema.Wishlist {
	if wishlist == nil {
		return nil
	}

	schemaWishlist := &schema.Wishlist{
		ID:          wishlist.ID,
		UserID:      wishlist.UserID,
		Name:        wishlist.Name,
		Description: wishlist.Description,
		IsPrivate:   wishlist.IsPrivate,
		CreatedAt:   wishlist.CreatedAt,
		UpdatedAt:   wishlist.UpdatedAt,
	}

	// Map items if present
	if len(wishlist.Items) > 0 {
		schemaWishlist.Items = make([]schema.WishlistItem, 0, len(wishlist.Items))
		for _, item := range wishlist.Items {
			mapped := MapWishlistItemToSchemaEntity(&item)
			if mapped != nil {
				schemaWishlist.Items = append(schemaWishlist.Items, *mapped)
			}
		}
	}

	return schemaWishlist
}

// MapWishlistItemFromSchemaToEntity converts schema.WishlistItem to domain WishlistItem entity
func MapWishlistItemFromSchemaToEntity(schemaItem *schema.WishlistItem) *WishlistItem {
	if schemaItem == nil {
		return nil
	}

	return &WishlistItem{
		ID:         schemaItem.ID,
		WishlistID: schemaItem.WishlistID,
		ListingID:  schemaItem.ListingID,
		AddedAt:    schemaItem.AddedAt,
		Source:     WishlistItemSource(schemaItem.Source),
	}
}

// MapWishlistItemToSchemaEntity converts domain WishlistItem entity to schema.WishlistItem
func MapWishlistItemToSchemaEntity(item *WishlistItem) *schema.WishlistItem {
	if item == nil {
		return nil
	}

	return &schema.WishlistItem{
		ID:         item.ID,
		WishlistID: item.WishlistID,
		ListingID:  item.ListingID,
		AddedAt:    item.AddedAt,
		Source:     schema.WishlistItemSource(item.Source),
	}
}

// --- Bulk Conversion Helpers ---

// MapWishlistsFromSchemaToEntities converts a slice of schema wishlists to domain entities
func MapWishlistsFromSchemaToEntities(schemaWishlists []*schema.Wishlist) []*Wishlist {
	if schemaWishlists == nil {
		return nil
	}

	entities := make([]*Wishlist, 0, len(schemaWishlists))
	for _, schemaWishlist := range schemaWishlists {
		if mapped := MapWishlistFromSchemaToEntity(schemaWishlist); mapped != nil {
			entities = append(entities, mapped)
		}
	}

	return entities
}

// MapWishlistItemsFromSchemaToEntities converts a slice of schema items to domain entities
func MapWishlistItemsFromSchemaToEntities(schemaItems []*schema.WishlistItem) []*WishlistItem {
	if schemaItems == nil {
		return nil
	}

	entities := make([]*WishlistItem, 0, len(schemaItems))
	for _, schemaItem := range schemaItems {
		if mapped := MapWishlistItemFromSchemaToEntity(schemaItem); mapped != nil {
			entities = append(entities, mapped)
		}
	}

	return entities
}

// --- Helper Functions for Repository Operations ---

// CreateSchemaWishlist creates a schema.Wishlist from basic parameters
func CreateSchemaWishlist(userID uuid.UUID, name string, description *string, isPrivate bool) *schema.Wishlist {
	return &schema.Wishlist{
		UserID:      userID,
		Name:        name,
		Description: description,
		IsPrivate:   isPrivate,
	}
}

// CreateSchemaWishlistItem creates a schema.WishlistItem from basic parameters
func CreateSchemaWishlistItem(wishlistID, listingID uuid.UUID, source WishlistItemSource) *schema.WishlistItem {
	return &schema.WishlistItem{
		WishlistID: wishlistID,
		ListingID:  listingID,
		Source:     schema.WishlistItemSource(source),
	}
}

// ApplyUpdateToSchemaWishlist applies partial updates to an existing schema wishlist
func ApplyUpdateToSchemaWishlist(wishlist *schema.Wishlist, name *string, description *string, isPrivate *bool) {
	if wishlist == nil {
		return
	}

	if name != nil {
		wishlist.Name = *name
	}

	if description != nil {
		wishlist.Description = description
	}

	if isPrivate != nil {
		wishlist.IsPrivate = *isPrivate
	}
}

// CreateBulkSchemaWishlistItems creates a slice of schema.WishlistItem for bulk operations
func CreateBulkSchemaWishlistItems(targetWishlistID uuid.UUID, listingIDs []uuid.UUID, source WishlistItemSource) []schema.WishlistItem {
	if len(listingIDs) == 0 {
		return nil
	}

	items := make([]schema.WishlistItem, 0, len(listingIDs))
	for _, listingID := range listingIDs {
		items = append(items, schema.WishlistItem{
			WishlistID: targetWishlistID,
			ListingID:  listingID,
			Source:     schema.WishlistItemSource(source),
		})
	}

	return items
}

// MapItemsForImport creates schema.WishlistItem for importing from another wishlist
func MapItemsForImport(sourceItems []*schema.WishlistItem, targetWishlistID uuid.UUID) []schema.WishlistItem {
	if len(sourceItems) == 0 {
		return nil
	}

	items := make([]schema.WishlistItem, 0, len(sourceItems))
	for _, sourceItem := range sourceItems {
		items = append(items, schema.WishlistItem{
			WishlistID: targetWishlistID,
			ListingID:  sourceItem.ListingID,
			Source:     schema.SourceImported,
		})
	}

	return items
}
