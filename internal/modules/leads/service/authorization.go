package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/leads/domain"

	"github.com/google/uuid"
)

// CanViewLead checks if a user can view a specific lead
// A user can view a lead if:
// - They are assigned to the lead
// - They own the listing the lead is for
// - They are a member of the business that owns the listing (if applicable)
func (s *ServiceImpl) CanViewLead(ctx context.Context, lead *domain.Lead, requesterID uuid.UUID) error {
	// Lead is assigned to requester
	if lead.AssignedTo != nil && *lead.AssignedTo == requesterID {
		return nil
	}

	// Get listing ownership information
	ownerInfo, err := s.propertyHooks.GetListingOwner(ctx, lead.ListingID)
	if err != nil {
		return fmt.Errorf("failed to get listing owner: %w", err)
	}

	// Requester owns the listing directly
	if ownerInfo.OwnerType == "user" && ownerInfo.OwnerID == requesterID {
		return nil
	}

	// Requester is a member of the business that owns the listing
	if ownerInfo.OwnerType == "business" && ownerInfo.BusinessID != nil {
		if s.businessHooks != nil {
			isMember, err := s.businessHooks.IsMember(ctx, requesterID, *ownerInfo.BusinessID)
			if err != nil {
				return fmt.Errorf("failed to check business membership: %w", err)
			}
			if isMember {
				return nil
			}
		}
	}

	return domain.ErrForbidden
}

// CanManageLead checks if a user can manage a lead (update status, assign, delete)
// A user can manage a lead if:
// - They own the listing the lead is for
// - They are an admin/owner of the business that owns the listing
func (s *ServiceImpl) CanManageLead(ctx context.Context, lead *domain.Lead, requesterID uuid.UUID) error {
	// Get listing ownership information
	ownerInfo, err := s.propertyHooks.GetListingOwner(ctx, lead.ListingID)
	if err != nil {
		return fmt.Errorf("failed to get listing owner: %w", err)
	}

	// Requester owns the listing directly
	if ownerInfo.OwnerType == "user" && ownerInfo.OwnerID == requesterID {
		return nil
	}

	// Requester has manage permissions in the business
	if ownerInfo.OwnerType == "business" && ownerInfo.BusinessID != nil {
		if s.businessHooks != nil {
			hasPermission, err := s.businessHooks.HasPermission(ctx, requesterID, *ownerInfo.BusinessID, "CanManageLeads")
			if err != nil {
				return fmt.Errorf("failed to check business permission: %w", err)
			}
			if hasPermission {
				return nil
			}
		}
	}

	return domain.ErrForbidden
}

// CanAssignLead checks if a user can assign a lead to someone
// Similar to CanManageLead, but may have different permission requirements
func (s *ServiceImpl) CanAssignLead(ctx context.Context, lead *domain.Lead, requesterID uuid.UUID) error {
	// For now, use the same logic as CanManageLead
	// In the future, this could have different permission requirements
	return s.CanManageLead(ctx, lead, requesterID)
}

// CanListLeadsForListing checks if a user can list leads for a specific listing
func (s *ServiceImpl) CanListLeadsForListing(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID) error {
	// Get listing ownership information
	ownerInfo, err := s.propertyHooks.GetListingOwner(ctx, listingID)
	if err != nil {
		return fmt.Errorf("failed to get listing owner: %w", err)
	}

	// Requester owns the listing directly
	if ownerInfo.OwnerType == "user" && ownerInfo.OwnerID == requesterID {
		return nil
	}

	// Requester is a member of the business that owns the listing
	if ownerInfo.OwnerType == "business" && ownerInfo.BusinessID != nil {
		if s.businessHooks != nil {
			isMember, err := s.businessHooks.IsMember(ctx, requesterID, *ownerInfo.BusinessID)
			if err != nil {
				return fmt.Errorf("failed to check business membership: %w", err)
			}
			if isMember {
				return nil
			}
		}
	}

	return domain.ErrForbidden
}

// CanListLeadsForBusiness checks if a user can list leads for a business
func (s *ServiceImpl) CanListLeadsForBusiness(ctx context.Context, businessID uuid.UUID, requesterID uuid.UUID) error {
	if s.businessHooks == nil {
		return domain.ErrForbidden
	}

	isMember, err := s.businessHooks.IsMember(ctx, requesterID, businessID)
	if err != nil {
		return fmt.Errorf("failed to check business membership: %w", err)
	}

	if !isMember {
		return domain.ErrNotBusinessMember
	}

	return nil
}
