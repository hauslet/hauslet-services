package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/auth/authorization"
	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreatePropertyWithListing creates a property and its initial listing in a transaction.
func (s *ServiceImpl) CreatePropertyWithListing(ctx context.Context, p domain.Property, l domain.Listing) (*domain.Property, *domain.Listing, error) {
	var createdProperty *domain.Property
	var createdListing *domain.Listing

	if err := s.authorizeSupplyAction(ctx, authorization.SupplyActionCreateListing, &authorization.SupplyOptions{
		ListingType: string(l.ListingType),
	}); err != nil {
		return nil, nil, err
	}

	s.log.Info("creating property with listing", "owner_id", p.OwnerID)

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Create property within transaction
		propertySchema := domain.MapPropertyToSchema(&p)
		if err := s.repo.CreatePropertyTx(ctx, tx, propertySchema); err != nil {
			s.log.Error("failed to create property in transaction", "owner_id", p.OwnerID, "error", err)
			return fmt.Errorf("failed to create property: %w", err)
		}
		createdProperty = domain.MapPropertyFromSchema(propertySchema)

		// Ensure listing references created property and owner
		l.PropertyID = createdProperty.ID
		if l.OwnerID == uuid.Nil {
			l.OwnerID = createdProperty.OwnerID
		}

		// If business-owned, enforce permission
		if l.OwnerType == domain.OwnerBusiness {
			if s.businessAuthorizer == nil {
				return fmt.Errorf("business authorizer not configured")
			}
			if err := s.businessAuthorizer.CanCreateListing(ctx, l.OwnerID); err != nil {
				s.log.Warn("requester lacks create permission for business listing", "business_id", l.OwnerID)
				return err
			}
		}

		// Create listing within transaction
		listingSchema := domain.MapListingToSchema(&l)
		if listingSchema.Slug == "" {
			listingSchema.Slug = generateSlug(l.Title) + "_" + shortid()
		}
		if err := s.repo.CreateListingTx(ctx, tx, listingSchema); err != nil {
			s.log.Error("failed to create listing in transaction", "property_id", createdProperty.ID, "error", err)
			return fmt.Errorf("failed to create listing: %w", err)
		}
		createdListing = domain.MapListingFromSchema(listingSchema)

		return nil
	})

	if err != nil {
		s.log.Error("transaction failed for property with listing", "owner_id", p.OwnerID, "error", err)
		return nil, nil, err
	}

	s.log.Info("created property with listing", "property_id", createdProperty.ID, "listing_id", createdListing.ID, "owner_id", p.OwnerID)
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
		s.log.Error("listing not found for composite update", "listing_id", id, "error", err)
		return nil, err
	}

	if !isAdminRole(requesterRole) && existing.OwnerID != requesterID {
		// Allow business members (via authorizer) to update business-owned listings when they have edit permission.
		if existing.OwnerType == domain.OwnerBusiness {
			if s.businessAuthorizer == nil {
				s.log.Warn("business authorizer not configured for update listing", "listing_id", id)
				return nil, domain.ErrForbidden
			}
			if err := s.businessAuthorizer.CanEditListing(ctx, existing.OwnerID); err != nil {
				s.log.Warn("requester lacks edit permission for business listing", "requester_id", requesterID, "listing_id", id, "owner_id", existing.OwnerID)
				return nil, domain.ErrForbidden
			}
		} else {
			s.log.Warn("requester forbidden to update listing", "requester_id", requesterID, "listing_id", id, "owner_id", existing.OwnerID)
			return nil, domain.ErrForbidden
		}
	}

	if len(propertyUpdates) > 0 {
		if _, err := s.PatchProperty(ctx, existing.PropertyID, propertyUpdates); err != nil {
			s.log.Error("failed to patch property", "property_id", existing.PropertyID, "listing_id", id, "error", err)
			return nil, err
		}
	}

	listing := existing
	if len(listingUpdates) == 0 {
		// keep existing listing reference
	} else {
		updated, err := s.PatchListing(ctx, id, listingUpdates)
		if err != nil {
			s.log.Error("failed to patch listing", "listing_id", id, "error", err)
			return nil, err
		}
		listing = updated
	}

	return listing, nil
}

func isAdminRole(role string) bool {
	return role == "admin" || role == "root"
}
