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

	s.log.Logf("INFO creating property with listing owner=%s", p.OwnerID)

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Create property within transaction
		propertySchema := domain.MapPropertyToSchema(&p)
		if err := s.repo.CreatePropertyTx(ctx, tx, propertySchema); err != nil {
			s.log.Logf("ERROR failed to create property in transaction owner=%s: %v", p.OwnerID, err)
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
			s.log.Logf("ERROR failed to create listing in transaction property=%s: %v", createdProperty.ID, err)
			return fmt.Errorf("failed to create listing: %w", err)
		}
		createdListing = domain.MapListingFromSchema(listingSchema)

		return nil
	})

	if err != nil {
		s.log.Logf("ERROR transaction failed for property with listing owner=%s: %v", p.OwnerID, err)
		return nil, nil, err
	}

	s.log.Logf("INFO created property=%s with listing=%s owner=%s", createdProperty.ID, createdListing.ID, p.OwnerID)
	return createdProperty, createdListing, nil
}

// UpdateListingWithProperty updates a listing and optionally its property with ownership and admin checks.
func (s *ServiceImpl) UpdateListingWithProperty(ctx context.Context, id uuid.UUID, listingUpdates map[string]any, propertyUpdates map[string]any, requesterID uuid.UUID, requesterRole string) (*domain.Listing, error) {
	if id == uuid.Nil {
		return nil, domain.ErrInvalidListingID
	}
	if requesterID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	existing, err := s.ensureListing(ctx, id, false)
	if err != nil {
		s.log.Logf("ERROR listing not found for composite update listing=%s: %v", id, err)
		return nil, err
	}

	if !isAdminRole(requesterRole) && existing.OwnerID != requesterID {
		s.log.Logf("WARN requester=%s forbidden to update listing=%s owner=%s", requesterID, id, existing.OwnerID)
		return nil, domain.ErrForbidden
	}

	if len(propertyUpdates) > 0 {
		if _, err := s.PatchProperty(ctx, existing.PropertyID, propertyUpdates); err != nil {
			s.log.Logf("ERROR failed to patch property=%s for listing update listing=%s: %v", existing.PropertyID, id, err)
			return nil, err
		}
	}

	listing := existing
	if len(listingUpdates) == 0 {
		// keep existing listing reference
	} else {
		updated, err := s.PatchListing(ctx, id, listingUpdates)
		if err != nil {
			s.log.Logf("ERROR failed to patch listing=%s: %v", id, err)
			return nil, err
		}
		listing = updated
	}

	return listing, nil
}

func isAdminRole(role string) bool {
	return role == "admin" || role == "root"
}
