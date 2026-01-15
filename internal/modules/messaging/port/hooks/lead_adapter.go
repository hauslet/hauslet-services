package hooks

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	leadsdomain "hauslet/internal/modules/leads/domain"
	leadsrepository "hauslet/internal/modules/leads/repository"
	leadsservice "hauslet/internal/modules/leads/service"
	messagingdomain "hauslet/internal/modules/messaging/domain"

	propertydomain "hauslet/internal/modules/property/domain"
	propertyrepository "hauslet/internal/modules/property/repository"
)

// LeadHooks defines the contract for interacting with the Leads module
type LeadHooks interface {
	// GetLeadParticipants returns the prospect (inquirer) and the owner/agent IDs
	GetLeadParticipants(ctx context.Context, leadID uuid.UUID) (prospectID, ownerID uuid.UUID, err error)

	// ValidateLeadAccess checks if a specific user is part of this lead
	ValidateLeadAccess(ctx context.Context, leadID, userID uuid.UUID) (bool, error)
}

// Ensure adapters conform to the hook contracts.
var _ messagingdomain.LeadHooks = (*leadHooksAdapter)(nil)

// leadHooksAdapter bridges messaging to the leads/property modules.
type leadHooksAdapter struct {
	leadRepo     leadsrepository.LeadRepository
	propertyRepo propertyrepository.Repository
	leadSvc      leadsservice.LeadService
}

// NewLeadHooksAdapter creates a new LeadHooks implementation.
func NewLeadHooksAdapter(
	leadRepo leadsrepository.LeadRepository,
	propertyRepo propertyrepository.Repository,
	leadSvc leadsservice.LeadService,
) messagingdomain.LeadHooks {
	return &leadHooksAdapter{
		leadRepo:     leadRepo,
		propertyRepo: propertyRepo,
		leadSvc:      leadSvc,
	}
}

func (a *leadHooksAdapter) GetLeadParticipants(ctx context.Context, leadID uuid.UUID) (prospectID, ownerID uuid.UUID, err error) {
	lead, err := a.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, uuid.Nil, fmt.Errorf("lead not found: %w", leadsdomain.ErrLeadNotFound)
		}
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to load lead %s: %w", leadID, err)
	}
	if lead == nil || lead.UserID == nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("lead %s missing user information", leadID)
	}
	prospectID = *lead.UserID

	if lead.AssignedTo != nil && *lead.AssignedTo != uuid.Nil {
		return prospectID, *lead.AssignedTo, nil
	}

	listing, err := a.propertyRepo.GetListingByID(ctx, lead.ListingID, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, uuid.Nil, fmt.Errorf("listing %s not found: %w", lead.ListingID, propertydomain.ErrListingNotFound)
		}
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to resolve listing %s for lead %s: %w", lead.ListingID, leadID, err)
	}
	if listing == nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("listing %s not found for lead %s", lead.ListingID, leadID)
	}
	if listing.OwnerID == uuid.Nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("listing %s missing owner", listing.ID)
	}

	return prospectID, listing.OwnerID, nil
}

func (a *leadHooksAdapter) ValidateLeadAccess(ctx context.Context, leadID, userID uuid.UUID) (bool, error) {
	if a.leadSvc == nil {
		return false, fmt.Errorf("lead service not configured")
	}

	_, err := a.leadSvc.GetLead(ctx, leadID, userID)
	if err != nil {
		if errors.Is(err, leadsdomain.ErrLeadNotFound) {
			return false, nil
		}
		if errors.Is(err, leadsdomain.ErrForbidden) || errors.Is(err, leadsdomain.ErrUnauthorized) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
