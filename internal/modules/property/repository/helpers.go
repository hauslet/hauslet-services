package repository

import (
	"fmt"
	"hauslet/internal/modules/property/repository/schema"
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
)

// PaginatedResult wraps paginated data with total count.
type PaginatedResult[T any] struct {
	Items      []T
	TotalCount int64
	Limit      int
	Offset     int
}

// ScoredListing carries a listing with a vector similarity score.
type ScoredListing struct {
	Listing  schema.Listing         `gorm:"embedded"`
	Score    float64                `gorm:"column:score"`
	Location *schema.GeographyPoint `gorm:"-"` // Manual hydration
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
	Query           *string
	OwnerID         *uuid.UUID
	PropertyID      *uuid.UUID
	OwnerTypes      []schema.OwnerType
	ListingTypes    []schema.ListingType
	Statuses        []schema.ListingStatus
	ReviewStatuses  []schema.ReviewStatus
	Published       *bool
	HasCalendar     *bool
	MinPrice        *float64
	MaxPrice        *float64
	Currency        *schema.CurrencyCode
	City            *string
	State           *string
	Country         *schema.CountryCode
	PropertyTypes   []schema.PropertyType
	Furnishings     []schema.FurnishingType
	MinBedrooms     *int
	MaxBedrooms     *int
	MinBathrooms    *int
	MaxBathrooms    *int
	Latitude        *float64
	Longitude       *float64
	RadiusMeters    *float64
	PublishedAfter  *time.Time
	PublishedBefore *time.Time
	CreatedAfter    *time.Time
	CreatedBefore   *time.Time
	IncludeDeleted  bool
	SortBy          ListingSortBy
	SortOrder       SortOrder

	// Type-specific filters
	ShortletFilter    *ShortletFilter
	RentalFilter      *RentalFilter
	SaleFilter        *SaleFilter
	PropertyExtension *PropertyFilterExtension
}

// ShortletFilter defines filters specific to shortlet listings
type ShortletFilter struct {
	// Pricing
	MinExtraGuestFee *float64
	MaxExtraGuestFee *float64

	// Stay Duration - Range filtering for min_nights
	MinNightsMin *int // Listings where min_nights >= this
	MinNightsMax *int // Listings where min_nights <= this

	// Stay Duration - Range filtering for max_nights
	MaxNightsMin *int // Listings where max_nights >= this
	MaxNightsMax *int // Listings where max_nights <= this

	// Capacity
	MinMaxGuests   *int // Listings where max_guests >= this
	BaseGuestCount *int // Exact match for base_guest_count

	// Timing - Range matching
	CheckInTimeAfter   *string // HH:MM format
	CheckInTimeBefore  *string // HH:MM format
	CheckOutTimeAfter  *string // HH:MM format
	CheckOutTimeBefore *string // HH:MM format

	// Type
	AccommodationTypes []schema.AccommodationType
}

// RentalFilter defines filters specific to rental listings
type RentalFilter struct {
	// Rental Terms
	RentalPricePeriods []schema.PaymentPeriod

	// Rental Period - Range filtering for min_rental_period
	MinRentalPeriodMin *int // Listings where min_rental_period >= this
	MinRentalPeriodMax *int // Listings where min_rental_period <= this

	// Rental Period - Range filtering for max_rental_period
	MaxRentalPeriodMin *int // Listings where max_rental_period >= this
	MaxRentalPeriodMax *int // Listings where max_rental_period <= this

	// Availability Date Range
	AvailableFrom *time.Time
	AvailableTo   *time.Time
}

// SaleFilter defines filters specific to sale listings
type SaleFilter struct {
	OwnershipTitles []string // Must match one of these
	PaymentPlan     *bool    // true = must have, false = must not have, nil = either
}

// PropertyFilterExtension defines additional property filters
type PropertyFilterExtension struct {
	PropertyClasses    []schema.PropertyClass
	PropertyConditions []schema.PropertyCondition
	Amenities          []string // Must have ALL (AND logic)
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
