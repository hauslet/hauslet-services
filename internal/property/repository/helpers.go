package repository

import (
	"fmt"
	"hauslet/internal/property/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Pagination captures standard limit/offset pagination.
type Pagination struct {
	Limit  int
	Offset int
}

// SortOrder defines sort direction.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "ASC"
	SortOrderDesc SortOrder = "DESC"
)

// PropertySortBy defines sortable fields for properties.
type PropertySortBy string

const (
	PropertySortByCreatedAt PropertySortBy = "created_at"
	PropertySortByUpdatedAt PropertySortBy = "updated_at"
	PropertySortByBedrooms  PropertySortBy = "bedrooms"
	PropertySortByBathrooms PropertySortBy = "bathrooms"
)

// ListingSortBy defines sortable fields for listings.
type ListingSortBy string

const (
	ListingSortByCreatedAt   ListingSortBy = "created_at"
	ListingSortByUpdatedAt   ListingSortBy = "updated_at"
	ListingSortByPublishedAt ListingSortBy = "published_at"
	ListingSortByViewCount   ListingSortBy = "view_count"
)

// PaginatedResult wraps paginated data with total count.
type PaginatedResult[T any] struct {
	Items      []T
	TotalCount int64
	Limit      int
	Offset     int
}

// SimilarityMetric defines the distance metric for vector search.
type SimilarityMetric string

const (
	SimilarityMetricCosine       SimilarityMetric = "cosine"        // Default: 1 - cosine distance
	SimilarityMetricL2           SimilarityMetric = "l2"            // Euclidean distance
	SimilarityMetricInnerProduct SimilarityMetric = "inner_product" // Dot product (for normalized vectors)
)

// GeospatialFilter defines criteria for location-based queries.
type GeospatialFilter struct {
	// Point-based search
	CenterLat    *float64 // Center point latitude
	CenterLng    *float64 // Center point longitude
	RadiusMeters *float64 // Search radius in meters

	// Bounding box search (alternative to point + radius)
	BoundingBox *BoundingBox

	// Polygon search (for custom shapes)
	PolygonWKT *string // Well-Known Text representation of polygon

	// Result options
	SortByDistance bool     // Sort results by distance from center point
	MaxDistance    *float64 // Maximum distance filter (meters)
}

// BoundingBox represents a rectangular geographic area.
type BoundingBox struct {
	NorthEastLat float64
	NorthEastLng float64
	SouthWestLat float64
	SouthWestLng float64
}

// SimilarityFilter defines criteria for vector similarity search.
type SimilarityFilter struct {
	// Input options (provide one)
	QueryText   *string   // Text to search for (will generate embedding)
	QueryVector []float32 // Pre-computed embedding vector

	// Search parameters
	MinSimilarity float64          // Minimum similarity threshold (0.0-1.0)
	Metric        SimilarityMetric // Distance metric to use
	TopK          int              // Number of results to return (default: 10)
}

// ImageSimilarityFilter defines criteria for image similarity search.
type ImageSimilarityFilter struct {
	// Input options (provide one)
	QueryImageURL *string   // Image URL to search for (will generate embedding)
	QueryVector   []float32 // Pre-computed image embedding vector

	// Search parameters
	MinSimilarity float64          // Minimum similarity threshold (0.0-1.0)
	Metric        SimilarityMetric // Distance metric to use
	TopK          int              // Number of results to return (default: 10)

	// Filter options
	ListingID     *uuid.UUID         // Limit search to specific listing's images
	MediaTypes    []schema.MediaType // Filter by media type
	IsPrimaryOnly bool               // Only search primary images
}

// ScoredResult wraps a result with similarity/distance scores.
type ScoredResult[T any] struct {
	Item            T
	SimilarityScore float64 // 0.0-1.0, higher is more similar
	DistanceMeters  float64 // Distance from query point (if geospatial)
	RelevanceScore  float64 // Combined weighted score
	Ranking         int     // Position in results (1-indexed)
}

// PropertyFilter defines optional criteria for querying properties.
type PropertyFilter struct {
	OwnerID        *uuid.UUID
	City           *string
	State          *string
	Country        *schema.CountryCode
	Classes        []schema.PropertyClass
	Types          []schema.PropertyType
	Furnishings    []schema.FurnishingType
	Conditions     []schema.PropertyCondition
	MinBedrooms    *int
	MinBathrooms   *int
	IncludeDeleted bool
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	UpdatedAfter   *time.Time
	UpdatedBefore  *time.Time
	SortBy         PropertySortBy
	SortOrder      SortOrder
}

// ListingFilter defines optional criteria for querying listings.
type ListingFilter struct {
	OwnerID         *uuid.UUID
	PropertyID      *uuid.UUID
	OwnerTypes      []schema.OwnerType
	ListingTypes    []schema.ListingType
	Statuses        []schema.ListingStatus
	ReviewStatuses  []schema.ReviewStatus
	Published       *bool
	HasCalendar     *bool
	PublishedAfter  *time.Time
	PublishedBefore *time.Time
	CreatedAfter    *time.Time
	CreatedBefore   *time.Time
	MinViewCount    *int
	IncludeDeleted  bool
	SortBy          ListingSortBy
	SortOrder       SortOrder
}

