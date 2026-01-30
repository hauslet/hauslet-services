package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"
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
	Query              *string
	OwnerID            *uuid.UUID
	PropertyID         *uuid.UUID
	OwnerTypes         []domain.OwnerType
	ListingTypes       []domain.ListingType
	Statuses           []domain.ListingStatus
	ReviewStatuses     []domain.ReviewStatus
	Published          *bool
	HasCalendar        *bool
	MinPrice           *float64
	MaxPrice           *float64
	Currency           *domain.CurrencyCode
	PublishedAfter     *time.Time
	PublishedBefore    *time.Time
	CreatedAfter       *time.Time
	CreatedBefore      *time.Time
	IncludeDeleted     bool
	SortBy             ListingSortBy
	SortOrder          SortOrder
	City               *string
	State              *string
	Country            *domain.CountryCode
	PropertyTypes      []domain.PropertyType
	Furnishings        []domain.FurnishingType
	MinBedrooms        *int
	MaxBedrooms        *int
	MinBathrooms       *int
	MaxBathrooms       *int
	Latitude           *float64
	Longitude          *float64
	RadiusMeters       *float64
	ExcludedListingIDs []uuid.UUID

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
	AccommodationTypes []domain.AccommodationType
}

// RentalFilter defines filters specific to rental listings
type RentalFilter struct {
	// Rental Terms
	RentalPricePeriods []domain.PaymentPeriod

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
	PropertyClasses    []domain.PropertyClass
	PropertyConditions []domain.PropertyCondition
	Amenities          []string // Must have ALL (AND logic)
}

// mapListingFilterToRepo converts service filters to repository filters.
func mapListingFilterToRepo(filter ListingFilter) repository.ListingFilter {
	repoFilter := repository.ListingFilter{
		Query:              filter.Query,
		OwnerID:            filter.OwnerID,
		PropertyID:         filter.PropertyID,
		Published:          filter.Published,
		HasCalendar:        filter.HasCalendar,
		MinPrice:           filter.MinPrice,
		MaxPrice:           filter.MaxPrice,
		IncludeDeleted:     filter.IncludeDeleted,
		CreatedAfter:       filter.CreatedAfter,
		CreatedBefore:      filter.CreatedBefore,
		PublishedAfter:     filter.PublishedAfter,
		PublishedBefore:    filter.PublishedBefore,
		City:               filter.City,
		State:              filter.State,
		Latitude:           filter.Latitude,
		Longitude:          filter.Longitude,
		RadiusMeters:       filter.RadiusMeters,
		ExcludedListingIDs: filter.ExcludedListingIDs,
	}

	if len(filter.OwnerTypes) > 0 {
		repoFilter.OwnerTypes = make([]schema.OwnerType, len(filter.OwnerTypes))
		for i, v := range filter.OwnerTypes {
			repoFilter.OwnerTypes[i] = schema.OwnerType(v)
		}
	}

	if len(filter.ListingTypes) > 0 {
		repoFilter.ListingTypes = make([]schema.ListingType, len(filter.ListingTypes))
		for i, v := range filter.ListingTypes {
			repoFilter.ListingTypes[i] = schema.ListingType(v)
		}
	}

	if len(filter.Statuses) > 0 {
		repoFilter.Statuses = make([]schema.ListingStatus, len(filter.Statuses))
		for i, v := range filter.Statuses {
			repoFilter.Statuses[i] = schema.ListingStatus(v)
		}
	}

	if len(filter.ReviewStatuses) > 0 {
		repoFilter.ReviewStatuses = make([]schema.ReviewStatus, len(filter.ReviewStatuses))
		for i, v := range filter.ReviewStatuses {
			repoFilter.ReviewStatuses[i] = schema.ReviewStatus(v)
		}
	}

	if filter.Country != nil {
		c := schema.CountryCode(*filter.Country)
		repoFilter.Country = &c
	}
	if filter.Currency != nil {
		c := schema.CurrencyCode(*filter.Currency)
		repoFilter.Currency = &c
	}
	if filter.Currency != nil {
		c := schema.CurrencyCode(*filter.Currency)
		repoFilter.Currency = &c
	}
	if len(filter.PropertyTypes) > 0 {
		repoFilter.PropertyTypes = make([]schema.PropertyType, len(filter.PropertyTypes))
		for i, v := range filter.PropertyTypes {
			repoFilter.PropertyTypes[i] = schema.PropertyType(v)
		}
	}
	if len(filter.Furnishings) > 0 {
		repoFilter.Furnishings = make([]schema.FurnishingType, len(filter.Furnishings))
		for i, v := range filter.Furnishings {
			repoFilter.Furnishings[i] = schema.FurnishingType(v)
		}
	}
	repoFilter.MinBedrooms = filter.MinBedrooms
	repoFilter.MaxBedrooms = filter.MaxBedrooms
	repoFilter.MinBathrooms = filter.MinBathrooms
	repoFilter.MaxBathrooms = filter.MaxBathrooms

	// Map type-specific filters
	if filter.ShortletFilter != nil {
		repoFilter.ShortletFilter = mapShortletFilter(filter.ShortletFilter)
	}
	if filter.RentalFilter != nil {
		repoFilter.RentalFilter = mapRentalFilter(filter.RentalFilter)
	}
	if filter.SaleFilter != nil {
		repoFilter.SaleFilter = &repository.SaleFilter{
			OwnershipTitles: filter.SaleFilter.OwnershipTitles,
			PaymentPlan:     filter.SaleFilter.PaymentPlan,
		}
	}
	if filter.PropertyExtension != nil {
		repoFilter.PropertyExtension = mapPropertyExtension(filter.PropertyExtension)
	}

	repoFilter.SortBy = mapListingSortBy(filter.SortBy)
	repoFilter.SortOrder = mapSortOrder(filter.SortOrder)

	return repoFilter
}

