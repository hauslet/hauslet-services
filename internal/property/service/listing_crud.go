package service

import (
	"context"
	"time"

	"hauslet/internal/property/domain"
	"hauslet/internal/property/repository"
	"hauslet/internal/property/repository/schema"

	"github.com/google/uuid"
)

// CreateListing creates a new listing with validation.
func (s *ServiceImpl) CreateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error) {
	if l.PropertyID == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}
	if l.OwnerID == uuid.Nil {
		return nil, domain.ErrInvalidOwnerID
	}

	// Ensure property exists
	exists, err := s.repo.PropertyExists(ctx, l.PropertyID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrPropertyNotFound
	}

	// Normalize media slice
	if l.Media == nil {
		l.Media = []domain.ListingMedia{}
	}

	schemaListing := domain.MapListingToSchema(&l)
	if schemaListing.ID == uuid.Nil {
		schemaListing.ID = uuid.New()
	}
	if schemaListing.Slug == "" {
		schemaListing.Slug = uuid.New().String()
	}

	if err := s.repo.CreateListing(ctx, schemaListing); err != nil {
		return nil, err
	}

	return domain.MapListingFromSchema(schemaListing), nil
}

// UpdateListing updates an existing listing with validation.
func (s *ServiceImpl) UpdateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error) {
	if l.ID == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}
	if l.PropertyID == uuid.Nil {
		return nil, domain.ErrInvalidPropertyID
	}
	if l.OwnerID == uuid.Nil {
		return nil, domain.ErrInvalidOwnerID
	}

	existing, err := s.ensureListing(ctx, l.ID, false)
	if err != nil {
		return nil, err
	}

	// Preserve immutable fields if not set
	if l.Slug == "" {
		l.Slug = existing.Slug
	}

	schemaListing := domain.MapListingToSchema(&l)
	if err := s.repo.UpdateListing(ctx, schemaListing); err != nil {
		return nil, err
	}

	return domain.MapListingFromSchema(schemaListing), nil
}

// PatchListing applies partial updates to a listing.
func (s *ServiceImpl) PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Listing, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	if err := s.repo.PatchListing(ctx, id, updates); err != nil {
		return nil, err
	}

	return s.ensureListing(ctx, id, false)
}

// GetListingByID retrieves a listing by its ID.
func (s *ServiceImpl) GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	return s.ensureListing(ctx, id, preloadMedia)
}

// GetListingBySlug retrieves a listing by its slug.
func (s *ServiceImpl) GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*domain.Listing, error) {
	if slug == "" {
		return nil, domain.ErrInvalidSlug
	}

	l, err := s.repo.GetListingBySlug(ctx, slug, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	return domain.MapListingFromSchema(l), nil
}

// GetListingsByIDs retrieves multiple listings by their IDs.
func (s *ServiceImpl) GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]domain.Listing, error) {
	if len(ids) == 0 {
		return []domain.Listing{}, nil
	}

	// Validate all IDs
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, domain.ErrInvalidListingID
		}
	}

	schemaListings, err := s.repo.GetListingsByIDs(ctx, ids, preloadMedia)
	if err != nil {
		return nil, err
	}

	return domain.MapListingsFromSchema(schemaListings), nil
}

// GetListingsByPropertyIDs retrieves all listings for multiple properties.
func (s *ServiceImpl) GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]domain.Listing, error) {
	if len(propertyIDs) == 0 {
		return []domain.Listing{}, nil
	}

	// Validate all IDs
	for _, id := range propertyIDs {
		if id == uuid.Nil {
			return nil, domain.ErrInvalidPropertyID
		}
	}

	schemaListings, err := s.repo.GetListingsByPropertyIDs(ctx, propertyIDs)
	if err != nil {
		return nil, err
	}

	return domain.MapListingsFromSchema(schemaListings), nil
}

// ListListings retrieves listings based on filter and pagination.
func (s *ServiceImpl) ListListings(ctx context.Context, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error) {
	repoFilter := mapListingFilterToRepo(filter)
	repoPagination := repository.Pagination{
		Limit:  page.Limit,
		Offset: page.Offset,
	}

	result, err := s.repo.ListListings(ctx, repoFilter, repoPagination)
	if err != nil {
		return nil, 0, err
	}

	listings := domain.MapListingsFromSchema(result.Items)
	return listings, result.TotalCount, nil
}

// ListListingsByProperty retrieves listings for a specific property.
func (s *ServiceImpl) ListListingsByProperty(ctx context.Context, propertyID uuid.UUID, page Pagination) ([]domain.Listing, int64, error) {
	if propertyID == uuid.Nil {
		return nil, 0, domain.ErrInvalidPropertyID
	}

	repoPagination := repository.Pagination{
		Limit:  page.Limit,
		Offset: page.Offset,
	}

	result, err := s.repo.ListListingsByPropertyID(ctx, propertyID, repoPagination)
	if err != nil {
		return nil, 0, err
	}

	return domain.MapListingsFromSchema(result.Items), result.TotalCount, nil
}

