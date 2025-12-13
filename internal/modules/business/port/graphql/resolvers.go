package graphql

import (
	"context"
	"fmt"

	"hauslet/internal/modules/business/domain"
	businessservice "hauslet/internal/modules/business/service"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// Resolver handles business-specific GraphQL fields
type Resolver struct {
	businessService businessservice.BusinessService
	log             *lgr.Logger
}

func NewResolver(businessService businessservice.BusinessService, log *lgr.Logger) *Resolver {
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
	businessID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	business, err := r.businessService.GetBusiness(ctx, businessID)
	if err != nil {
		r.log.Logf("ERROR Failed to get business %s: %v", id, err)
		return nil, err
	}

	return business, nil
}

// BusinessBySlug retrieves a business by slug
func (r *Resolver) BusinessBySlug(ctx context.Context, slug string) (*domain.Business, error) {
	business, err := r.businessService.GetBusinessBySlug(ctx, slug)
	if err != nil {
		r.log.Logf("ERROR Failed to get business by slug %s: %v", slug, err)
		return nil, err
	}

	return business, nil
}

// MyBusinesses retrieves all businesses the authenticated user is a member of
func (r *Resolver) MyBusinesses(ctx context.Context) ([]domain.Business, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to get businesses")
		return nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	businesses, err := r.businessService.ListUserBusinesses(ctx, userID)
	if err != nil {
		r.log.Logf("ERROR Failed to get businesses for user %s: %v", v.UserID, err)
		return nil, err
	}

	return businesses, nil
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
		r.log.Logf("ERROR Failed to list all businesses: %v", err)
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
		r.log.Logf("ERROR Failed to search businesses: %v", err)
		return nil, err
	}

	return businesses, nil
}

// BusinessMembers retrieves all members of a business
func (r *Resolver) BusinessMembers(ctx context.Context, businessID string) ([]domain.BusinessMember, error) {
	id, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	members, err := r.businessService.GetBusinessMembers(ctx, id)
	if err != nil {
		r.log.Logf("ERROR Failed to get business members: %v", err)
		return nil, err
	}

	return members, nil
}

// MyMemberships retrieves all memberships for the authenticated user
func (r *Resolver) MyMemberships(ctx context.Context) ([]domain.BusinessMember, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to get memberships")
		return nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	memberships, err := r.businessService.GetUserMemberships(ctx, userID)
	if err != nil {
		r.log.Logf("ERROR Failed to get memberships for user %s: %v", v.UserID, err)
		return nil, err
	}

	return memberships, nil
}

// BusinessMember retrieves a specific member
func (r *Resolver) BusinessMember(ctx context.Context, businessID, userID string) (*domain.BusinessMember, error) {
	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	member, err := r.businessService.GetMember(ctx, bid, uid)
	if err != nil {
		r.log.Logf("ERROR Failed to get member: %v", err)
		return nil, err
	}

	return member, nil
}

// BusinessInvitations retrieves all invitations for a business
func (r *Resolver) BusinessInvitations(ctx context.Context, businessID string) ([]domain.BusinessInvitation, error) {
	id, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	invitations, err := r.businessService.GetBusinessInvitations(ctx, id)
	if err != nil {
		r.log.Logf("ERROR Failed to get business invitations: %v", err)
		return nil, err
	}

	return invitations, nil
}

// MyInvitations retrieves all invitations for a user email
func (r *Resolver) MyInvitations(ctx context.Context, email string) ([]domain.BusinessInvitation, error) {
	invitations, err := r.businessService.GetUserInvitations(ctx, email)
	if err != nil {
		r.log.Logf("ERROR Failed to get user invitations: %v", err)
		return nil, err
	}

	return invitations, nil
}

// MyBusinessPermissions retrieves the authenticated user's permissions for a business
func (r *Resolver) MyBusinessPermissions(ctx context.Context, businessID string) (*domain.MemberPermissions, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to get permissions")
		return nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	permissions, err := r.businessService.GetUserPermissions(ctx, userID, bid)
	if err != nil {
		r.log.Logf("ERROR Failed to get user permissions: %v", err)
		return nil, err
	}

	return permissions, nil
}

// ===========================
// MUTATIONS
// ===========================

// CreateBusiness creates a new business
func (r *Resolver) CreateBusiness(ctx context.Context, input model.CreateBusinessInput) (*domain.Business, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to create business")
		return nil, fmt.Errorf("unauthenticated")
	}

	creatorID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	// Convert GraphQL input to domain input
	domainInput := convertCreateBusinessInput(input)

	business, err := r.businessService.CreateBusiness(ctx, domainInput, creatorID)
	if err != nil {
		r.log.Logf("ERROR Failed to create business: %v", err)
		return nil, err
	}

	r.log.Logf("INFO Business created: %s by user %s", business.ID, v.UserID)
	return business, nil
}

// UpdateBusiness updates an existing business
func (r *Resolver) UpdateBusiness(ctx context.Context, id string, input model.UpdateBusinessInput) (*domain.Business, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to update business")
		return nil, fmt.Errorf("unauthenticated")
	}

	updaterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	businessID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	// Convert GraphQL input to domain input
	domainInput := convertUpdateBusinessInput(input)

	business, err := r.businessService.UpdateBusiness(ctx, businessID, domainInput, updaterID)
	if err != nil {
		r.log.Logf("ERROR Failed to update business: %v", err)
		return nil, err
	}

	return business, nil
}

// DeleteBusiness deletes a business
func (r *Resolver) DeleteBusiness(ctx context.Context, id string) (bool, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to delete business")
		return false, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID")
	}

	businessID, err := uuid.Parse(id)
	if err != nil {
		return false, fmt.Errorf("invalid business ID")
	}

	if err := r.businessService.DeleteBusiness(ctx, businessID, userID); err != nil {
		r.log.Logf("ERROR Failed to delete business: %v", err)
		return false, err
	}

	return true, nil
}

