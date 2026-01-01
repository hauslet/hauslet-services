package service

import (
	"context"
	"fmt"

	businessservice "hauslet/internal/modules/business/service"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// PropertyHooksAdapter implements PropertyHooks by wrapping PropertyService
type PropertyHooksAdapter struct {
	propertySvc propertyservice.PropertyService
}

// NewPropertyHooksAdapter creates a new PropertyHooksAdapter
func NewPropertyHooksAdapter(propertySvc propertyservice.PropertyService) PropertyHooks {
	return &PropertyHooksAdapter{
		propertySvc: propertySvc,
	}
}

// ListingExists checks if a listing exists and is active
func (a *PropertyHooksAdapter) ListingExists(ctx context.Context, listingID uuid.UUID) (bool, error) {
	listing, err := a.propertySvc.GetListingByID(ctx, listingID, false)
	if err != nil {
		return false, nil // Listing doesn't exist or error occurred
	}

	// Check if listing is active/published
	return listing != nil && (listing.Status == "active" || listing.Status == "published"), nil
}

// GetListingOwner retrieves ownership information for a listing
func (a *PropertyHooksAdapter) GetListingOwner(ctx context.Context, listingID uuid.UUID) (*ListingOwnerInfo, error) {
	listing, err := a.propertySvc.GetListingByID(ctx, listingID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get listing: %w", err)
	}

	if listing == nil {
		return nil, fmt.Errorf("listing not found")
	}

	ownerInfo := &ListingOwnerInfo{
		OwnerID:   listing.OwnerID,
		OwnerType: string(listing.OwnerType),
	}

	// If owner is a business, OwnerID is the business ID
	if listing.OwnerType == "business" {
		ownerInfo.BusinessID = &listing.OwnerID
	}

	return ownerInfo, nil
}

// BusinessHooksAdapter implements BusinessHooks by wrapping BusinessService
type BusinessHooksAdapter struct {
	businessSvc businessservice.BusinessService
}

// NewBusinessHooksAdapter creates a new BusinessHooksAdapter
func NewBusinessHooksAdapter(businessSvc businessservice.BusinessService) BusinessHooks {
	return &BusinessHooksAdapter{
		businessSvc: businessSvc,
	}
}

// HasPermission checks if a user has a specific permission in a business
func (a *BusinessHooksAdapter) HasPermission(ctx context.Context, userID, businessID uuid.UUID, permission string) (bool, error) {
	// Use the built-in HasPermission method from BusinessService
	hasPermission, err := a.businessSvc.HasPermission(ctx, userID, businessID, permission)
	if err != nil {
		return false, nil // Error checking permission
	}

	return hasPermission, nil
}

// IsMember checks if a user is a member of a business
func (a *BusinessHooksAdapter) IsMember(ctx context.Context, userID, businessID uuid.UUID) (bool, error) {
	member, err := a.businessSvc.GetMember(ctx, businessID, userID)
	if err != nil {
		return false, nil // Not a member
	}

	return member != nil && member.IsActive, nil
}

// GetBusinessMembers retrieves all members of a business (for auto-assignment)
func (a *BusinessHooksAdapter) GetBusinessMembers(ctx context.Context, businessID uuid.UUID) ([]uuid.UUID, error) {
	members, err := a.businessSvc.GetBusinessMembers(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to get business members: %w", err)
	}

	var memberIDs []uuid.UUID
	for _, member := range members {
		if member.IsActive {
			memberIDs = append(memberIDs, member.UserID)
		}
	}

	return memberIDs, nil
}