// mapShortletFilter converts service shortlet filter to repository shortlet filter
func mapShortletFilter(filter *ShortletFilter) *repository.ShortletFilter {
	if filter == nil {
		return nil
	}

	repoFilter := &repository.ShortletFilter{
		MinExtraGuestFee:   filter.MinExtraGuestFee,
		MaxExtraGuestFee:   filter.MaxExtraGuestFee,
		MinNightsMin:       filter.MinNightsMin,
		MinNightsMax:       filter.MinNightsMax,
		MaxNightsMin:       filter.MaxNightsMin,
		MaxNightsMax:       filter.MaxNightsMax,
		MinMaxGuests:       filter.MinMaxGuests,
		BaseGuestCount:     filter.BaseGuestCount,
		CheckInTimeAfter:   filter.CheckInTimeAfter,
		CheckInTimeBefore:  filter.CheckInTimeBefore,
		CheckOutTimeAfter:  filter.CheckOutTimeAfter,
		CheckOutTimeBefore: filter.CheckOutTimeBefore,
	}

	// Map accommodation types
	if len(filter.AccommodationTypes) > 0 {
		repoFilter.AccommodationTypes = make([]schema.AccommodationType, len(filter.AccommodationTypes))
		for i, v := range filter.AccommodationTypes {
			repoFilter.AccommodationTypes[i] = schema.AccommodationType(v)
		}
	}

	return repoFilter
}

// mapRentalFilter converts service rental filter to repository rental filter
func mapRentalFilter(filter *RentalFilter) *repository.RentalFilter {
	if filter == nil {
		return nil
	}

	repoFilter := &repository.RentalFilter{
		MinRentalPeriodMin: filter.MinRentalPeriodMin,
		MinRentalPeriodMax: filter.MinRentalPeriodMax,
		MaxRentalPeriodMin: filter.MaxRentalPeriodMin,
		MaxRentalPeriodMax: filter.MaxRentalPeriodMax,
		AvailableFrom:      filter.AvailableFrom,
		AvailableTo:        filter.AvailableTo,
	}

	// Map rental price periods
	if len(filter.RentalPricePeriods) > 0 {
		repoFilter.RentalPricePeriods = make([]schema.PaymentPeriod, len(filter.RentalPricePeriods))
		for i, v := range filter.RentalPricePeriods {
			repoFilter.RentalPricePeriods[i] = schema.PaymentPeriod(v)
		}
	}

	return repoFilter
}

