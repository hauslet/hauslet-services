package service

import (
	"context"

	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/repository"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/platform/storage"

	"github.com/google/uuid"
)

// ServiceImpl implements the Service interface.
type ServiceImpl struct {
	repo             repository.Repository
	storage          *storage.R2Storage
	queue            *platformQueue.Client
	thumbnailSubject string
}

// NewPropertyService creates a new property service.
func NewPropertyService(repo repository.Repository, storage *storage.R2Storage, queue *platformQueue.Client, thumbnailSubject string) Service {
	return &ServiceImpl{
		repo:             repo,
		storage:          storage,
		queue:            queue,
		thumbnailSubject: thumbnailSubject,
	}
}

// ensureProperty fetches a property by ID, returning an error if not found.
func (s *ServiceImpl) ensureProperty(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	p, err := s.repo.GetPropertyByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrPropertyNotFound
	}

	return domain.MapPropertyFromSchema(p), nil
}

// ensureListing fetches a listing by ID, returning an error if not found.
func (s *ServiceImpl) ensureListing(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error) {
	l, err := s.repo.GetListingByID(ctx, id, preloadMedia)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.ErrListingNotFound
	}

	return domain.MapListingFromSchema(l), nil
}
