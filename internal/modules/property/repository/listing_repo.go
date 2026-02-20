package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hauslet/internal/modules/property/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateListing inserts a new listing record.
func (r *GormRepository) CreateListing(ctx context.Context, listing *schema.Listing) error {
	if listing == nil {
		return fmt.Errorf("listing cannot be nil")
	}

	// Ensure property exists before creating the listing.
	exists, err := r.PropertyExists(ctx, listing.PropertyID)
	if err != nil {
		return fmt.Errorf("failed to verify property for listing: %w", err)
	}
	if !exists {
		return fmt.Errorf("property %s not found: %w", listing.PropertyID, gorm.ErrRecordNotFound)
	}

	// Enforce one listing per property at the application layer (DB unique index also exists).
	var existing int64
	if err := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("property_id = ?", listing.PropertyID).
		Count(&existing).Error; err != nil {
		return fmt.Errorf("failed to check existing listing for property: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("listing already exists for property %s", listing.PropertyID)
	}

	if err := r.db.WithContext(ctx).Create(listing).Error; err != nil {
		return fmt.Errorf("failed to create listing: %w", err)
	}
	return nil
}

// CreateListingTx inserts a new listing record within a transaction.
func (r *GormRepository) CreateListingTx(ctx context.Context, tx *gorm.DB, listing *schema.Listing) error {
	if listing == nil {
		return fmt.Errorf("listing cannot be nil")
	}

	// Ensure property exists before creating the listing.
	var propertyCount int64
	if err := tx.WithContext(ctx).Model(&schema.Property{}).Where("id = ?", listing.PropertyID).Count(&propertyCount).Error; err != nil {
		return fmt.Errorf("failed to verify property for listing: %w", err)
	}
	if propertyCount == 0 {
		return fmt.Errorf("property %s not found: %w", listing.PropertyID, gorm.ErrRecordNotFound)
	}

	// Enforce one listing per property at the application layer (DB unique index also exists).
	var existing int64
	if err := tx.WithContext(ctx).
		Model(&schema.Listing{}).
		Where("property_id = ?", listing.PropertyID).
		Count(&existing).Error; err != nil {
		return fmt.Errorf("failed to check existing listing for property: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("listing already exists for property %s", listing.PropertyID)
	}

	if err := tx.WithContext(ctx).Create(listing).Error; err != nil {
		return fmt.Errorf("failed to create listing: %w", err)
	}
	return nil
}

// UpdateListing updates all fields on an existing listing.
func (r *GormRepository) UpdateListing(ctx context.Context, listing *schema.Listing) error {
	result := r.db.WithContext(ctx).Save(listing)
	if result.Error != nil {
		return fmt.Errorf("failed to update listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// PatchListing applies partial updates to a listing by ID.
func (r *GormRepository) PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	// Prepare a model instance so GORM hooks see the updated enum values instead of zero-values.
	listingModel := schema.Listing{ID: id}

	if status, ok := updates["status"]; ok {
		typed := schema.ListingStatus(fmt.Sprint(status))
		updates["status"] = typed
		listingModel.Status = typed
	}

	if latestReviewStatus, ok := updates["latest_review_status"]; ok {
		typed := schema.ReviewStatus(fmt.Sprint(latestReviewStatus))
		updates["latest_review_status"] = typed
		listingModel.LatestReviewStatus = typed
	}

	if currency, ok := updates["currency"]; ok {
		typed := schema.CurrencyCode(fmt.Sprint(currency))
		updates["currency"] = typed
		listingModel.Currency = typed
	}

	if ownerType, ok := updates["owner_type"]; ok {
		typed := schema.OwnerType(fmt.Sprint(ownerType))
		updates["owner_type"] = typed
		listingModel.OwnerType = typed
	}

	result := r.db.WithContext(ctx).
		Model(&listingModel).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to patch listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// GetListingByID fetches a listing by primary key.
func (r *GormRepository) GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*schema.Listing, error) {
	var listing schema.Listing
	query := r.db.WithContext(ctx)
	if preloadMedia {
		query = query.Preload("Media")
	}
	if err := query.First(&listing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("listing not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get listing: %w", err)
	}
	return &listing, nil
}

// GetListingWithPropertyByID fetches a listing along with its property in a single query.
func (r *GormRepository) GetListingWithPropertyByID(ctx context.Context, id uuid.UUID) (*schema.Listing, *schema.Property, error) {
	var listing schema.Listing
	if err := r.db.WithContext(ctx).
		Preload("Property").
		First(&listing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("listing not found: %w", err)
		}
		return nil, nil, fmt.Errorf("failed to get listing: %w", err)
	}

	var propertyCopy *schema.Property
	if listing.Property != nil {
		copy := *listing.Property
		propertyCopy = &copy
	}

	listing.Property = nil
	return &listing, propertyCopy, nil
}

// GetListingBySlug fetches a listing by slug.
func (r *GormRepository) GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*schema.Listing, error) {
	var listing schema.Listing
	query := r.db.WithContext(ctx)
	if preloadMedia {
		query = query.Preload("Media")
	}
	if err := query.First(&listing, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("listing not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get listing by slug: %w", err)
	}
	return &listing, nil
}

// GetListingByPublicID fetches a listing by the associated property's public ID.
func (r *GormRepository) GetListingByPublicID(ctx context.Context, publicID string, preloadMedia bool) (*schema.Listing, error) {
	var listing schema.Listing

	query := r.db.WithContext(ctx).
		Model(&schema.Listing{}).
		Joins("JOIN properties ON properties.id = listings.property_id").
		Where("properties.public_id = ?", publicID)
		// Preload("Property")

	if preloadMedia {
		query = query.Preload("Media")
	}

	if err := query.Select("listings.*").First(&listing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("listing not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get listing by public id: %w", err)
	}

	return &listing, nil
}

// ListListings returns listings that match the provided filter and pagination.
func (r *GormRepository) ListListings(ctx context.Context, filter ListingFilter, page Pagination) (*PaginatedResult[schema.Listing], error) {
	var listings []schema.Listing
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).Model(&schema.Listing{})
	if !filter.IncludeDeleted {
		baseQuery = baseQuery.Where("listings.deleted_at IS NULL")
	}
	baseQuery = applyListingFilter(baseQuery, filter)

	// When any property-level filters are present we must JOIN the properties
	// table and apply those predicates. Without this, city/state/geo/etc. on
	// ListingFilter are silently ignored.
	if needsPropertyJoin(filter) {
		baseQuery = baseQuery.Joins("JOIN properties ON properties.id = listings.property_id")
		baseQuery = applyPropertyFiltersForListing(baseQuery, filter)
	}

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count listings: %w", err)
	}

	// Apply sorting and pagination
	query := baseQuery
	query = applyListingSort(query, filter.SortBy, filter.SortOrder)
	query = applyPagination(query, page)

	if err := query.Preload("Media").Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to list listings: %w", err)
	}

	return &PaginatedResult[schema.Listing]{
		Items:      listings,
		TotalCount: totalCount,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}

