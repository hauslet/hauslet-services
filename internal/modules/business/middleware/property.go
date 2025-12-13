package middleware

import (
	"context"
	"fmt"

	"hauslet/internal/modules/business/domain"
	"hauslet/internal/modules/business/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// PropertyAuthHelper provides authorization helpers for property operations
type PropertyAuthHelper struct {
	businessService service.BusinessService
}

// NewPropertyAuthHelper creates a new property authorization helper
func NewPropertyAuthHelper(businessService service.BusinessService) *PropertyAuthHelper {
	return &PropertyAuthHelper{
		businessService: businessService,
	}
}

// CanCreateListing checks if the user can create a listing for the business
func (h *PropertyAuthHelper) CanCreateListing(ctx context.Context, businessID uuid.UUID) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	// Check if user has permission
	hasPermission, err := h.businessService.HasPermission(ctx, userID, businessID, "CanCreateListings")
	if err != nil {
		return err
	}

	if !hasPermission {
		return domain.ErrInsufficientPermissions
	}

	return nil
}

// CanEditListing checks if the user can edit a listing for the business
func (h *PropertyAuthHelper) CanEditListing(ctx context.Context, businessID uuid.UUID) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	// Check if user has permission
	hasPermission, err := h.businessService.HasPermission(ctx, userID, businessID, "CanEditListings")
	if err != nil {
		return err
	}

	if !hasPermission {
		return domain.ErrInsufficientPermissions
	}

	return nil
}

// CanDeleteListing checks if the user can delete a listing for the business
func (h *PropertyAuthHelper) CanDeleteListing(ctx context.Context, businessID uuid.UUID) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	// Check if user has permission
	hasPermission, err := h.businessService.HasPermission(ctx, userID, businessID, "CanDeleteListings")
	if err != nil {
		return err
	}

	if !hasPermission {
		return domain.ErrInsufficientPermissions
	}

	return nil
}

// CanPublishListing checks if the user can publish a listing for the business
func (h *PropertyAuthHelper) CanPublishListing(ctx context.Context, businessID uuid.UUID) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	// Check if user has permission
	hasPermission, err := h.businessService.HasPermission(ctx, userID, businessID, "CanPublishListings")
	if err != nil {
		return err
	}

	if !hasPermission {
		return domain.ErrInsufficientPermissions
	}

	return nil
}

// CanManageMedia checks if the user can manage media for the business
func (h *PropertyAuthHelper) CanManageMedia(ctx context.Context, businessID uuid.UUID) error {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	// Check if user has permission
	hasPermission, err := h.businessService.HasPermission(ctx, userID, businessID, "CanManageMedia")
	if err != nil {
		return err
	}

	if !hasPermission {
		return domain.ErrInsufficientPermissions
	}

	return nil
}

// GetBusinessMembership retrieves the user's membership in the business
// Returns membership if user is a member, error otherwise
func (h *PropertyAuthHelper) GetBusinessMembership(ctx context.Context, businessID uuid.UUID) (*domain.BusinessMember, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	membership, err := h.businessService.GetMember(ctx, businessID, userID)
	if err != nil {
		return nil, err
	}

	return membership, nil
}
