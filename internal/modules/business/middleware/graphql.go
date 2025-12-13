package middleware

import (
	"context"
	"fmt"

	"hauslet/internal/modules/business/domain"
	"hauslet/internal/modules/business/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// GraphQLAuthHelper provides authorization helpers for GraphQL resolvers
type GraphQLAuthHelper struct {
	businessService service.BusinessService
}

// NewGraphQLAuthHelper creates a new GraphQL authorization helper
func NewGraphQLAuthHelper(businessService service.BusinessService) *GraphQLAuthHelper {
	return &GraphQLAuthHelper{
		businessService: businessService,
	}
}

// RequireMembership checks if the current user is a member of the specified business
// Returns the membership if successful, error otherwise
func (h *GraphQLAuthHelper) RequireMembership(ctx context.Context, businessID uuid.UUID) (*domain.BusinessMember, error) {
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
		if err == domain.ErrMemberNotFound {
			return nil, fmt.Errorf("forbidden: not a business member")
		}
		return nil, err
	}

	return membership, nil
}

// RequireOwnership checks if the current user is an owner of the specified business
func (h *GraphQLAuthHelper) RequireOwnership(ctx context.Context, businessID uuid.UUID) (*domain.BusinessMember, error) {
	membership, err := h.RequireMembership(ctx, businessID)
	if err != nil {
		return nil, err
	}

	if !membership.IsOwner() {
		return nil, fmt.Errorf("forbidden: owner access required")
	}

	return membership, nil
}

// RequireAdmin checks if the current user is an admin or owner of the specified business
func (h *GraphQLAuthHelper) RequireAdmin(ctx context.Context, businessID uuid.UUID) (*domain.BusinessMember, error) {
	membership, err := h.RequireMembership(ctx, businessID)
	if err != nil {
		return nil, err
	}

	if !membership.IsAdmin() && !membership.IsOwner() {
		return nil, fmt.Errorf("forbidden: admin access required")
	}

	return membership, nil
}

// RequirePermission checks if the current user has the specified permission in the business
func (h *GraphQLAuthHelper) RequirePermission(ctx context.Context, businessID uuid.UUID, permission string) (*domain.BusinessMember, error) {
	membership, err := h.RequireMembership(ctx, businessID)
	if err != nil {
		return nil, err
	}

	// Owners and admins have all permissions
	if membership.IsOwner() || membership.IsAdmin() {
		return membership, nil
	}

	// Check specific permission
	hasPermission := false
	switch permission {
	case "CanCreateListings":
		hasPermission = membership.Permissions.CanCreateListings
	case "CanEditListings":
		hasPermission = membership.Permissions.CanEditListings
	case "CanDeleteListings":
		hasPermission = membership.Permissions.CanDeleteListings
	case "CanPublishListings":
		hasPermission = membership.Permissions.CanPublishListings
	case "CanManageMedia":
		hasPermission = membership.Permissions.CanManageMedia
	case "CanViewAnalytics":
		hasPermission = membership.Permissions.CanViewAnalytics
	case "CanManageMembers":
		hasPermission = membership.Permissions.CanManageMembers
	case "CanEditBusiness":
		hasPermission = membership.Permissions.CanEditBusiness
	case "CanViewFinancials":
		hasPermission = membership.Permissions.CanViewFinancials
	default:
		return nil, fmt.Errorf("unknown permission: %s", permission)
	}

	if !hasPermission {
		return nil, fmt.Errorf("forbidden: %s permission required", permission)
	}

	return membership, nil
}

// CheckMembership checks if the current user is a member without returning error
// Returns membership and true if member, nil and false otherwise
func (h *GraphQLAuthHelper) CheckMembership(ctx context.Context, businessID uuid.UUID) (*domain.BusinessMember, bool) {
	membership, err := h.RequireMembership(ctx, businessID)
	if err != nil {
		return nil, false
	}
	return membership, true
}

// CheckOwnership checks if the current user is an owner without returning error
func (h *GraphQLAuthHelper) CheckOwnership(ctx context.Context, businessID uuid.UUID) bool {
	_, err := h.RequireOwnership(ctx, businessID)
	return err == nil
}

// CheckPermission checks if the current user has the specified permission without returning error
func (h *GraphQLAuthHelper) CheckPermission(ctx context.Context, businessID uuid.UUID, permission string) bool {
	_, err := h.RequirePermission(ctx, businessID, permission)
	return err == nil
}

// GetCurrentUserID returns the current user's ID from the viewer context
func (h *GraphQLAuthHelper) GetCurrentUserID(ctx context.Context) (uuid.UUID, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return uuid.Nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID")
	}

	return userID, nil
}

// EnrichContextWithBusiness loads business and membership into context
// Useful for batch operations to avoid repeated DB calls
func (h *GraphQLAuthHelper) EnrichContextWithBusiness(ctx context.Context, businessID uuid.UUID) (context.Context, error) {
	// Get business
	business, err := h.businessService.GetBusiness(ctx, businessID)
	if err != nil {
		return ctx, err
	}

	ctx = WithBusiness(ctx, business)

	// Try to get membership if user is authenticated
	v := viewer.FromContext(ctx)
	if v != nil && v.UserID != "" {
		userID, err := uuid.Parse(v.UserID)
		if err == nil {
			membership, err := h.businessService.GetMember(ctx, businessID, userID)
			if err == nil {
				ctx = WithMembership(ctx, membership)
				ctx = WithPermissions(ctx, &membership.Permissions)
			}
		}
	}

	return ctx, nil
}
