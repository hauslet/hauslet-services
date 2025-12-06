package service

import (
	"context"

	"hauslet/internal/property/domain"

	"github.com/google/uuid"
)

// FindPropertiesNearPoint finds properties near a geographic point.
func (s *ServiceImpl) FindPropertiesNearPoint(ctx context.Context, query NearPointQuery, filter PropertyFilter) ([]ScoredResult[domain.Property], int64, error) {
	// TODO: Implement
	return nil, 0, nil
}

// FindPropertiesInBoundingBox finds properties within a bounding box.
func (s *ServiceImpl) FindPropertiesInBoundingBox(ctx context.Context, bbox BoundingBox, filter PropertyFilter, page Pagination) ([]domain.Property, int64, error) {
	// TODO: Implement
	return nil, 0, nil
}

// FindPropertiesInPolygon finds properties within a polygon.
func (s *ServiceImpl) FindPropertiesInPolygon(ctx context.Context, poly PolygonQuery, filter PropertyFilter) ([]domain.Property, int64, error) {
	// TODO: Implement
	return nil, 0, nil
}

// CalculatePropertyDistance calculates the distance between two properties.
func (s *ServiceImpl) CalculatePropertyDistance(ctx context.Context, propertyID1, propertyID2 uuid.UUID) (float64, error) {
	// TODO: Implement
	return 0, nil
}

// FindNearbyProperties finds properties near another property.
func (s *ServiceImpl) FindNearbyProperties(ctx context.Context, propertyID uuid.UUID, radiusMeters float64, limit int) ([]ScoredResult[domain.Property], error) {
	// TODO: Implement
	return nil, nil
}

// FindListingsNearPoint finds listings near a geographic point.
func (s *ServiceImpl) FindListingsNearPoint(ctx context.Context, query NearPointQuery, filter ListingFilter) ([]ScoredResult[domain.Listing], int64, error) {
	// TODO: Implement
	return nil, 0, nil
}

// FindListingsInBoundingBox finds listings within a bounding box.
func (s *ServiceImpl) FindListingsInBoundingBox(ctx context.Context, bbox BoundingBox, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error) {
	// TODO: Implement
	return nil, 0, nil
}