// ListListingsByPropertyID returns all listings for a specific property.
func (r *GormRepository) ListListingsByPropertyID(ctx context.Context, propertyID uuid.UUID, page Pagination) (*PaginatedResult[schema.Listing], error) {
	var listings []schema.Listing
	var totalCount int64

	baseQuery := r.db.WithContext(ctx).Model(&schema.Listing{}).Where("property_id = ?", propertyID)

	// Get total count
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count listings by property: %w", err)
	}

	// Apply pagination
	query := baseQuery
	query = applyPagination(query, page)
	query = query.Order("created_at DESC")

	if err := query.Preload("Media").Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to list listings by property: %w", err)
	}

	return &PaginatedResult[schema.Listing]{
		Items:      listings,
		TotalCount: totalCount,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}

// ListingExists checks if a listing exists by ID.
func (r *GormRepository) ListingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&schema.Listing{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check listing existence: %w", err)
	}
	return count > 0, nil
}

// GetListingsByIDs fetches multiple listings by their IDs.
func (r *GormRepository) GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]schema.Listing, error) {
	if len(ids) == 0 {
		return []schema.Listing{}, nil
	}

	var listings []schema.Listing
	query := r.db.WithContext(ctx).Where("id IN ?", ids)
	if preloadMedia {
		query = query.Preload("Media")
	}

	if err := query.Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to get listings by IDs: %w", err)
	}

	return listings, nil
}

// GetListingsByPropertyIDs fetches all listings for multiple properties.
func (r *GormRepository) GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]schema.Listing, error) {
	if len(propertyIDs) == 0 {
		return []schema.Listing{}, nil
	}

	var listings []schema.Listing
	if err := r.db.WithContext(ctx).
		Where("property_id IN ?", propertyIDs).
		Order("created_at DESC").
		Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to get listings by property IDs: %w", err)
	}

	return listings, nil
}