// mapPropertyExtension converts service property extension to repository property extension
func mapPropertyExtension(filter *PropertyFilterExtension) *repository.PropertyFilterExtension {
	if filter == nil {
		return nil
	}

	repoFilter := &repository.PropertyFilterExtension{
		Amenities: filter.Amenities,
	}

	// Map property classes
	if len(filter.PropertyClasses) > 0 {
		repoFilter.PropertyClasses = make([]schema.PropertyClass, len(filter.PropertyClasses))
		for i, v := range filter.PropertyClasses {
			repoFilter.PropertyClasses[i] = schema.PropertyClass(v)
		}
	}

	// Map property conditions
	if len(filter.PropertyConditions) > 0 {
		repoFilter.PropertyConditions = make([]schema.PropertyCondition, len(filter.PropertyConditions))
		for i, v := range filter.PropertyConditions {
			repoFilter.PropertyConditions[i] = schema.PropertyCondition(v)
		}
	}

	return repoFilter
}

func mapListingSortBy(sortBy ListingSortBy) repository.ListingSortBy {
	switch sortBy {
	case ListingSortCreatedAt:
		return repository.ListingSortByCreatedAt
	case ListingSortUpdatedAt:
		return repository.ListingSortByUpdatedAt
	case ListingSortPublishedAt:
		return repository.ListingSortByPublishedAt
	default:
		return repository.ListingSortByCreatedAt
	}
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

func textChangedMaterially(oldVal, newVal string) bool {
	if oldVal == newVal {
		return false
	}
	normOld := normalizeText(oldVal)
	normNew := normalizeText(newVal)
	if normOld == normNew {
		return false
	}

	// Levenshtein-based ratio; treat small edits (<5% of max length) as minor.
	dist := levenshtein(normOld, normNew)
	maxLen := float64(max(len(normOld), len(normNew)))
	if maxLen == 0 {
		return false
	}
	ratio := float64(dist) / maxLen
	return ratio > 0.05
}

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	lenA := len(ar)
	lenB := len(br)

	if lenA == 0 {
		return lenB
	}
	if lenB == 0 {
		return lenA
	}

	dist := make([][]int, lenA+1)
	for i := range dist {
		dist[i] = make([]int, lenB+1)
	}

	for i := 0; i <= lenA; i++ {
		dist[i][0] = i
	}
	for j := 0; j <= lenB; j++ {
		dist[0][j] = j
	}

	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			dist[i][j] = minInt(
				dist[i-1][j]+1,      // deletion
				dist[i][j-1]+1,      // insertion
				dist[i-1][j-1]+cost, // substitution
			)
		}
	}
	return dist[lenA][lenB]
}

func minInt(a, b, c int) int {
	return min(a, min(b, c))
}

