package graphql

import (
	"context"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/business/domain"
	businessservice "hauslet/internal/modules/business/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles business-specific GraphQL fields
type Resolver struct {
	businessService businessservice.BusinessService
	log             *slog.Logger
}

func NewResolver(businessService businessservice.BusinessService, log *slog.Logger) *Resolver {
	return &Resolver{
		businessService: businessService,
		log:             log,
	}
}

// ===========================
// QUERIES
// ===========================

// Business retrieves a business by ID
func (r *Resolver) Business(ctx context.Context, id string) (*domain.Business, error) {
	businessID, err := parseUUID(id, "business")
	if err != nil {
		return nil, err
	}

	business, err := r.businessService.GetBusiness(ctx, businessID)
	if err != nil {
		r.log.Error("failed to get business", "business_id", id, "error", err)
		return nil, err
	}

	return business, nil
}

// BusinessBySlug retrieves a business by slug
func (r *Resolver) BusinessBySlug(ctx context.Context, slug string) (*domain.Business, error) {
	business, err := r.businessService.GetBusinessBySlug(ctx, slug)
	if err != nil {
		r.log.Error("failed to get business by slug", "slug", slug, "error", err)
		return nil, err
	}

	return business, nil
}

// AllBusinesses retrieves all businesses with pagination
func (r *Resolver) AllBusinesses(ctx context.Context, limit, offset *int) ([]domain.Business, error) {
	l := 20
	o := 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}

	businesses, err := r.businessService.ListAllBusinesses(ctx, l, o)
	if err != nil {
		r.log.Error("failed to list all businesses", "error", err)
		return nil, err
	}

	return businesses, nil
}

// SearchBusinesses searches businesses by name
func (r *Resolver) SearchBusinesses(ctx context.Context, query string, limit, offset *int) ([]domain.Business, error) {
	l := 20
	o := 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}

	businesses, err := r.businessService.SearchBusinesses(ctx, query, l, o)
	if err != nil {
		r.log.Error("failed to search businesses", "error", err)
		return nil, err
	}

	return businesses, nil
}

// BusinessMembers retrieves all members of a business
func (r *Resolver) BusinessMembers(ctx context.Context, businessID string) ([]domain.BusinessMember, error) {
	id, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	members, err := r.businessService.GetBusinessMembers(ctx, id)
	if err != nil {
		r.log.Error("failed to get business members", "error", err)
		return nil, err
	}

	return members, nil
}

// MyMemberships retrieves all memberships for the authenticated user
func (r *Resolver) MyMemberships(ctx context.Context) ([]domain.BusinessMember, error) {
	userID, err := r.getAuthenticatedUserID(ctx, "get memberships")
	if err != nil {
		return nil, err
	}

	memberships, err := r.businessService.GetUserMemberships(ctx, userID)
	if err != nil {
		r.log.Error("failed to get memberships for user", "user_id", userID, "error", err)
		return nil, err
	}

	return memberships, nil
}

// BusinessMember retrieves a specific member
func (r *Resolver) BusinessMember(ctx context.Context, businessID, userID string) (*domain.BusinessMember, error) {
	bid, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	uid, err := parseUUID(userID, "user")
	if err != nil {
		return nil, err
	}

	member, err := r.businessService.GetMember(ctx, bid, uid)
	if err != nil {
		r.log.Error("failed to get member", "error", err)
		return nil, err
	}

	return member, nil
}

// BusinessInvitations retrieves all invitations for a business
func (r *Resolver) BusinessInvitations(ctx context.Context, businessID string) ([]domain.BusinessInvitation, error) {
	id, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	invitations, err := r.businessService.GetBusinessInvitations(ctx, id)
	if err != nil {
		r.log.Error("failed to get business invitations", "error", err)
		return nil, err
	}

	return invitations, nil
}

// MyInvitations retrieves all invitations for a user email
func (r *Resolver) MyInvitations(ctx context.Context, email string) ([]domain.BusinessInvitation, error) {
	invitations, err := r.businessService.GetUserInvitations(ctx, email)
	if err != nil {
		r.log.Error("failed to get user invitations", "error", err)
		return nil, err
	}

	return invitations, nil
}