// FindExpiredSuspensions returns listings with expired suspensions.
func (r *GormRepository) FindExpiredSuspensions(ctx context.Context, now time.Time) ([]schema.Listing, error) {
	var listings []schema.Listing
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("suspended_until IS NOT NULL AND suspended_until < ?", now).
		Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to list expired suspensions: %w", err)
	}
	return listings, nil
}

// FindSuspensionsEndingSoon returns listings with suspensions ending between from and to times.
// Excludes listings that have already been notified (suspension_ending_notified_at is set).
func (r *GormRepository) FindSuspensionsEndingSoon(ctx context.Context, from, to time.Time) ([]schema.Listing, error) {
	var listings []schema.Listing
	if err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("suspended_until IS NOT NULL AND suspended_until >= ? AND suspended_until < ?", from, to).
		Where("suspension_ending_notified_at IS NULL").
		Find(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to list suspensions ending soon: %w", err)
	}
	return listings, nil
}

// SearchListings performs a vector similarity search with optional filters.
// SearchListings performs semantic search if Query is provided; otherwise applies filters only.
func (r *GormRepository) SearchListings(ctx context.Context, embedding *schema.VectorEmbedding, filter ListingFilter, limit int) ([]ScoredListing, error) {
	if embedding == nil {
		return nil, fmt.Errorf("embedding is required")
	}

	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}

	var results []ScoredListing

	// Updated query for Hybrid Search (Vector + Native FTS)
	// We calculate two scores:
	// 1. Vector Score: (listings.text_embedding <=> ?) -> Cosine Distance (0=best, 2=worst)
	// 2. Text Score: ts_rank(search_vector, q) -> TF-IDF Score (Higher is better)
	//
	// We order by a combined rank to surface the best results from either method.
	query := r.db.WithContext(ctx).
		Table("listings").
		Select(`
			listings.*, 
			(listings.text_embedding <=> ?) AS vector_dist, 
			ts_rank(listings.search_vector, websearch_to_tsquery('english', ?)) AS text_score,
			ST_Y(properties.location::geometry) as lat, 
			ST_X(properties.location::geometry) as lng
		`, embedding, filter.Query).
		Joins("JOIN properties ON properties.id = listings.property_id")

	if !filter.IncludeDeleted {
		query = query.Where("listings.deleted_at IS NULL")
	}

	query = applyListingFilter(query, filter)

	// Apply type-specific JSONB filters
	query = applyShortletFilter(query, filter.ShortletFilter)
	query = applyRentalFilter(query, filter.RentalFilter)
	query = applySaleFilter(query, filter.SaleFilter)

	// Apply property extension filters
	query = applyPropertyExtensionFilter(query, filter.PropertyExtension)

	// Apply robust location/attribute filters
	if filter.City != nil && *filter.City != "" {
		query = query.Where("LOWER(properties.city) = LOWER(?)", *filter.City)
	}
	if filter.State != nil && *filter.State != "" {
		query = query.Where("LOWER(properties.state) = LOWER(?)", *filter.State)
	}
	if filter.Country != nil && *filter.Country != "" {
		query = query.Where("properties.country = ?", *filter.Country)
	}
	if len(filter.PropertyTypes) > 0 {
		query = query.Where("properties.property_type IN ?", filter.PropertyTypes)
	}
	if len(filter.Furnishings) > 0 {
		query = query.Where("properties.furnishing_type IN ?", filter.Furnishings)
	}
	if filter.MinBedrooms != nil {
		query = query.Where("properties.bedrooms >= ?", *filter.MinBedrooms)
	}
	if filter.MaxBedrooms != nil {
		query = query.Where("properties.bedrooms <= ?", *filter.MaxBedrooms)
	}
	if filter.MinBathrooms != nil {
		query = query.Where("properties.bathrooms >= ?", *filter.MinBathrooms)
	}
	if filter.MaxBathrooms != nil {
		query = query.Where("properties.bathrooms <= ?", *filter.MaxBathrooms)
	}
	if filter.Latitude != nil && filter.Longitude != nil && filter.RadiusMeters != nil && *filter.RadiusMeters > 0 {
		query = query.Where("properties.location IS NOT NULL").
			Where("ST_DWithin(properties.location, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)", *filter.Longitude, *filter.Latitude, *filter.RadiusMeters)
	}
	// Price filters
	if filter.MinPrice != nil {
		query = query.Where(
			r.db.Where("listing_type = ? AND (rental_details->>'rental_price')::numeric >= ?", schema.ListingRent, *filter.MinPrice).
				Or("listing_type = ? AND (shortlet_details->>'nightly_rate')::numeric >= ?", schema.ListingShortLet, *filter.MinPrice).
				Or("listing_type = ? AND (sale_details->>'sale_price')::numeric >= ?", schema.ListingSale, *filter.MinPrice),
		)
	}
	if filter.MaxPrice != nil {
		query = query.Where(
			r.db.Where("listing_type = ? AND (rental_details->>'rental_price')::numeric <= ?", schema.ListingRent, *filter.MaxPrice).
				Or("listing_type = ? AND (shortlet_details->>'nightly_rate')::numeric <= ?", schema.ListingShortLet, *filter.MaxPrice).
				Or("listing_type = ? AND (sale_details->>'sale_price')::numeric <= ?", schema.ListingSale, *filter.MaxPrice),
		)
	}
	if filter.Currency != nil {
		query = query.Where("currency = ?", *filter.Currency)
	}

	// Hybrid Search Logic:
	// We want to find listings that match EITHER vector similarity OR text similarity.
	// 1. Vector Match: Distance < 0.6 (fairly loose to capture concepts)
	// 2. Text Match: Matches websearch query in Title, Description, or ExtraDescription
	//
	// Then sort by refined hybrid score:
	// - Convert vector distance to similarity: (1 - dist/2)
	// - Combine with ts_rank
	if filter.Query != nil && *filter.Query != "" {
		query = query.Where(
			"(listings.text_embedding <=> ?) < 0.6 OR (listings.search_vector @@ websearch_to_tsquery('english', ?))",
			embedding, filter.Query,
		)

		// Postgres doesn't allow aliases in ORDER BY expressions, so we must repeat the calculation.
		query = query.Clauses(clause.OrderBy{
			Expression: clause.Expr{
				SQL:  "((1 - (listings.text_embedding <=> ?) / 2) + ts_rank(listings.search_vector, websearch_to_tsquery('english', ?))) DESC",
				Vars: []interface{}{embedding, filter.Query},
			},
		})
	} else {
		// Fallback for no-query search (e.g. "similar listings") - rely purely on vector distance.
		// Use manual expression to avoid alias issues in some Postgres versions/drivers.
		query = query.Order(gorm.Expr("listings.text_embedding <=> ? ASC", embedding))
	}

	// Limit results
	query = query.Limit(limit)

	// Use a dedicated result struct that flat-maps the columns we need.
	type SearchResultRow struct {
		schema.Listing
		VectorDist float64  `gorm:"column:vector_dist"`
		TextScore  float64  `gorm:"column:text_score"`
		Lat        *float64 `gorm:"column:lat"`
		Lng        *float64 `gorm:"column:lng"`
	}

	var rowsData []SearchResultRow

	// Preload Media relation for the embedded Listing
	if err := query.Preload("Media").Find(&rowsData).Error; err != nil {
		return nil, fmt.Errorf("failed to search listings: %w", err)
	}

	results = make([]ScoredListing, len(rowsData))
	for i, row := range rowsData {
		sl := ScoredListing{
			Listing:   row.Listing,
			Score:     row.VectorDist, // Used as SemanticScore (Distance)
			TextScore: row.TextScore,  // Used for fuzzy match boost
		}
		if row.Lat != nil && row.Lng != nil {
			sl.Location = schema.NewGeographyPoint(*row.Lat, *row.Lng)
		}
		results[i] = sl
	}

	return results, nil
}