const (
	// MaxPaginationLimit prevents fetching too many records at once.
	MaxPaginationLimit = 1000
	// DefaultPaginationLimit is used when no limit is specified.
	DefaultPaginationLimit = 20
	// BatchInsertSize controls how many records to insert in a single batch.
	BatchInsertSize = 100
)

// applyPagination applies limit/offset with sane defaults and max limits.
func applyPagination(db *gorm.DB, page Pagination) *gorm.DB {
	limit := page.Limit
	offset := page.Offset

	if limit <= 0 {
		limit = DefaultPaginationLimit
	}
	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}
	if offset < 0 {
		offset = 0
	}

	return db.Limit(limit).Offset(offset)
}

// applyPropertyFilter applies property-specific filters.
func applyPropertyFilter(db *gorm.DB, f PropertyFilter) *gorm.DB {
	if f.OwnerID != nil {
		db = db.Where("owner_id = ?", *f.OwnerID)
	}
	if f.City != nil {
		db = db.Where("city = ?", *f.City)
	}
	if f.State != nil {
		db = db.Where("state = ?", *f.State)
	}
	if f.Country != nil {
		db = db.Where("country = ?", *f.Country)
	}
	if len(f.Classes) > 0 {
		db = db.Where("property_class IN ?", f.Classes)
	}
	if len(f.Types) > 0 {
		db = db.Where("property_type IN ?", f.Types)
	}
	if len(f.Furnishings) > 0 {
		db = db.Where("furnishing_type IN ?", f.Furnishings)
	}
	if len(f.Conditions) > 0 {
		db = db.Where("property_condition IN ?", f.Conditions)
	}
	if f.MinBedrooms != nil {
		db = db.Where("bedrooms >= ?", *f.MinBedrooms)
	}
	if f.MinBathrooms != nil {
		db = db.Where("bathrooms >= ?", *f.MinBathrooms)
	}
	if f.CreatedAfter != nil {
		db = db.Where("created_at >= ?", *f.CreatedAfter)
	}
	if f.CreatedBefore != nil {
		db = db.Where("created_at <= ?", *f.CreatedBefore)
	}
	if f.UpdatedAfter != nil {
		db = db.Where("updated_at >= ?", *f.UpdatedAfter)
	}
	if f.UpdatedBefore != nil {
		db = db.Where("updated_at <= ?", *f.UpdatedBefore)
	}
	return db
}

// applyPropertySort applies sorting to property queries.
func applyPropertySort(db *gorm.DB, sortBy PropertySortBy, sortOrder SortOrder) *gorm.DB {
	// Default sorting
	if sortBy == "" {
		sortBy = PropertySortByCreatedAt
	}
	if sortOrder == "" {
		sortOrder = SortOrderDesc
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)
	return db.Order(orderClause)
}

// applyListingFilter applies listing-specific filters.
func applyListingFilter(db *gorm.DB, f ListingFilter) *gorm.DB {
	if f.OwnerID != nil {
		db = db.Where("owner_id = ?", *f.OwnerID)
	}
	if f.PropertyID != nil {
		db = db.Where("property_id = ?", *f.PropertyID)
	}
	if len(f.OwnerTypes) > 0 {
		db = db.Where("owner_type IN ?", f.OwnerTypes)
	}
	if len(f.ListingTypes) > 0 {
		db = db.Where("listing_type IN ?", f.ListingTypes)
	}
	if len(f.Statuses) > 0 {
		db = db.Where("status IN ?", f.Statuses)
	}
	if len(f.ReviewStatuses) > 0 {
		db = db.Where("latest_review_status IN ?", f.ReviewStatuses)
	}
	if f.Published != nil {
		db = db.Where("published = ?", *f.Published)
	}
	if f.HasCalendar != nil {
		db = db.Where("has_calendar = ?", *f.HasCalendar)
	}
	if f.PublishedAfter != nil {
		db = db.Where("published_at >= ?", *f.PublishedAfter)
	}
	if f.PublishedBefore != nil {
		db = db.Where("published_at <= ?", *f.PublishedBefore)
	}
	if f.CreatedAfter != nil {
		db = db.Where("created_at >= ?", *f.CreatedAfter)
	}
	if f.CreatedBefore != nil {
		db = db.Where("created_at <= ?", *f.CreatedBefore)
	}
	if f.MinViewCount != nil {
		db = db.Where("view_count >= ?", *f.MinViewCount)
	}

	return db
}

// applyListingSort applies sorting to listing queries.
func applyListingSort(db *gorm.DB, sortBy ListingSortBy, sortOrder SortOrder) *gorm.DB {
	// Default sorting
	if sortBy == "" {
		sortBy = ListingSortByCreatedAt
	}
	if sortOrder == "" {
		sortOrder = SortOrderDesc
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)
	return db.Order(orderClause)
}
