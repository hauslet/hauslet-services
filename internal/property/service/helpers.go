package service

import (
	"hauslet/internal/property/domain"
	"time"

	"github.com/google/uuid"
)

// Pagination captures standard limit/offset pagination.
type Pagination struct {
	Limit  int
	Offset int
}

// SortOrder defines sort direction.
type SortOrder string

const (
	SortAsc  SortOrder = "ASC"
	SortDesc SortOrder = "DESC"
)

// PropertySortBy defines sortable fields for properties.
type PropertySortBy string

const (
	PropertySortCreatedAt PropertySortBy = "created_at"
	PropertySortUpdatedAt PropertySortBy = "updated_at"
	PropertySortBedrooms  PropertySortBy = "bedrooms"
	PropertySortBathrooms PropertySortBy = "bathrooms"
)

// ListingSortBy defines sortable fields for listings.
type ListingSortBy string

const (
	ListingSortCreatedAt   ListingSortBy = "created_at"
	ListingSortUpdatedAt   ListingSortBy = "updated_at"
	ListingSortPublishedAt ListingSortBy = "published_at"
	ListingSortViewCount   ListingSortBy = "view_count"
)

// PropertyFilter defines optional criteria for querying properties.
type PropertyFilter struct {
	OwnerID        *uuid.UUID
	City           *string
	State          *string
	Country        *domain.CountryCode
	Classes        []domain.PropertyClass
	Types          []domain.PropertyType
	Furnishings    []domain.FurnishingType
	Conditions     []domain.PropertyCondition
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
	OwnerTypes      []domain.OwnerType
	ListingTypes    []domain.ListingType
	Statuses        []domain.ListingStatus
	ReviewStatuses  []domain.ReviewStatus
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

// BoundingBox represents a rectangular geographic area.
type BoundingBox struct {
	NorthEastLat float64
	NorthEastLng float64
	SouthWestLat float64
	SouthWestLng float64
}

// NearPointQuery specifies a point-radius search.
type NearPointQuery struct {
	Lat          float64
	Lng          float64
	RadiusMeters float64
	Limit        int
	Offset       int
}

// PolygonQuery specifies a polygon search via WKT.
type PolygonQuery struct {
	PolygonWKT string
	Limit      int
	Offset     int
}

// SimilarityQuery defines criteria for vector similarity search.
type SimilarityQuery struct {
	QueryVector   []float32 // required
	TopK          int       // defaults applied if <=0
	MinSimilarity float64   // 0..1; optional threshold
}

// ScoredResult wraps an item with optional scores/distances.
type ScoredResult[T any] struct {
	Item            T
	SimilarityScore float64 // 0..1 if applicable
	DistanceMeters  float64 // if geospatial
	Ranking         int
}