// SoftDeleteListing marks a listing as deleted.
func (r *GormRepository) SoftDeleteListing(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&schema.Listing{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// HardDeleteListing permanently deletes a listing.
func (r *GormRepository) HardDeleteListing(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&schema.Listing{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to hard delete listing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("listing not found: %w", gorm.ErrRecordNotFound)
	}
	return nil
}

// applyShortletFilter applies shortlet-specific JSONB filters
func applyShortletFilter(db *gorm.DB, filter *ShortletFilter) *gorm.DB {
	if filter == nil {
		return db
	}

	// Extra guest fee range
	if filter.MinExtraGuestFee != nil {
		db = db.Where("(shortlet_details->>'extra_guest_fee')::float >= ?", *filter.MinExtraGuestFee)
	}
	if filter.MaxExtraGuestFee != nil {
		db = db.Where("(shortlet_details->>'extra_guest_fee')::float <= ?", *filter.MaxExtraGuestFee)
	}

	// Min nights range (filter listings based on their min_nights property)
	if filter.MinNightsMin != nil {
		db = db.Where("(shortlet_details->>'min_nights')::int >= ?", *filter.MinNightsMin)
	}
	if filter.MinNightsMax != nil {
		db = db.Where("(shortlet_details->>'min_nights')::int <= ?", *filter.MinNightsMax)
	}

	// Max nights range (filter listings based on their max_nights property)
	if filter.MaxNightsMin != nil {
		db = db.Where("(shortlet_details->>'max_nights')::int >= ?", *filter.MaxNightsMin)
	}
	if filter.MaxNightsMax != nil {
		db = db.Where("(shortlet_details->>'max_nights')::int <= ?", *filter.MaxNightsMax)
	}

	// Max guests capacity
	if filter.MinMaxGuests != nil {
		db = db.Where("(shortlet_details->>'max_guests')::int >= ?", *filter.MinMaxGuests)
	}
	if filter.BaseGuestCount != nil {
		db = db.Where("(shortlet_details->>'base_guest_count')::int = ?", *filter.BaseGuestCount)
	}

	// Check-in time range
	if filter.CheckInTimeAfter != nil {
		db = db.Where("shortlet_details->>'check_in_time' >= ?", *filter.CheckInTimeAfter)
	}
	if filter.CheckInTimeBefore != nil {
		db = db.Where("shortlet_details->>'check_in_time' <= ?", *filter.CheckInTimeBefore)
	}

	// Check-out time range
	if filter.CheckOutTimeAfter != nil {
		db = db.Where("shortlet_details->>'check_out_time' >= ?", *filter.CheckOutTimeAfter)
	}
	if filter.CheckOutTimeBefore != nil {
		db = db.Where("shortlet_details->>'check_out_time' <= ?", *filter.CheckOutTimeBefore)
	}

	// Accommodation types (IN clause)
	if len(filter.AccommodationTypes) > 0 {
		db = db.Where("shortlet_details->>'accommodation_type' IN ?", filter.AccommodationTypes)
	}

	return db
}

// applyRentalFilter applies rental-specific JSONB filters
func applyRentalFilter(db *gorm.DB, filter *RentalFilter) *gorm.DB {
	if filter == nil {
		return db
	}

	// Rental price periods
	if len(filter.RentalPricePeriods) > 0 {
		db = db.Where("rental_details->>'rental_price_period' IN ?", filter.RentalPricePeriods)
	}

	// Min rental period range
	if filter.MinRentalPeriodMin != nil {
		db = db.Where("(rental_details->>'min_rental_period')::int >= ?", *filter.MinRentalPeriodMin)
	}
	if filter.MinRentalPeriodMax != nil {
		db = db.Where("(rental_details->>'min_rental_period')::int <= ?", *filter.MinRentalPeriodMax)
	}

	// Max rental period range
	if filter.MaxRentalPeriodMin != nil {
		db = db.Where("(rental_details->>'max_rental_period')::int >= ?", *filter.MaxRentalPeriodMin)
	}
	if filter.MaxRentalPeriodMax != nil {
		db = db.Where("(rental_details->>'max_rental_period')::int <= ?", *filter.MaxRentalPeriodMax)
	}

	// Availability date range
	if filter.AvailableFrom != nil {
		db = db.Where("(rental_details->>'rental_availability_from')::timestamp >= ?", *filter.AvailableFrom)
	}
	if filter.AvailableTo != nil {
		db = db.Where("(rental_details->>'rental_availability_from')::timestamp <= ?", *filter.AvailableTo)
	}

	return db
}

// applySaleFilter applies sale-specific JSONB filters
func applySaleFilter(db *gorm.DB, filter *SaleFilter) *gorm.DB {
	if filter == nil {
		return db
	}

	// Ownership titles
	if len(filter.OwnershipTitles) > 0 {
		db = db.Where("sale_details->>'ownership_title' IN ?", filter.OwnershipTitles)
	}

	// Payment plan
	if filter.PaymentPlan != nil {
		db = db.Where("(sale_details->>'payment_plan')::boolean = ?", *filter.PaymentPlan)
	}

	return db
}

// applyPropertyExtensionFilter applies additional property filters
// Note: This should be called after the properties table is joined
func applyPropertyExtensionFilter(db *gorm.DB, filter *PropertyFilterExtension) *gorm.DB {
	if filter == nil {
		return db
	}

	// Property classes
	if len(filter.PropertyClasses) > 0 {
		db = db.Where("properties.property_class IN ?", filter.PropertyClasses)
	}

	// Property conditions
	if len(filter.PropertyConditions) > 0 {
		db = db.Where("properties.property_condition IN ?", filter.PropertyConditions)
	}

	// Amenities - ALL must be present (AND logic)
	// The amenities are stored as JSONB array of AmenityGroup objects
	// We need to check if all requested amenities exist in the flattened amenities
	for _, amenity := range filter.Amenities {
		// Check if the amenity exists anywhere in the amenities JSONB array
		db = db.Where(
			"EXISTS (SELECT 1 FROM jsonb_array_elements(properties.amenities) AS ag "+
				"WHERE EXISTS (SELECT 1 FROM jsonb_array_elements(ag->'amenities') AS a "+
				"WHERE a->>'name' = ?))",
			amenity,
		)
	}

	return db
}

// PatchShortletDetails merges partial updates into the existing ShortletDetails JSON.
// It implements "Read-Patch-Write" to ensure safety.
func (r *GormRepository) PatchShortletDetails(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		var listing schema.Listing
		if err := tx.First(&listing, "id = ?", id).Error; err != nil {
			return fmt.Errorf("listing not found: %w", err)
		}

		if listing.ShortletDetails == nil {
			// If nil, initialize empty
			listing.ShortletDetails = &schema.ShortletDetail{}
		}

		// Merge Logic:
		// 1. Convert PATCH map to JSON bytes
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return fmt.Errorf("failed to marshal patch: %w", err)
		}

		// 2. Unmarshal PATCH bytes INTO the existing struct
		// This respects the existing values while overwriting only what's in the patch
		if err := json.Unmarshal(patchBytes, listing.ShortletDetails); err != nil {
			return fmt.Errorf("failed to apply patch to details: %w", err)
		}

		// 3. Save specific column back to DB to avoid side effects
		// Use Select().Updates() to respect GORM serializer tags
		if err := tx.Model(&listing).Select("ShortletDetails").Updates(&listing).Error; err != nil {
			return fmt.Errorf("failed to save patched listing: %w", err)
		}

		return nil
	})
}

