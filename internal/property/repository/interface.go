package repository

import (
	"context"
	"time"

	"hauslet/internal/property/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PropertyRepository defines storage operations for physical properties.
type PropertyRepository interface {
	// Standard CRUD operations
	CreateProperty(ctx context.Context, property *schema.Property) error
	CreatePropertyTx(ctx context.Context, tx *gorm.DB, property *schema.Property) error
	GetPropertyByPublicID(ctx context.Context, publicID string) (*schema.Property, error)
	UpdateProperty(ctx context.Context, property *schema.Property) error
	UpdatePropertyTx(ctx context.Context, tx *gorm.DB, property *schema.Property) error
	PatchProperty(ctx context.Context, id uuid.UUID, updates map[string]any) error
	PatchPropertyTx(ctx context.Context, tx *gorm.DB, id uuid.UUID, updates map[string]any) error
	GetPropertyByID(ctx context.Context, id uuid.UUID) (*schema.Property, error)
	ListProperties(ctx context.Context, filter PropertyFilter, page Pagination) (*PaginatedResult[schema.Property], error)
	CountProperties(ctx context.Context, filter PropertyFilter) (int64, error)
	PropertyExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetPropertiesByIDs(ctx context.Context, ids []uuid.UUID) ([]schema.Property, error)
	BulkCreateProperties(ctx context.Context, properties []schema.Property) error
	BulkDeleteProperties(ctx context.Context, ids []uuid.UUID, hard bool) error
	SoftDeleteProperty(ctx context.Context, id uuid.UUID) error
	HardDeleteProperty(ctx context.Context, id uuid.UUID) error

	// Geospatial operations
	FindPropertiesNearPoint(ctx context.Context, lat, lng, radiusMeters float64, filter PropertyFilter, page Pagination) (*PaginatedResult[ScoredResult[schema.Property]], error)
	FindPropertiesInBoundingBox(ctx context.Context, bbox BoundingBox, filter PropertyFilter, page Pagination) (*PaginatedResult[schema.Property], error)
	FindPropertiesInPolygon(ctx context.Context, polygonWKT string, filter PropertyFilter, page Pagination) (*PaginatedResult[schema.Property], error)
	CalculatePropertyDistance(ctx context.Context, propertyID1, propertyID2 uuid.UUID) (float64, error)
	FindNearbyProperties(ctx context.Context, propertyID uuid.UUID, radiusMeters float64, limit int) ([]ScoredResult[schema.Property], error)
}

// ListingRepository defines storage operations for commercial listings.
type ListingRepository interface {
	// Standard CRUD operations
	CreateListing(ctx context.Context, listing *schema.Listing) error
	CreateListingTx(ctx context.Context, tx *gorm.DB, listing *schema.Listing) error
	UpdateListing(ctx context.Context, listing *schema.Listing) error
	UpdateListingTx(ctx context.Context, tx *gorm.DB, listing *schema.Listing) error
	PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) error
	PatchListingTx(ctx context.Context, tx *gorm.DB, id uuid.UUID, updates map[string]any) error
	GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*schema.Listing, error)
	GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*schema.Listing, error)
	ListListings(ctx context.Context, filter ListingFilter, page Pagination) (*PaginatedResult[schema.Listing], error)
	ListListingsByPropertyID(ctx context.Context, propertyID uuid.UUID, page Pagination) (*PaginatedResult[schema.Listing], error)
	CountListings(ctx context.Context, filter ListingFilter) (int64, error)
	ListingExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]schema.Listing, error)
	GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]schema.Listing, error)
	BulkCreateListings(ctx context.Context, listings []schema.Listing) error
	BulkDeleteListings(ctx context.Context, ids []uuid.UUID, hard bool) error
	UpdateListingStatus(ctx context.Context, id uuid.UUID, status schema.ListingStatus, reason string, changedBy *uuid.UUID) error
	UpdatePublishState(ctx context.Context, id uuid.UUID, published bool, publishedAt *time.Time) error
	AddListingMedia(ctx context.Context, listingID uuid.UUID, media []schema.ListingMedia) error
	DeleteListingMedia(ctx context.Context, listingID uuid.UUID, mediaIDs []uuid.UUID) error
	UpdateListingMedia(ctx context.Context, listingID uuid.UUID, mediaID uuid.UUID, updates map[string]any) error
	ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]schema.ListingMedia, error)
	FindStaleListingMedia(ctx context.Context, olderThan time.Time, limit int) ([]schema.ListingMedia, error)
	SoftDeleteListing(ctx context.Context, id uuid.UUID) error
	HardDeleteListing(ctx context.Context, id uuid.UUID) error
	IncrementView(ctx context.Context, id uuid.UUID, viewedAt time.Time) error

	// Geospatial operations (via property location)
	FindListingsNearPoint(ctx context.Context, lat, lng, radiusMeters float64, filter ListingFilter, page Pagination) (*PaginatedResult[ScoredResult[schema.Listing]], error)
	FindListingsInBoundingBox(ctx context.Context, bbox BoundingBox, filter ListingFilter, page Pagination) (*PaginatedResult[schema.Listing], error)

	// Text similarity search operations
	SearchListingsByText(ctx context.Context, queryVector []float32, filter ListingFilter, similarity SimilarityFilter) ([]ScoredResult[schema.Listing], error)
	FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int, minSimilarity float64) ([]ScoredResult[schema.Listing], error)
	UpdateListingEmbedding(ctx context.Context, listingID uuid.UUID, embedding []float32, model, version string) error

	// Image similarity search operations
	SearchImagesByVector(ctx context.Context, queryVector []float32, imagFilter ImageSimilarityFilter) ([]ScoredResult[schema.ListingMedia], error)
	FindSimilarImages(ctx context.Context, mediaID uuid.UUID, limit int, minSimilarity float64) ([]ScoredResult[schema.ListingMedia], error)
	UpdateMediaEmbedding(ctx context.Context, mediaID uuid.UUID, embedding []float32, model, version string) error
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