// AddBusinessMember adds a member to a business
func (r *Resolver) AddBusinessMember(ctx context.Context, businessID, userID string, role domain.MemberRole, customPermissions *model.MemberPermissionsInput) (*domain.BusinessMember, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to add business member")
		return nil, fmt.Errorf("unauthenticated")
	}

	inviterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid target user ID")
	}

	// Convert custom permissions if provided
	var domainPerms *domain.MemberPermissions
	if customPermissions != nil {
		domainPerms = convertMemberPermissionsInput(customPermissions)
	}

	member, err := r.businessService.AddMember(ctx, bid, uid, inviterID, role, domainPerms)
	if err != nil {
		r.log.Logf("ERROR Failed to add member: %v", err)
		return nil, err
	}

	return member, nil
}

// UpdateMemberRole updates a member's role
func (r *Resolver) UpdateMemberRole(ctx context.Context, businessID, memberID string, role domain.MemberRole) (*domain.BusinessMember, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to update member role")
		return nil, fmt.Errorf("unauthenticated")
	}

	updaterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	mid, err := uuid.Parse(memberID)
	if err != nil {
		return nil, fmt.Errorf("invalid member ID")
	}

	member, err := r.businessService.UpdateMemberRole(ctx, bid, mid, role, updaterID)
	if err != nil {
		r.log.Logf("ERROR Failed to update member role: %v", err)
		return nil, err
	}

	return member, nil
}

// UpdateMemberPermissions updates a member's permissions
func (r *Resolver) UpdateMemberPermissions(ctx context.Context, businessID, memberID string, permissions model.MemberPermissionsInput) (*domain.BusinessMember, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to update member permissions")
		return nil, fmt.Errorf("unauthenticated")
	}

	updaterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	mid, err := uuid.Parse(memberID)
	if err != nil {
		return nil, fmt.Errorf("invalid member ID")
	}

	domainPerms := *convertMemberPermissionsInput(&permissions)

	member, err := r.businessService.UpdateMemberPermissions(ctx, bid, mid, domainPerms, updaterID)
	if err != nil {
		r.log.Logf("ERROR Failed to update member permissions: %v", err)
		return nil, err
	}

	return member, nil
}

// RemoveMember removes a member from a business
func (r *Resolver) RemoveMember(ctx context.Context, businessID, memberID string) (bool, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to remove member")
		return false, fmt.Errorf("unauthenticated")
	}

	removerID, err := uuid.Parse(v.UserID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID")
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return false, fmt.Errorf("invalid business ID")
	}

	mid, err := uuid.Parse(memberID)
	if err != nil {
		return false, fmt.Errorf("invalid member ID")
	}

	if err := r.businessService.RemoveMember(ctx, bid, mid, removerID); err != nil {
		r.log.Logf("ERROR Failed to remove member: %v", err)
		return false, err
	}

	return true, nil
}

// InviteMember creates an invitation for a user to join a business
func (r *Resolver) InviteMember(ctx context.Context, businessID string, input model.InviteMemberInput) (*domain.BusinessInvitation, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to invite member")
		return nil, fmt.Errorf("unauthenticated")
	}

	inviterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	bid, err := uuid.Parse(businessID)
	if err != nil {
		return nil, fmt.Errorf("invalid business ID")
	}

	var domainPerms *domain.MemberPermissions
	if input.CustomPermissions != nil {
		domainPerms = convertMemberPermissionsInput(input.CustomPermissions)
	}

	invitation, err := r.businessService.InviteUser(ctx, bid, input.Email, input.Role, inviterID, domainPerms)
	if err != nil {
		r.log.Logf("ERROR Failed to invite member: %v", err)
		return nil, err
	}

	return invitation, nil
}

// AcceptInvitation accepts an invitation and creates a membership
func (r *Resolver) AcceptInvitation(ctx context.Context, token string) (*domain.BusinessMember, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to accept invitation")
		return nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	member, err := r.businessService.AcceptInvitation(ctx, token, userID)
	if err != nil {
		r.log.Logf("ERROR Failed to accept invitation: %v", err)
		return nil, err
	}

	return member, nil
}

// DeclineInvitation declines an invitation
func (r *Resolver) DeclineInvitation(ctx context.Context, token string) (bool, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to decline invitation")
		return false, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID")
	}

	if err := r.businessService.DeclineInvitation(ctx, token, userID); err != nil {
		r.log.Logf("ERROR Failed to decline invitation: %v", err)
		return false, err
	}

	return true, nil
}

// RevokeInvitation revokes an invitation
func (r *Resolver) RevokeInvitation(ctx context.Context, invitationID string) (bool, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Logf("WARN Unauthenticated attempt to revoke invitation")
		return false, fmt.Errorf("unauthenticated")
	}

	revokerID, err := uuid.Parse(v.UserID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID")
	}

	iid, err := uuid.Parse(invitationID)
	if err != nil {
		return false, fmt.Errorf("invalid invitation ID")
	}

	if err := r.businessService.RevokeInvitation(ctx, iid, revokerID); err != nil {
		r.log.Logf("ERROR Failed to revoke invitation: %v", err)
		return false, err
	}

	return true, nil
}

// ===========================
// HELPER FUNCTIONS
// ===========================

func convertCreateBusinessInput(input model.CreateBusinessInput) domain.CreateBusinessInput {
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

func convertUpdateBusinessInput(input model.UpdateBusinessInput) domain.UpdateBusinessInput {
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

func convertMemberPermissionsInput(input *model.MemberPermissionsInput) *domain.MemberPermissions {
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
