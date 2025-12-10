package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreatePropertyWithListing creates a property and its initial listing in a transaction.
func (s *ServiceImpl) CreatePropertyWithListing(ctx context.Context, p domain.Property, l domain.Listing) (*domain.Property, *domain.Listing, error) {
	var createdProperty *domain.Property
	var createdListing *domain.Listing

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Create property within transaction
		propertySchema := domain.MapPropertyToSchema(&p)
		if err := s.repo.CreatePropertyTx(ctx, tx, propertySchema); err != nil {
			return fmt.Errorf("failed to create property: %w", err)
		}
		createdProperty = domain.MapPropertyFromSchema(propertySchema)

		// Ensure listing references created property and owner
		l.PropertyID = createdProperty.ID
		if l.OwnerID == uuid.Nil {
			l.OwnerID = createdProperty.OwnerID
		}

		// Create listing within transaction
		listingSchema := domain.MapListingToSchema(&l)
		if listingSchema.Slug == "" {
			listingSchema.Slug = generateSlug(l.Title) + "_" + shortid()
		}
		if err := s.repo.CreateListingTx(ctx, tx, listingSchema); err != nil {
			return fmt.Errorf("failed to create listing: %w", err)
		}
		createdListing = domain.MapListingFromSchema(listingSchema)

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return createdProperty, createdListing, nil
}