// MyBusinessPermissions retrieves the authenticated user's permissions for a business
func (r *Resolver) MyBusinessPermissions(ctx context.Context, businessID string) (*domain.MemberPermissions, error) {
	userID, err := r.getAuthenticatedUserID(ctx, "get permissions")
	if err != nil {
		return nil, err
	}

	bid, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	permissions, err := r.businessService.GetUserPermissions(ctx, userID, bid)
	if err != nil {
		r.log.Error("failed to get user permissions", "error", err)
		return nil, err
	}

	return permissions, nil
}

// ===========================
// MUTATIONS
// ===========================

// CreateBusiness creates a new business
func (r *Resolver) CreateBusiness(ctx context.Context, input CreateBusinessInput) (*domain.Business, error) {
	creatorID, err := r.getAuthenticatedUserID(ctx, "create business")
	if err != nil {
		return nil, err
	}

	// Convert GraphQL input to domain input
	domainInput := convertCreateBusinessInput(input)

	business, err := r.businessService.CreateBusiness(ctx, domainInput, creatorID)
	if err != nil {
		r.log.Error("failed to create business", "error", err)
		return nil, err
	}

	r.log.Info("business created", "business_id", business.ID, "user_id", creatorID)
	return business, nil
}

// UpdateBusiness updates an existing business
func (r *Resolver) UpdateBusiness(ctx context.Context, id string, input UpdateBusinessInput) (*domain.Business, error) {
	updaterID, err := r.getAuthenticatedUserID(ctx, "update business")
	if err != nil {
		return nil, err
	}

	businessID, err := parseUUID(id, "business")
	if err != nil {
		return nil, err
	}

	// Convert GraphQL input to domain input
	domainInput := convertUpdateBusinessInput(input)

	business, err := r.businessService.UpdateBusiness(ctx, businessID, domainInput, updaterID)
	if err != nil {
		r.log.Error("failed to update business", "error", err)
		return nil, err
	}

	return business, nil
}

// DeleteBusiness deletes a business
func (r *Resolver) DeleteBusiness(ctx context.Context, id string) (bool, error) {
	userID, err := r.getAuthenticatedUserID(ctx, "delete business")
	if err != nil {
		return false, err
	}

	businessID, err := parseUUID(id, "business")
	if err != nil {
		return false, err
	}

	if err := r.businessService.DeleteBusiness(ctx, businessID, userID); err != nil {
		r.log.Error("failed to delete business", "error", err)
		return false, err
	}

	return true, nil
}

// AddBusinessMember adds a member to a business
func (r *Resolver) AddBusinessMember(ctx context.Context, businessID, userID string, role domain.MemberRole, customPermissions *MemberPermissionsInput) (*domain.BusinessMember, error) {
	inviterID, err := r.getAuthenticatedUserID(ctx, "add business member")
	if err != nil {
		return nil, err
	}

	bid, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	uid, err := parseUUID(userID, "user")
	if err != nil {
		return nil, err
	}

	// Convert custom permissions if provided
	var domainPerms *domain.MemberPermissions
	if customPermissions != nil {
		domainPerms = convertMemberPermissionsInput(customPermissions)
	}

	member, err := r.businessService.AddMember(ctx, bid, uid, inviterID, role, domainPerms)
	if err != nil {
		r.log.Error("failed to add member", "error", err)
		return nil, err
	}

	return member, nil
}

// UpdateMemberRole updates a member's role
func (r *Resolver) UpdateMemberRole(ctx context.Context, businessID, memberID string, role domain.MemberRole) (*domain.BusinessMember, error) {
	updaterID, err := r.getAuthenticatedUserID(ctx, "update member role")
	if err != nil {
		return nil, err
	}

	bid, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	mid, err := parseUUID(memberID, "member")
	if err != nil {
		return nil, err
	}

	member, err := r.businessService.UpdateMemberRole(ctx, bid, mid, role, updaterID)
	if err != nil {
		r.log.Error("failed to update member role", "error", err)
		return nil, err
	}

	return member, nil
}

