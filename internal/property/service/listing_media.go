package service

import (
	"context"
	"fmt"

	"hauslet/internal/property/domain"
	"hauslet/internal/property/repository/schema"

	"github.com/google/uuid"
)

// ReplaceListingMedia replaces all media for a listing.
func (s *ServiceImpl) ReplaceListingMedia(ctx context.Context, listingID uuid.UUID, media []domain.ListingMedia) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		return err
	}

	mapped, err := mapListingMediaToSchema(media)
	if err != nil {
		return err
	}

	return s.repo.ReplaceListingMedia(ctx, listingID, mapped)
}

// AddListingMedia adds media to a listing.
func (s *ServiceImpl) AddListingMedia(ctx context.Context, listingID uuid.UUID, media []domain.ListingMedia) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	if len(media) == 0 {
		return nil
	}

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		return err
	}

	mapped, err := mapListingMediaToSchema(media)
	if err != nil {
		return err
	}

	return s.repo.AddListingMedia(ctx, listingID, mapped)
}

// UpdateListingMedia updates specific media for a listing.
func (s *ServiceImpl) UpdateListingMedia(ctx context.Context, listingID uuid.UUID, mediaID uuid.UUID, updates map[string]any) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}
	if mediaID == uuid.Nil {
		return domain.ErrMediaNotFound
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		return err
	}

	return s.repo.UpdateListingMedia(ctx, listingID, mediaID, updates)
}

// DeleteListingMedia deletes media from a listing.
func (s *ServiceImpl) DeleteListingMedia(ctx context.Context, listingID uuid.UUID, mediaIDs []uuid.UUID) error {
	if listingID == uuid.Nil {
		return domain.ErrInvalidListingID
	}

	if len(mediaIDs) == 0 {
		return nil
	}

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		return err
	}

	return s.repo.DeleteListingMedia(ctx, listingID, mediaIDs)
}

// ListListingMedia retrieves all media for a listing.
func (s *ServiceImpl) ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error) {
	if listingID == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}

	if err := s.ensureListingExists(ctx, listingID); err != nil {
		return nil, err
	}

	media, err := s.repo.ListListingMedia(ctx, listingID)
	if err != nil {
		return nil, err
	}

	if len(media) == 0 {
		return []domain.ListingMedia{}, nil
	}

	result := make([]domain.ListingMedia, len(media))
	for i, m := range media {
		result[i] = *domain.MapListingMediaFromSchema(&m)
	}
	return result, nil
}

// ensureListingExists is a lightweight existence check for listing-scoped operations.
func (s *ServiceImpl) ensureListingExists(ctx context.Context, id uuid.UUID) error {
	exists, err := s.repo.ListingExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrListingNotFound
	}
	return nil
}

// mapListingMediaToSchema converts domain media to schema media, ensuring IDs are set.
func mapListingMediaToSchema(media []domain.ListingMedia) ([]schema.ListingMedia, error) {
	if len(media) == 0 {
		return []schema.ListingMedia{}, nil
	}

	mapped := make([]schema.ListingMedia, len(media))
	for i, m := range media {
		if m.URL == "" {
			return nil, fmt.Errorf("media url cannot be empty")
		}
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		schemaMedia := domain.MapListingMediaToSchema(&m)
		if schemaMedia == nil {
			return nil, fmt.Errorf("failed to map media at index %d", i)
		}
		mapped[i] = *schemaMedia
	}
	return mapped, nil
}