func (s *ServiceImpl) unpublishAndEnqueueModeration(ctx context.Context, existing *domain.Listing) error {
	if existing == nil || existing.ID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	// Reload with media for payload construction
	listing, err := s.ensureListing(ctx, existing.ID, true)
	if err != nil {
		return err
	}

	property, err := s.ensureProperty(ctx, listing.PropertyID)
	if err != nil {
		return err
	}

	if s.moderationHooks == nil {
		return fmt.Errorf("moderation hooks not configured")
	}

	// Build payload similar to PublishListingRequest
	listingPayload := map[string]any{
		"title":               listing.Title,
		"description":         listing.Description,
		"extra_description":   listing.ExtraDescription,
		"currency":            listing.Currency,
		"listing_type":        listing.ListingType,
		"address":             property.Address,
		"city":                property.City,
		"state":               property.State,
		"country":             property.Country,
		"features_commercial": property.FeaturesCommercial,
	}

	switch listing.ListingType {
	case domain.ListingRent:
		listingPayload["rental_terms"] = listing.RentalDetails.RentalTerms
	case domain.ListingSale:
		listingPayload["sale_ownership_title"] = listing.SaleDetails.OwnershipTitle
		listingPayload["sale_terms"] = listing.SaleDetails.SaleTerms
	case domain.ListingShortLet:
		// no-op
	}

	for i, media := range listing.Media {
		listingPayload[fmt.Sprintf("media_key_%d", i)] = media.Key
		listingPayload[fmt.Sprintf("media_caption_%d", i)] = media.Caption
		listingPayload[fmt.Sprintf("media_group_%d", i)] = media.Group
	}

	listingPayloadStr, err := serializeToJSON(listingPayload)
	if err != nil {
		return fmt.Errorf("failed to serialize listing payload: %w", err)
	}

	lc, err := s.GetListingCompleteness(ctx, listing.ID, listing.OwnerID)
	if err != nil {
		return fmt.Errorf("failed to get listing completeness: %w", err)
	}
	s.log.Info("listing completeness before unpublish", "listing_id", listing.ID, "completeness_score", float64(lc.CompletionScore))

	// Decide next status based on completeness.
	nextStatus := domain.StatusUnderReview
	if !lc.ReadyToPublish {
		nextStatus = domain.StatusRequiresUpdates
	}

	// Mark as unpublished with the chosen status.
	now := time.Now()
	updates := map[string]any{
		"status":               nextStatus,
		"latest_review_status": domain.ReviewPending,
		"published":            false,
		"published_at":         nil,
		"status_changed_at":    now,
	}

	if err := s.repo.PatchListing(ctx, listing.ID, updates); err != nil {
		return err
	}

	// Invalidate cache to ensure subsequent reads get the updated status
	publicID := s.getPropertyPublicID(ctx, listing.PropertyID)
	s.invalidateListingCache(ctx, listing.ID, listing.Slug, publicID)

	// If not ready, stop after unpublishing.
	if !lc.ReadyToPublish {
		s.log.Info("listing not ready to publish; skipped moderation", "listing_id", listing.ID, "status", nextStatus)
		return nil
	}

	// Enqueue text moderation
	if err := s.moderationHooks.EnqueueAIModeration(ctx, listing.ID, "listing_text", listingPayloadStr); err != nil {
		return fmt.Errorf("failed to enqueue text moderation: %w", err)
	}

	// Enqueue media moderation
	for _, media := range listing.Media {
		if media.Key == "" {
			continue
		}
		var contentType string
		switch media.Type {
		case domain.MediaTypeImage:
			contentType = "listing_image"
		case domain.MediaTypeVideo:
			contentType = "listing_video"
		default:
			continue
		}
		if err := s.moderationHooks.EnqueueAIModeration(ctx, listing.ID, contentType, media.Key); err != nil {
			return fmt.Errorf("failed to enqueue moderation for media %s: %w", media.ID, err)
		}
	}

	return nil
}

// shouldUnpublish decides if changes warrant unpublishing for re-moderation.
func shouldUnpublish(existing, updated *domain.Listing) bool {
	if existing == nil || updated == nil {
		return false
	}
	// Only consider live listings.
	if !existing.Published || existing.Status != domain.StatusActive {
		return false
	}

	// Critical fields with minor-change tolerance.
	if textChangedMaterially(existing.Title, updated.Title) {
		return true
	}
	if textChangedMaterially(existing.Description, updated.Description) {
		return true
	}
	if textChangedMaterially(existing.ExtraDescription, updated.ExtraDescription) {
		return true
	}

	// Structural/typed fields.
	if existing.ListingType != updated.ListingType {
		return true
	}
	if existing.Currency != updated.Currency {
		return true
	}

	return false
}

// shouldUnpublishPatch evaluates map updates for critical changes.
func shouldUnpublishPatch(existing *domain.Listing, updates map[string]any) bool {
	if existing == nil {
		return false
	}
	if !existing.Published || existing.Status != domain.StatusActive {
		return false
	}

	// Check critical text fields with minor-change tolerance.
	if v, ok := updates["title"]; ok {
		if newVal, ok2 := v.(string); ok2 && textChangedMaterially(existing.Title, newVal) {
			return true
		}
	}
	if v, ok := updates["description"]; ok {
		if newVal, ok2 := v.(string); ok2 && textChangedMaterially(existing.Description, newVal) {
			return true
		}
	}
	if v, ok := updates["extra_description"]; ok {
		if newVal, ok2 := v.(string); ok2 && textChangedMaterially(existing.ExtraDescription, newVal) {
			return true
		}
	}

	// Structural/typed fields.
	if v, ok := updates["listing_type"]; ok {
		if newVal, ok2 := v.(domain.ListingType); ok2 && existing.ListingType != newVal {
			return true
		}
		if newValStr, ok2 := v.(string); ok2 && string(existing.ListingType) != newValStr {
			return true
		}
	}
	if v, ok := updates["currency"]; ok {
		if newVal, ok2 := v.(domain.CurrencyCode); ok2 && existing.Currency != newVal {
			return true
		}
		if newValStr, ok2 := v.(string); ok2 && string(existing.Currency) != newValStr {
			return true
		}
	}

	return false
}