// UpdateMemberPermissions updates a member's permissions
func (r *Resolver) UpdateMemberPermissions(ctx context.Context, businessID, memberID string, permissions MemberPermissionsInput) (*domain.BusinessMember, error) {
	updaterID, err := r.getAuthenticatedUserID(ctx, "update member permissions")
	if err != nil {
		return nil, err
	}

	bid, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	mid, err := parseUUID(memberID, "member")
	if err != nil {
		return nil, err
	}

	domainPerms := *convertMemberPermissionsInput(&permissions)

	member, err := r.businessService.UpdateMemberPermissions(ctx, bid, mid, domainPerms, updaterID)
	if err != nil {
		r.log.Error("failed to update member permissions", "error", err)
		return nil, err
	}

	return member, nil
}

// RemoveMember removes a member from a business
func (r *Resolver) RemoveMember(ctx context.Context, businessID, memberID string) (bool, error) {
	removerID, err := r.getAuthenticatedUserID(ctx, "remove member")
	if err != nil {
		return false, err
	}

	bid, err := parseUUID(businessID, "business")
	if err != nil {
		return false, err
	}

	mid, err := parseUUID(memberID, "member")
	if err != nil {
		return false, err
	}

	if err := r.businessService.RemoveMember(ctx, bid, mid, removerID); err != nil {
		r.log.Error("failed to remove member", "error", err)
		return false, err
	}

	return true, nil
}

// InviteMember creates an invitation for a user to join a business
func (r *Resolver) InviteMember(ctx context.Context, businessID string, input InviteMemberInput) (*domain.BusinessInvitation, error) {
	inviterID, err := r.getAuthenticatedUserID(ctx, "invite member")
	if err != nil {
		return nil, err
	}

	bid, err := parseUUID(businessID, "business")
	if err != nil {
		return nil, err
	}

	var domainPerms *domain.MemberPermissions
	if input.CustomPermissions != nil {
		domainPerms = convertMemberPermissionsInput(input.CustomPermissions)
	}

	invitation, err := r.businessService.InviteUser(ctx, bid, input.Email, input.Role, inviterID, domainPerms)
	if err != nil {
		r.log.Error("failed to invite member", "error", err)
		return nil, err
	}

	return invitation, nil
}

// AcceptInvitation accepts an invitation and creates a membership
func (r *Resolver) AcceptInvitation(ctx context.Context, token string) (*domain.BusinessMember, error) {
	userID, err := r.getAuthenticatedUserID(ctx, "accept invitation")
	if err != nil {
		return nil, err
	}

	member, err := r.businessService.AcceptInvitation(ctx, token, userID)
	if err != nil {
		r.log.Error("failed to accept invitation", "error", err)
		return nil, err
	}

	return member, nil
}

// DeclineInvitation declines an invitation
func (r *Resolver) DeclineInvitation(ctx context.Context, token string) (bool, error) {
	userID, err := r.getAuthenticatedUserID(ctx, "decline invitation")
	if err != nil {
		return false, err
	}

	if err := r.businessService.DeclineInvitation(ctx, token, userID); err != nil {
		r.log.Error("failed to decline invitation", "error", err)
		return false, err
	}

	return true, nil
}

// RevokeInvitation revokes an invitation
func (r *Resolver) RevokeInvitation(ctx context.Context, invitationID string) (bool, error) {
	revokerID, err := r.getAuthenticatedUserID(ctx, "revoke invitation")
	if err != nil {
		return false, err
	}

	iid, err := parseUUID(invitationID, "invitation")
	if err != nil {
		return false, err
	}

	if err := r.businessService.RevokeInvitation(ctx, iid, revokerID); err != nil {
		r.log.Error("failed to revoke invitation", "error", err)
		return false, err
	}

	return true, nil
}

// ===========================
// HELPER FUNCTIONS
// ===========================

// Authentication Helpers

// getAuthenticatedUserID extracts and validates the user ID from context
func (r *Resolver) getAuthenticatedUserID(ctx context.Context, operation string) (uuid.UUID, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		r.log.Warn("unauthenticated attempt", "operation", operation)
		return uuid.Nil, err
	}

	return userID, nil
}

