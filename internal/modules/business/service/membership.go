package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/business/domain"
	"hauslet/internal/modules/business/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AddMember adds a new member to a business
func (s *BusinessServiceImpl) AddMember(ctx context.Context, businessID, userID, invitedBy uuid.UUID, role domain.MemberRole, customPermissions *domain.MemberPermissions) (*domain.BusinessMember, error) {
	// Check if the inviter has permission to add members
	hasPermission, err := s.HasPermission(ctx, invitedBy, businessID, "CanManageMembers")
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, domain.ErrInsufficientPermissions
	}

	// Check if user is already a member
	isMember, err := s.repo.IsMember(ctx, businessID, userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if isMember {
		return nil, domain.ErrMemberAlreadyExists
	}

	// Determine permissions
	permissions := domain.GetDefaultPermissions(role)
	if customPermissions != nil {
		permissions = *customPermissions
	}

	// Create member
	now := time.Now()
	member := &domain.BusinessMember{
		ID:          uuid.New(),
		BusinessID:  businessID,
		UserID:      userID,
		Role:        role,
		Permissions: permissions,
		IsActive:    true,
		InvitedBy:   &invitedBy,
		JoinedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	schemaMember, err := domain.MapBusinessMemberToSchema(member)
	if err != nil {
		return nil, fmt.Errorf("failed to map member: %w", err)
	}

	if err := s.repo.AddMember(ctx, schemaMember); err != nil {
		s.log.Error("Failed to add member to business", "business_id", businessID, "error", err)
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	s.log.Info(" Member added to business", "business_id", businessID, "user_id", userID, "role", role)
	return member, nil
}

// UpdateMemberRole updates a member's role
func (s *BusinessServiceImpl) UpdateMemberRole(ctx context.Context, businessID, memberID uuid.UUID, newRole domain.MemberRole, updatedBy uuid.UUID) (*domain.BusinessMember, error) {
	// Get the member to update
	schemaMember, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrMemberNotFound
		}
		return nil, err
	}

	// Verify member belongs to the business
	if schemaMember.BusinessID != businessID {
		return nil, domain.ErrMemberNotFound
	}

	// Get the updater's membership
	updaterMember, err := s.repo.GetMember(ctx, businessID, updatedBy)
	if err != nil {
		return nil, domain.ErrInsufficientPermissions
	}

	updaterDomain := domain.MapBusinessMemberFromSchema(updaterMember)
	memberDomain := domain.MapBusinessMemberFromSchema(schemaMember)

	// Check if updater can manage this member
	if !updaterDomain.CanManageOtherMember(memberDomain) {
		return nil, domain.ErrInsufficientPermissions
	}

	// Prevent changing own role
	if schemaMember.UserID == updatedBy {
		return nil, domain.ErrCannotModifyOwnRole
	}

	// Update role and permissions
	schemaMember.Role = schema.MemberRole(newRole)
	schemaMember.Permissions = domain.MapBusinessMemberToPermissionsMap(domain.GetDefaultPermissions(newRole))
	schemaMember.UpdatedAt = time.Now()

	if err := s.repo.UpdateMember(ctx, schemaMember); err != nil {
		s.log.Error("Failed to update member role", "error", err)
		return nil, fmt.Errorf("failed to update member role: %w", err)
	}

	s.log.Info(" Member role updated", "member_id", memberID, "new_role", newRole, "business_id", businessID, "updated_by", updatedBy)
	return domain.MapBusinessMemberFromSchema(schemaMember), nil
}

// UpdateMemberPermissions updates a member's custom permissions
func (s *BusinessServiceImpl) UpdateMemberPermissions(ctx context.Context, businessID, memberID uuid.UUID, permissions domain.MemberPermissions, updatedBy uuid.UUID) (*domain.BusinessMember, error) {
	// Get the member to update
	schemaMember, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrMemberNotFound
		}
		return nil, err
	}

	// Verify member belongs to the business
	if schemaMember.BusinessID != businessID {
		return nil, domain.ErrMemberNotFound
	}

	// Check if updater has permission
	hasPermission, err := s.HasPermission(ctx, updatedBy, businessID, "CanManageMembers")
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, domain.ErrInsufficientPermissions
	}

	// Prevent changing own permissions
	if schemaMember.UserID == updatedBy {
		return nil, domain.ErrCannotModifyOwnRole
	}

	// Update permissions
	schemaMember.Permissions = domain.MapBusinessMemberToPermissionsMap(permissions)
	schemaMember.UpdatedAt = time.Now()

	if err := s.repo.UpdateMember(ctx, schemaMember); err != nil {
		s.log.Error("Failed to update member permissions", "error", err)
		return nil, fmt.Errorf("failed to update member permissions: %w", err)
	}

	s.log.Info(" Member permissions updated", "member_id", memberID, "business_id", businessID, "updated_by", updatedBy)
	return domain.MapBusinessMemberFromSchema(schemaMember), nil
}

// RemoveMember removes a member from a business
func (s *BusinessServiceImpl) RemoveMember(ctx context.Context, businessID, memberID, removedBy uuid.UUID) error {
	// Get the member to remove
	schemaMember, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.ErrMemberNotFound
		}
		return err
	}

	// Verify member belongs to the business
	if schemaMember.BusinessID != businessID {
		return domain.ErrMemberNotFound
	}

	memberDomain := domain.MapBusinessMemberFromSchema(schemaMember)

	// Cannot remove owners
	if memberDomain.IsOwner() {
		// Check if there are other owners
		owners, err := s.repo.GetOwners(ctx, businessID)
		if err != nil {
			return err
		}
		if len(owners) <= 1 {
			return domain.ErrMustHaveOneOwner
		}
	}

	// Get the remover's membership
	removerMember, err := s.repo.GetMember(ctx, businessID, removedBy)
	if err != nil {
		return domain.ErrInsufficientPermissions
	}

	removerDomain := domain.MapBusinessMemberFromSchema(removerMember)

	// Check if remover can manage this member
	if !removerDomain.CanManageOtherMember(memberDomain) {
		return domain.ErrInsufficientPermissions
	}

	// Remove member
	if err := s.repo.RemoveMember(ctx, businessID, schemaMember.UserID); err != nil {
		s.log.Error("Failed to remove member", "error", err)
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.log.Info(" Member removed", "member_id", memberID, "business_id", businessID, "removed_by", removedBy)
	return nil
}

// GetBusinessMembers retrieves all members of a business
func (s *BusinessServiceImpl) GetBusinessMembers(ctx context.Context, businessID uuid.UUID) ([]domain.BusinessMember, error) {
	schemaMembers, err := s.repo.ListBusinessMembers(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to get business members: %w", err)
	}

	return domain.MapBusinessMembersFromSchema(schemaMembers), nil
}

// GetUserMemberships retrieves all memberships for a user
func (s *BusinessServiceImpl) GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]domain.BusinessMember, error) {
	schemaMembers, err := s.repo.ListUserMemberships(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user memberships: %w", err)
	}

	return domain.MapBusinessMembersFromSchema(schemaMembers), nil
}

// GetMember retrieves a specific member
func (s *BusinessServiceImpl) GetMember(ctx context.Context, businessID, userID uuid.UUID) (*domain.BusinessMember, error) {
	schemaMember, err := s.repo.GetMember(ctx, businessID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrMemberNotFound
		}
		return nil, err
	}

	return domain.MapBusinessMemberFromSchema(schemaMember), nil
}