// PatchRentalDetails merges partial updates into the existing RentalDetails JSON.
func (r *GormRepository) PatchRentalDetails(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		var listing schema.Listing
		if err := tx.First(&listing, "id = ?", id).Error; err != nil {
			return fmt.Errorf("listing not found: %w", err)
		}

		if listing.RentalDetails == nil {
			listing.RentalDetails = &schema.RentalDetail{}
		}

		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return fmt.Errorf("failed to marshal patch: %w", err)
		}

		if err := json.Unmarshal(patchBytes, listing.RentalDetails); err != nil {
			return fmt.Errorf("failed to apply patch: %w", err)
		}

		if err := tx.Model(&listing).Select("RentalDetails").Updates(&listing).Error; err != nil {
			return fmt.Errorf("failed to save patched listing: %w", err)
		}
		return nil
	})
}

// PatchSaleDetails merges partial updates into the existing SaleDetails JSON.
func (r *GormRepository) PatchSaleDetails(ctx context.Context, id uuid.UUID, patch map[string]any) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		var listing schema.Listing
		if err := tx.First(&listing, "id = ?", id).Error; err != nil {
			return fmt.Errorf("listing not found: %w", err)
		}

		if listing.SaleDetails == nil {
			listing.SaleDetails = &schema.SaleDetail{}
		}

		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return fmt.Errorf("failed to marshal patch: %w", err)
		}

		if err := json.Unmarshal(patchBytes, listing.SaleDetails); err != nil {
			return fmt.Errorf("failed to apply patch: %w", err)
		}

		if err := tx.Model(&listing).Select("SaleDetails").Updates(&listing).Error; err != nil {
			return fmt.Errorf("failed to save patched listing: %w", err)
		}
		return nil
	})
}
