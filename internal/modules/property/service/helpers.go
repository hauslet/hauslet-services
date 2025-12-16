package service

import (
	"hauslet/internal/modules/property/domain"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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

var (
	nonSlugChars = regexp.MustCompile(`[^a-z0-9\-]+`)
	multiHyphens = regexp.MustCompile(`-+`)
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

// ScoredResult wraps an item with optional scores/distances.
type ScoredResult[T any] struct {
	Item            T
	SimilarityScore float64 // 0..1 if applicable
	DistanceMeters  float64 // if geospatial
	Ranking         int
}

// generateSlug creates a clean URL slug for Hauslet entities.
func generateSlug(input string) string {
	if input == "" {
		return ""
	}

	// Normalize and remove diacritics (é → e, ü → u, etc.)
	t := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)),
		norm.NFC,
	)
	normalized, _, _ := transform.String(t, input)

	// Lowercase
	slug := strings.ToLower(normalized)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-slug characters
	slug = nonSlugChars.ReplaceAllString(slug, "-")

	// Collapse multiple hyphens
	slug = multiHyphens.ReplaceAllString(slug, "-")

	// Trim hyphens from start/end
	slug = strings.Trim(slug, "-")

	return slug
}

func shortid() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
