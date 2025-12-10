package service

import (
	"context"
	"time"

	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
)

// Service aggregates property and listing operations.
// This intentionally does not split property vs listing at the service layer.
type Service interface {
	// Property lifecycle
	CreateProperty(ctx context.Context, p domain.Property) (*domain.Property, error)
	UpdateProperty(ctx context.Context, p domain.Property) (*domain.Property, error)
	PatchProperty(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Property, error)
	GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error)
	GetPropertyByPublicID(ctx context.Context, publicID string) (*domain.Property, error)
	GetPropertiesByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Property, error)
	ListProperties(ctx context.Context, filter PropertyFilter, page Pagination) ([]domain.Property, int64, error)
	DeleteProperty(ctx context.Context, id uuid.UUID, hard bool) error

	// Listing lifecycle
	CreateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error)
	UpdateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error)
	PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Listing, error)
	GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error)
	GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*domain.Listing, error)
	GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]domain.Listing, error)
	GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]domain.Listing, error)
	ListListings(ctx context.Context, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error)
	ListListingsByProperty(ctx context.Context, propertyID uuid.UUID, page Pagination) ([]domain.Listing, int64, error)
	DeleteListing(ctx context.Context, id uuid.UUID, hard bool) error
	UpdateListingStatus(ctx context.Context, id uuid.UUID, status domain.ListingStatus, reason string, changedBy *uuid.UUID) error
	UpdatePublishState(ctx context.Context, id uuid.UUID, published bool, publishedAt *time.Time) error
	PublishListing(ctx context.Context, id uuid.UUID, publishedAt *time.Time, changedBy *uuid.UUID) (*domain.Listing, error)
	UnpublishListing(ctx context.Context, id uuid.UUID, changedBy *uuid.UUID) (*domain.Listing, error)
	IncrementListingView(ctx context.Context, id uuid.UUID, viewedAt time.Time) error
	BulkUpdateListingStatus(ctx context.Context, ids []uuid.UUID, status domain.ListingStatus, reason string, changedBy *uuid.UUID) error
	BulkArchiveListings(ctx context.Context, ids []uuid.UUID, changedBy *uuid.UUID) error
	OnModerationComplete(ctx context.Context, listingID uuid.UUID, approved bool, reason string) error
	UpdateMediaReviewStatus(ctx context.Context, mediaID uuid.UUID, approved bool, reason string) error
	GetListingCompleteness(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID) (*domain.ListingCompleteness, error)
	GetUserListings(ctx context.Context, ownerID uuid.UUID, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error)
	SearchListings(ctx context.Context, sim SimilarityQuery, filter ListingFilter) ([]ScoredResult[domain.Listing], error)

	// Listing media
	UploadListingMedia(ctx context.Context, listingID uuid.UUID, media []domain.ListingMediaInput) ([]domain.ListingMediaResult, error)
	UpdateListingMedia(ctx context.Context, listingID uuid.UUID, mediaID uuid.UUID, updates domain.ListingMediaUpdateInput) error
	DeleteListingMedia(ctx context.Context, listingID uuid.UUID, media []domain.ListingMediaDeleteInput) error
	ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error)
	FinalizeListingMedia(ctx context.Context, data domain.FinalizedListingMedia) error

	// Composite operations
	CreatePropertyWithListing(ctx context.Context, p domain.Property, l domain.Listing) (*domain.Property, *domain.Listing, error)

	// Geospatial searches (PostGIS-backed)
	FindPropertiesNearPoint(ctx context.Context, query NearPointQuery, filter PropertyFilter) ([]ScoredResult[domain.Property], int64, error)
	FindPropertiesInBoundingBox(ctx context.Context, bbox BoundingBox, filter PropertyFilter, page Pagination) ([]domain.Property, int64, error)
	FindPropertiesInPolygon(ctx context.Context, poly PolygonQuery, filter PropertyFilter) ([]domain.Property, int64, error)
	CalculatePropertyDistance(ctx context.Context, propertyID1, propertyID2 uuid.UUID) (float64, error)
	FindNearbyProperties(ctx context.Context, propertyID uuid.UUID, radiusMeters float64, limit int) ([]ScoredResult[domain.Property], error)
	FindListingsNearPoint(ctx context.Context, query NearPointQuery, filter ListingFilter) ([]ScoredResult[domain.Listing], int64, error)
	FindListingsInBoundingBox(ctx context.Context, bbox BoundingBox, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error)

	// Text vector similarity (pgvector-backed)
	SearchListingsByText(ctx context.Context, sim SimilarityQuery, filter ListingFilter) ([]ScoredResult[domain.Listing], error)
	FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int, minSimilarity float64) ([]ScoredResult[domain.Listing], error)
	UpdateListingEmbedding(ctx context.Context, listingID uuid.UUID, embedding []float32, model, version string) error
}