// DeleteListing deletes a listing (soft or hard).
func (s *ServiceImpl) DeleteListing(ctx context.Context, id uuid.UUID, hard bool) error {
	if id == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	if hard {
		return s.repo.HardDeleteListing(ctx, id)
	}

	return s.repo.SoftDeleteListing(ctx, id)
}

// UpdateListingStatus updates the status of a listing.
func (s *ServiceImpl) UpdateListingStatus(ctx context.Context, id uuid.UUID, status domain.ListingStatus, reason string, changedBy *uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	return s.repo.UpdateListingStatus(ctx, id, schema.ListingStatus(status), reason, changedBy)
}

// UpdatePublishState updates the publish state of a listing.
func (s *ServiceImpl) UpdatePublishState(ctx context.Context, id uuid.UUID, published bool, publishedAt *time.Time) error {
	if id == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	return s.repo.UpdatePublishState(ctx, id, published, publishedAt)
}

// PublishListing publishes a listing.
func (s *ServiceImpl) PublishListing(ctx context.Context, id uuid.UUID, publishedAt *time.Time, changedBy *uuid.UUID) (*domain.Listing, error) {
	if err := s.UpdatePublishState(ctx, id, true, publishedAt); err != nil {
		return nil, err
	}
	return s.ensureListing(ctx, id, false)
}

// UnpublishListing unpublishes a listing.
func (s *ServiceImpl) UnpublishListing(ctx context.Context, id uuid.UUID, changedBy *uuid.UUID) (*domain.Listing, error) {
	if err := s.UpdatePublishState(ctx, id, false, nil); err != nil {
		return nil, err
	}
	return s.ensureListing(ctx, id, false)
}

// IncrementListingView increments the view count for a listing.
func (s *ServiceImpl) IncrementListingView(ctx context.Context, id uuid.UUID, viewedAt time.Time) error {
	if id == uuid.Nil {
		return domain.ErrInvalidListingID
	}
	return s.repo.IncrementView(ctx, id, viewedAt)
}

// mapListingFilterToRepo converts service filters to repository filters.
func mapListingFilterToRepo(filter ListingFilter) repository.ListingFilter {
	repoFilter := repository.ListingFilter{
		OwnerID:         filter.OwnerID,
		PropertyID:      filter.PropertyID,
		Published:       filter.Published,
		HasCalendar:     filter.HasCalendar,
		IncludeDeleted:  filter.IncludeDeleted,
		CreatedAfter:    filter.CreatedAfter,
		CreatedBefore:   filter.CreatedBefore,
		PublishedAfter:  filter.PublishedAfter,
		PublishedBefore: filter.PublishedBefore,
		MinViewCount:    filter.MinViewCount,
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

	repoFilter.SortBy = mapListingSortBy(filter.SortBy)
	repoFilter.SortOrder = mapSortOrder(filter.SortOrder)

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
	case ListingSortViewCount:
		return repository.ListingSortByViewCount
	default:
		return repository.ListingSortByCreatedAt
	}
}

// BulkUpdateListingStatus updates the status of multiple listings.
func (s *ServiceImpl) BulkUpdateListingStatus(ctx context.Context, ids []uuid.UUID, status domain.ListingStatus, reason string, changedBy *uuid.UUID) error {
	// TODO: Implement
	return nil
}

// BulkArchiveListings archives multiple listings.
func (s *ServiceImpl) BulkArchiveListings(ctx context.Context, ids []uuid.UUID, changedBy *uuid.UUID) error {
	// TODO: Implement
	return nil
}

// OnModerationComplete handles the completion of moderation for a listing.
func (s *ServiceImpl) OnModerationComplete(ctx context.Context, listingID uuid.UUID, approved bool, reason string) error {
	// TODO: Implement
	return nil
}

// UpdateMediaReviewStatus updates the review status of media.
func (s *ServiceImpl) UpdateMediaReviewStatus(ctx context.Context, mediaID uuid.UUID, approved bool, reason string) error {
	// TODO: Implement
	return nil
}

// GetListingCompleteness calculates the completeness of a listing.
func (s *ServiceImpl) GetListingCompleteness(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID) (*domain.ListingCompleteness, error) {
	// TODO: Implement
	return nil, nil
}

// GetUserListings retrieves listings owned by a specific user.
func (s *ServiceImpl) GetUserListings(ctx context.Context, ownerID uuid.UUID, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error) {
	// TODO: Implement
	return nil, 0, nil
}

// SearchListings searches for listings using text similarity.
func (s *ServiceImpl) SearchListings(ctx context.Context, sim SimilarityQuery, filter ListingFilter) ([]ScoredResult[domain.Listing], error) {
	// TODO: Implement
	return nil, nil
}
