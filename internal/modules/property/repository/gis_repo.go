package repository

import (
	"context"
	"fmt"
	"hauslet/internal/modules/property/repository/schema"
	"strings"

	"github.com/google/uuid"
)

// =============================================================================
// GEOSPATIAL METHODS (PostGIS) - TODO: Implement
// =============================================================================

// FindPropertiesNearPoint finds properties within a radius of a geographic point.
func (r *GormRepository) FindPropertiesNearPoint(ctx context.Context, lat, lng, radiusMeters float64, filter PropertyFilter, page Pagination) (*PaginatedResult[ScoredResult[schema.Property]], error) {
	if radiusMeters <= 0 {
		return nil, fmt.Errorf("radiusMeters must be positive")
	}

	base := r.db.WithContext(ctx).Model(&schema.Property{}).Where("location IS NOT NULL").
		Where("ST_DWithin(location, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)", lng, lat, radiusMeters)
	base = applyPropertyFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count nearby properties: %w", err)
	}

	limit := page.Limit
	if limit <= 0 {
		limit = DefaultPaginationLimit
	}
	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}
	offset := page.Offset
	if offset < 0 {
		offset = 0
	}

	type row struct {
		schema.Property
		DistanceMeters float64 `gorm:"column:distance_meters"`
	}
	var rows []row
	if err := base.
		Select("properties.*, ST_Distance(location, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) AS distance_meters", lng, lat).
		Order("distance_meters ASC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to find nearby properties: %w", err)
	}

	results := make([]ScoredResult[schema.Property], 0, len(rows))
	for _, rrow := range rows {
		results = append(results, ScoredResult[schema.Property]{
			Item:           rrow.Property,
			DistanceMeters: rrow.DistanceMeters,
		})
	}

	return &PaginatedResult[ScoredResult[schema.Property]]{
		Items:      results,
		TotalCount: total,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// FindPropertiesInBoundingBox finds properties within a rectangular geographic area.
func (r *GormRepository) FindPropertiesInBoundingBox(ctx context.Context, bbox BoundingBox, filter PropertyFilter, page Pagination) (*PaginatedResult[schema.Property], error) {
	base := r.db.WithContext(ctx).Model(&schema.Property{}).Where("location IS NOT NULL").
		Where("ST_Contains(ST_MakeEnvelope(?, ?, ?, ?, 4326), location::geometry)",
			bbox.SouthWestLng, bbox.SouthWestLat, bbox.NorthEastLng, bbox.NorthEastLat)
	base = applyPropertyFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count properties in bounding box: %w", err)
	}

	base = applyPagination(base, page)
	var properties []schema.Property
	if err := base.Find(&properties).Error; err != nil {
		return nil, fmt.Errorf("failed to find properties in bounding box: %w", err)
	}

	return &PaginatedResult[schema.Property]{
		Items:      properties,
		TotalCount: total,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}

// FindPropertiesInPolygon finds properties within a custom polygon shape.
func (r *GormRepository) FindPropertiesInPolygon(ctx context.Context, polygonWKT string, filter PropertyFilter, page Pagination) (*PaginatedResult[schema.Property], error) {
	if strings.TrimSpace(polygonWKT) == "" {
		return nil, fmt.Errorf("polygonWKT cannot be empty")
	}

	base := r.db.WithContext(ctx).Model(&schema.Property{}).Where("location IS NOT NULL").
		Where("ST_Contains(ST_GeomFromText(?, 4326), location::geometry)", polygonWKT)
	base = applyPropertyFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count properties in polygon: %w", err)
	}

	base = applyPagination(base, page)
	var properties []schema.Property
	if err := base.Find(&properties).Error; err != nil {
		return nil, fmt.Errorf("failed to find properties in polygon: %w", err)
	}

	return &PaginatedResult[schema.Property]{
		Items:      properties,
		TotalCount: total,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}

// CalculatePropertyDistance calculates the distance in meters between two properties.
func (r *GormRepository) CalculatePropertyDistance(ctx context.Context, propertyID1, propertyID2 uuid.UUID) (float64, error) {
	var distance float64
	err := r.db.WithContext(ctx).
		Raw(`
			SELECT ST_Distance(p1.location, p2.location)
			FROM properties p1, properties p2
			WHERE p1.id = ? AND p2.id = ? AND p1.location IS NOT NULL AND p2.location IS NOT NULL
		`, propertyID1, propertyID2).
		Scan(&distance).Error
	if err != nil {
		return 0, fmt.Errorf("failed to calculate property distance: %w", err)
	}
	return distance, nil
}

// FindNearbyProperties finds properties near another property within a radius.
func (r *GormRepository) FindNearbyProperties(ctx context.Context, propertyID uuid.UUID, radiusMeters float64, limit int) ([]ScoredResult[schema.Property], error) {
	if radiusMeters <= 0 {
		return nil, fmt.Errorf("radiusMeters must be positive")
	}
	if limit <= 0 || limit > MaxPaginationLimit {
		limit = DefaultPaginationLimit
	}

	type row struct {
		schema.Property
		DistanceMeters float64 `gorm:"column:distance_meters"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT p2.*, ST_Distance(p2.location, p1.location) AS distance_meters
		FROM properties p1
		JOIN properties p2 ON p2.location IS NOT NULL
		WHERE p1.id = ?
		  AND p1.location IS NOT NULL
		  AND p2.id <> p1.id
		  AND ST_DWithin(p2.location, p1.location, ?)
		ORDER BY distance_meters ASC
		LIMIT ?
	`, propertyID, radiusMeters, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby properties: %w", err)
	}

	results := make([]ScoredResult[schema.Property], 0, len(rows))
	for _, rrow := range rows {
		results = append(results, ScoredResult[schema.Property]{
			Item:           rrow.Property,
			DistanceMeters: rrow.DistanceMeters,
		})
	}
	return results, nil
}

// FindListingsNearPoint finds listings (via property location) within a radius of a point.
func (r *GormRepository) FindListingsNearPoint(ctx context.Context, lat, lng, radiusMeters float64, filter ListingFilter, page Pagination) (*PaginatedResult[ScoredResult[schema.Listing]], error) {
	if radiusMeters <= 0 {
		return nil, fmt.Errorf("radiusMeters must be positive")
	}

	base := r.db.WithContext(ctx).Table("listings").
		Joins("JOIN properties ON properties.id = listings.property_id").
		Where("properties.location IS NOT NULL").
		Where("ST_DWithin(properties.location, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)", lng, lat, radiusMeters)
	base = applyListingFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count listings near point: %w", err)
	}

	limit := page.Limit
	if limit <= 0 {
		limit = DefaultPaginationLimit
	}
	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}
	offset := page.Offset
	if offset < 0 {
		offset = 0
	}

	type row struct {
		schema.Listing
		DistanceMeters float64 `gorm:"column:distance_meters"`
	}
	var rows []row
	if err := base.
		Select("listings.*, ST_Distance(properties.location, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) AS distance_meters", lng, lat).
		Order("distance_meters ASC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to find listings near point: %w", err)
	}

	results := make([]ScoredResult[schema.Listing], 0, len(rows))
	for _, rrow := range rows {
		results = append(results, ScoredResult[schema.Listing]{
			Item:           rrow.Listing,
			DistanceMeters: rrow.DistanceMeters,
		})
	}

	return &PaginatedResult[ScoredResult[schema.Listing]]{
		Items:      results,
		TotalCount: total,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// FindListingsInBoundingBox finds listings within a rectangular geographic area.
func (r *GormRepository) FindListingsInBoundingBox(ctx context.Context, bbox BoundingBox, filter ListingFilter, page Pagination) (*PaginatedResult[schema.Listing], error) {
	base := r.db.WithContext(ctx).Table("listings").
		Joins("JOIN properties ON properties.id = listings.property_id").
		Where("properties.location IS NOT NULL").
		Where("ST_Contains(ST_MakeEnvelope(?, ?, ?, ?, 4326), properties.location::geometry)",
			bbox.SouthWestLng, bbox.SouthWestLat, bbox.NorthEastLng, bbox.NorthEastLat)
	base = applyListingFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count listings in bounding box: %w", err)
	}

	base = applyPagination(base, page)
	var listings []schema.Listing
	if err := base.Select("listings.*").Scan(&listings).Error; err != nil {
		return nil, fmt.Errorf("failed to find listings in bounding box: %w", err)
	}

	return &PaginatedResult[schema.Listing]{
		Items:      listings,
		TotalCount: total,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}, nil
}