// parseUUID parses a UUID string with appropriate error message
func parseUUID(id string, entityType string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s ID", entityType)
	}
	return parsed, nil
}

// Input Conversion Helpers

func convertCreateBusinessInput(input CreateBusinessInput) domain.CreateBusinessInput {
	domainInput := domain.CreateBusinessInput{
		Name:               input.Name,
		DisplayName:        input.DisplayName,
		Description:        input.Description,
		BusinessType:       domain.BusinessType(input.BusinessType),
		RegistrationNumber: input.RegistrationNumber,
		TaxID:              input.TaxID,
		LegalEntityType:    input.LegalEntityType,
		Email:              input.Email,
		PhoneNumbers:       input.PhoneNumbers,
		Website:            input.Website,
		LogoURL:            input.LogoURL,
		CoverImageURL:      input.CoverImageURL,
		BrandColor:         input.BrandColor,
		BillingEmail:       input.BillingEmail,
	}

	// Convert address
	if input.Address != nil {
		domainInput.Address = domain.Address{
			HouseNumber:    input.Address.HouseNumber,
			Street:         input.Address.Street,
			Area:           input.Address.Area,
			LGA:            input.Address.Lga,
			City:           input.Address.City,
			State:          input.Address.State,
			Country:        input.Address.Country,
			PostalCode:     input.Address.PostalCode,
			District:       input.Address.District,
			DigitalAddress: input.Address.DigitalAddress,
		}
	}

	// Convert location
	if input.Location != nil {
		domainInput.Location = &domain.Location{
			Lat:  input.Location.Lat,
			Lng:  input.Location.Lng,
			SRID: 4326,
		}
	}

	return domainInput
}

func convertUpdateBusinessInput(input UpdateBusinessInput) domain.UpdateBusinessInput {
	domainInput := domain.UpdateBusinessInput{
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		Email:         input.Email,
		PhoneNumbers:  input.PhoneNumbers,
		Website:       input.Website,
		LogoURL:       input.LogoURL,
		CoverImageURL: input.CoverImageURL,
		BrandColor:    input.BrandColor,
		BillingEmail:  input.BillingEmail,
	}

	// Convert address
	if input.Address != nil {
		domainInput.Address = &domain.Address{
			HouseNumber:    input.Address.HouseNumber,
			Street:         input.Address.Street,
			Area:           input.Address.Area,
			LGA:            input.Address.Lga,
			City:           input.Address.City,
			State:          input.Address.State,
			Country:        input.Address.Country,
			PostalCode:     input.Address.PostalCode,
			District:       input.Address.District,
			DigitalAddress: input.Address.DigitalAddress,
		}
	}

	// Convert location
	if input.Location != nil {
		domainInput.Location = &domain.Location{
			Lat:  input.Location.Lat,
			Lng:  input.Location.Lng,
			SRID: 4326,
		}
	}

	return domainInput
}

func convertMemberPermissionsInput(input *MemberPermissionsInput) *domain.MemberPermissions {
	if input == nil {
		return nil
	}

	perms := &domain.MemberPermissions{}

	if input.CanCreateListings != nil {
		perms.CanCreateListings = *input.CanCreateListings
	}
	if input.CanEditListings != nil {
		perms.CanEditListings = *input.CanEditListings
	}
	if input.CanDeleteListings != nil {
		perms.CanDeleteListings = *input.CanDeleteListings
	}
	if input.CanPublishListings != nil {
		perms.CanPublishListings = *input.CanPublishListings
	}
	if input.CanManageMedia != nil {
		perms.CanManageMedia = *input.CanManageMedia
	}
	if input.CanViewAnalytics != nil {
		perms.CanViewAnalytics = *input.CanViewAnalytics
	}
	if input.CanManageMembers != nil {
		perms.CanManageMembers = *input.CanManageMembers
	}
	if input.CanEditBusiness != nil {
		perms.CanEditBusiness = *input.CanEditBusiness
	}
	if input.CanViewFinancials != nil {
		perms.CanViewFinancials = *input.CanViewFinancials
	}

	return perms
}
