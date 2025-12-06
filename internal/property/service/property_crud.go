package service

import (
	"context"

	"hauslet/internal/property/domain"
	"hauslet/internal/property/repository"
	"hauslet/internal/property/repository/schema"

	"github.com/google/uuid"
)

// CreateProperty creates a new property with validation.
func (s *ServiceImpl) CreateProperty(ctx context.Context, p domain.Property) (*domain.Property, error) {
	if p.OwnerID == uuid.Nil {
		return nil, domain.ErrInvalidOwnerID
	}

	if p.Location != nil && !p.Location.Valid() {
		return nil, domain.ErrInvalidLocation
	}

	// Normalize nil slices
	if p.Amenities == nil {
		p.Amenities = []string{}
	}
	if p.FeaturesCommercial == nil {
		p.FeaturesCommercial = []string{}
	}

	// Set defaults
	if p.Units == 0 {
		p.Units = 1
	}

	schemaProperty := domain.MapPropertyToSchema(&p)
	if err := s.repo.CreateProperty(ctx, schemaProperty); err != nil {
		return nil, err
	}

	return domain.MapPropertyFromSchema(schemaProperty), nil
}

// UpdateProperty updates an existing property with validation.
func (s *ServiceImpl) UpdateProperty(ctx context.Context, p domain.Property) (*domain.Property, error) {
	if p.ID == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}

	if p.OwnerID == uuid.Nil {
		return nil, domain.ErrInvalidOwnerID
	}

	if p.Location != nil && !p.Location.Valid() {
		return nil, domain.ErrInvalidLocation
	}

	existing, err := s.ensureProperty(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	// Preserve created timestamp
	p.CreatedAt = existing.CreatedAt

	schemaProperty := domain.MapPropertyToSchema(&p)
	if err := s.repo.UpdateProperty(ctx, schemaProperty); err != nil {
		return nil, err
	}

	return domain.MapPropertyFromSchema(schemaProperty), nil
}

// PatchProperty applies partial updates to a property.
func (s *ServiceImpl) PatchProperty(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Property, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}

	// Validate property exists
	if _, err := s.ensureProperty(ctx, id); err != nil {
		return nil, err
	}

	// Validate location if being updated
	if loc, ok := updates["location"]; ok {
		if location, ok := loc.(*domain.Location); ok && location != nil {
			if !location.Valid() {
				return nil, domain.ErrInvalidLocation
			}
		}
	}

	if err := s.repo.PatchProperty(ctx, id, updates); err != nil {
		return nil, err
	}

	return s.ensureProperty(ctx, id)
}

// GetPropertyByID retrieves a property by its ID.
func (s *ServiceImpl) GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}

	return s.ensureProperty(ctx, id)
}

// GetPropertyByPublicID retrieves a property by its public ID.
func (s *ServiceImpl) GetPropertyByPublicID(ctx context.Context, publicID string) (*domain.Property, error) {
	if publicID == "" {
		return nil, domain.ErrInvalidPropertyID
	}

	p, err := s.repo.GetPropertyByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrPropertyNotFound
	}

	return domain.MapPropertyFromSchema(p), nil
}

// GetPropertiesByIDs retrieves multiple properties by their IDs.
func (s *ServiceImpl) GetPropertiesByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Property, error) {
	if len(ids) == 0 {
		return []domain.Property{}, nil
	}

	// Validate all IDs
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, domain.ErrInvalidPropertyID
		}
	}

	schemaProperties, err := s.repo.GetPropertiesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	return domain.MapPropertiesFromSchema(schemaProperties), nil
}

// ListProperties retrieves properties based on filter and pagination.
func (s *ServiceImpl) ListProperties(ctx context.Context, filter PropertyFilter, page Pagination) ([]domain.Property, int64, error) {
	repoFilter := mapPropertyFilterToRepo(filter)
	repoPagination := repository.Pagination{
		Limit:  page.Limit,
		Offset: page.Offset,
	}

	result, err := s.repo.ListProperties(ctx, repoFilter, repoPagination)
	if err != nil {
		return nil, 0, err
	}

	properties := domain.MapPropertiesFromSchema(result.Items)
	return properties, result.TotalCount, nil
}

// DeleteProperty deletes a property (soft or hard).
func (s *ServiceImpl) DeleteProperty(ctx context.Context, id uuid.UUID, hard bool) error {
	if id == uuid.Nil {
		return domain.ErrInvalidPropertyID
	}

	// Verify property exists
	if _, err := s.ensureProperty(ctx, id); err != nil {
		return err
	}

	if hard {
		return s.repo.HardDeleteProperty(ctx, id)
	}

	return s.repo.SoftDeleteProperty(ctx, id)
}

// mapPropertyFilterToRepo converts service filter to repository filter.
func mapPropertyFilterToRepo(filter PropertyFilter) repository.PropertyFilter {
	repoFilter := repository.PropertyFilter{
		OwnerID:        filter.OwnerID,
		City:           filter.City,
		State:          filter.State,
		MinBedrooms:    filter.MinBedrooms,
		MinBathrooms:   filter.MinBathrooms,
		IncludeDeleted: filter.IncludeDeleted,
		CreatedAfter:   filter.CreatedAfter,
		CreatedBefore:  filter.CreatedBefore,
		UpdatedAfter:   filter.UpdatedAfter,
		UpdatedBefore:  filter.UpdatedBefore,
	}

	// Map enums from domain to repository schema types (direct casting since values match)
	if filter.Country != nil {
		schemaCountry := schema.CountryCode(*filter.Country)
		repoFilter.Country = &schemaCountry
	}

	if len(filter.Classes) > 0 {
		repoFilter.Classes = make([]schema.PropertyClass, len(filter.Classes))
		for i, c := range filter.Classes {
			repoFilter.Classes[i] = schema.PropertyClass(c)
		}
	}

	if len(filter.Types) > 0 {
		repoFilter.Types = make([]schema.PropertyType, len(filter.Types))
		for i, t := range filter.Types {
			repoFilter.Types[i] = schema.PropertyType(t)
		}
	}

	if len(filter.Furnishings) > 0 {
		repoFilter.Furnishings = make([]schema.FurnishingType, len(filter.Furnishings))
		for i, f := range filter.Furnishings {
			repoFilter.Furnishings[i] = schema.FurnishingType(f)
		}
	}

	if len(filter.Conditions) > 0 {
		repoFilter.Conditions = make([]schema.PropertyCondition, len(filter.Conditions))
		for i, c := range filter.Conditions {
			repoFilter.Conditions[i] = schema.PropertyCondition(c)
		}
	}

	// Map sort fields
	repoFilter.SortBy = mapPropertySortBy(filter.SortBy)
	repoFilter.SortOrder = mapSortOrder(filter.SortOrder)

	return repoFilter
}

// mapPropertySortBy converts service sort field to repository sort field.
func mapPropertySortBy(sortBy PropertySortBy) repository.PropertySortBy {
	switch sortBy {
	case PropertySortCreatedAt:
		return repository.PropertySortByCreatedAt
	case PropertySortUpdatedAt:
		return repository.PropertySortByUpdatedAt
	case PropertySortBedrooms:
		return repository.PropertySortByBedrooms
	case PropertySortBathrooms:
		return repository.PropertySortByBathrooms
	default:
		return repository.PropertySortByCreatedAt
	}
}

// mapSortOrder converts service sort order to repository sort order.
func mapSortOrder(order SortOrder) repository.SortOrder {
	switch order {
	case SortAsc:
		return repository.SortOrderAsc
	case SortDesc:
		return repository.SortOrderDesc
	default:
		return repository.SortOrderDesc
	}
}
