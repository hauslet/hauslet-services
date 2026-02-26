package service

import (
	"slices"
	"context"
	"encoding/json"

	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/repository"
	"hauslet/internal/modules/property/repository/schema"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
		p.Amenities = []domain.AmenityGroup{}
	}
	if p.FeaturesCommercial == nil {
		p.FeaturesCommercial = []domain.AmenityGroup{}
	}

	// Set defaults
	if p.Units == 0 {
		p.Units = 1
	}

	schemaProperty := domain.MapPropertyToSchema(&p)
	if err := s.repo.CreateProperty(ctx, schemaProperty); err != nil {
		s.log.Error("failed to create property", "owner_id", p.OwnerID, "error", err)
		return nil, err
	}

	created := domain.MapPropertyFromSchema(schemaProperty)
	s.cacheProperty(ctx, created)

	s.log.Info("created property", "id", schemaProperty.ID, "owner_id", p.OwnerID, "type", p.PropertyType)
	return created, nil
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
		s.log.Error("property not found for update property", "id", p.ID, "error", err)
		return nil, err
	}

	// Normalize nil slices to empty groups for JSON serialization
	if p.Amenities == nil {
		p.Amenities = []domain.AmenityGroup{}
	}
	if p.FeaturesCommercial == nil {
		p.FeaturesCommercial = []domain.AmenityGroup{}
	}

	// Preserve created timestamp
	p.CreatedAt = existing.CreatedAt

	schemaProperty := domain.MapPropertyToSchema(&p)
	if err := s.repo.UpdateProperty(ctx, schemaProperty); err != nil {
		s.log.Error("failed to update property", "id", p.ID, "error", err)
		return nil, err
	}

	s.invalidatePropertyCache(ctx, p.ID)
	updated := domain.MapPropertyFromSchema(schemaProperty)
	s.cacheProperty(ctx, updated)

	s.log.Info("updated property", "id", p.ID)
	return updated, nil
}

// PatchProperty applies partial updates to a property.
func (s *ServiceImpl) PatchProperty(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Property, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}

	// Validate property exists
	if _, err := s.ensureProperty(ctx, id); err != nil {
		s.log.Error("property not found for patch property", "id", id, "error", err)
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

	// Normalize JSONB fields to proper JSON to avoid GORM passing record literals.
	if amenities, ok := updates["amenities"]; ok {
		if groups, ok := amenities.([]domain.AmenityGroup); ok {
			b, err := json.Marshal(domain.MapAmenityGroupsToSchema(groups))
			if err != nil {
				return nil, err
			}
			updates["amenities"] = datatypes.JSON(b)
		}
	}
	if feats, ok := updates["features_commercial"]; ok {
		if groups, ok := feats.([]domain.AmenityGroup); ok {
			b, err := json.Marshal(domain.MapAmenityGroupsToSchema(groups))
			if err != nil {
				return nil, err
			}
			updates["features_commercial"] = datatypes.JSON(b)
		}
	}

	if err := s.repo.PatchProperty(ctx, id, updates); err != nil {
		s.log.Error("failed to patch property", "id", id, "error", err)
		return nil, err
	}

	s.invalidatePropertyCache(ctx, id)

	s.log.Info("patched property", "id", id, "fields", len(updates))
	updated, err := s.ensureProperty(ctx, id)
	if err != nil {
		return nil, err
	}
	s.cacheProperty(ctx, updated)
	return updated, nil
}

// GetPropertyByID retrieves a property by its ID.
func (s *ServiceImpl) GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}

	var cached domain.Property
	if ok, err := s.getCachedValue(ctx, propertyCacheKey(id), &cached); err == nil && ok {
		return &cached, nil
	} else if err != nil {
		s.log.Warn("property cache read failed", "id", id, "error", err)
	}

	p, err := s.ensureProperty(ctx, id)
	if err != nil {
		return nil, err
	}
	s.cacheProperty(ctx, p)
	return p, nil
}

// GetPropertiesByIDs retrieves multiple properties by their IDs.
func (s *ServiceImpl) GetPropertiesByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Property, error) {
	if len(ids) == 0 {
		return []domain.Property{}, nil
	}

	// Validate all IDs
	if slices.Contains(ids, uuid.Nil) {
			return nil, domain.ErrInvalidPropertyID
		}

	schemaProperties, err := s.repo.GetPropertiesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	return domain.MapPropertiesFromSchema(schemaProperties), nil
}

// GetDistinctLocations returns distinct City, State, Country combinations for active properties.
func (s *ServiceImpl) GetDistinctLocations(ctx context.Context) ([]domain.LocationCombination, error) {
	locations, err := s.repo.GetDistinctLocations(ctx)
	if err != nil {
		s.log.Error("failed to get distinct locations from repo", "error", err)
		return nil, err
	}

	result := make([]domain.LocationCombination, 0, len(locations))
	for _, loc := range locations {
		result = append(result, domain.LocationCombination{
			City:    loc.City,
			State:   loc.State,
			Country: string(loc.Country),
		})
	}

	return result, nil
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
		s.log.Error("property not found for delete property", "id", id, "error", err)
		return err
	}

	if hard {
		s.log.Warn("hard deleting property", "id", id)
		if err := s.repo.HardDeleteProperty(ctx, id); err != nil {
			s.log.Error("failed to hard delete property", "id", id, "error", err)
			return err
		}
		s.log.Info("hard deleted property", "id", id)
		s.invalidatePropertyCache(ctx, id)
		return nil
	}

	if err := s.repo.SoftDeleteProperty(ctx, id); err != nil {
		s.log.Error("failed to soft delete property", "id", id, "error", err)
		return err
	}
	s.log.Info("soft deleted property", "id", id)
	s.invalidatePropertyCache(ctx, id)
	return nil
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
