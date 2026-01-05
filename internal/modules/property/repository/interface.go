package repository

import (
	"context"
	"time"

	"hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PropertyRepository defines storage operations for physical properties.
type PropertyRepository interface {
	// Standard CRUD operations
	CreateProperty(ctx context.Context, property *schema.Property) error
	CreatePropertyTx(ctx context.Context, tx *gorm.DB, property *schema.Property) error
	UpdateProperty(ctx context.Context, property *schema.Property) error
	PatchProperty(ctx context.Context, id uuid.UUID, updates map[string]any) error
	GetPropertyByID(ctx context.Context, id uuid.UUID) (*schema.Property, error)
	ListProperties(ctx context.Context, filter PropertyFilter, page Pagination) (*PaginatedResult[schema.Property], error)
	PropertyExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetPropertiesByIDs(ctx context.Context, ids []uuid.UUID) ([]schema.Property, error)
	SoftDeleteProperty(ctx context.Context, id uuid.UUID) error
	HardDeleteProperty(ctx context.Context, id uuid.UUID) error
}

// ListingRepository defines storage operations for commercial listings.
type ListingRepository interface {
	// Standard CRUD operations
	CreateListing(ctx context.Context, listing *schema.Listing) error
	CreateListingTx(ctx context.Context, tx *gorm.DB, listing *schema.Listing) error
	UpdateListing(ctx context.Context, listing *schema.Listing) error
	PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) error
	GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*schema.Listing, error)
	GetListingWithPropertyByID(ctx context.Context, id uuid.UUID) (*schema.Listing, *schema.Property, error)
	GetListingByPublicID(ctx context.Context, publicID string, preloadMedia bool) (*schema.Listing, error)
	GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*schema.Listing, error)
	ListListings(ctx context.Context, filter ListingFilter, page Pagination) (*PaginatedResult[schema.Listing], error)
	ListListingsByPropertyID(ctx context.Context, propertyID uuid.UUID, page Pagination) (*PaginatedResult[schema.Listing], error)
	ListingExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]schema.Listing, error)
	GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]schema.Listing, error)
	AddListingMedia(ctx context.Context, listingID uuid.UUID, media []schema.ListingMedia) error
	DeleteListingMedia(ctx context.Context, listingID uuid.UUID, mediaIDs []uuid.UUID) error
	UpdateListingMedia(ctx context.Context, listingID uuid.UUID, mediaID uuid.UUID, updates map[string]any) error
	ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]schema.ListingMedia, error)
	FindStaleListingMedia(ctx context.Context, olderThan time.Time, limit int) ([]schema.ListingMedia, error)
	SearchListings(ctx context.Context, embedding *schema.VectorEmbedding, filter ListingFilter, limit int) ([]ScoredListing, error)
	SoftDeleteListing(ctx context.Context, id uuid.UUID) error
	HardDeleteListing(ctx context.Context, id uuid.UUID) error
}

// Repository aggregates property and listing persistence.
type Repository interface {
	PropertyRepository
	ListingRepository

	// Transaction executes a function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, the transaction is committed.
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}
