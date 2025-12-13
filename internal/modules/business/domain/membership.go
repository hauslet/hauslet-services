package domain

import (
	"time"

	"github.com/google/uuid"
)

// MemberPermissions represents granular permissions for a business member
type MemberPermissions struct {
	CanCreateListings  bool
	CanEditListings    bool
	CanDeleteListings  bool
	CanPublishListings bool
	CanManageMedia     bool
	CanViewAnalytics   bool
	CanManageMembers   bool
	CanEditBusiness    bool
	CanViewFinancials  bool
}

// BusinessMember represents a member of a business
type BusinessMember struct {
	ID         uuid.UUID
	BusinessID uuid.UUID
	UserID     uuid.UUID

	Role MemberRole

	// Permissions (customizable per member)
	Permissions MemberPermissions

	// Status
	IsActive  bool
	InvitedBy *uuid.UUID
	JoinedAt  time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// HasPermission checks if the member has a specific permission
func (m *BusinessMember) HasPermission(permission string) bool {
	if !m.IsActive {
		return false
	}

	switch permission {
	case "CanCreateListings":
		return m.Permissions.CanCreateListings
	case "CanEditListings":
		return m.Permissions.CanEditListings
	case "CanDeleteListings":
		return m.Permissions.CanDeleteListings
	case "CanPublishListings":
		return m.Permissions.CanPublishListings
	case "CanManageMedia":
		return m.Permissions.CanManageMedia
	case "CanViewAnalytics":
		return m.Permissions.CanViewAnalytics
	case "CanManageMembers":
		return m.Permissions.CanManageMembers
	case "CanEditBusiness":
		return m.Permissions.CanEditBusiness
	case "CanViewFinancials":
		return m.Permissions.CanViewFinancials
	default:
		return false
	}
}

// IsOwner checks if the member is an owner
func (m *BusinessMember) IsOwner() bool {
	return m.Role == RoleOwner
}

// IsAdmin checks if the member is an admin or owner
func (m *BusinessMember) IsAdmin() bool {
	return m.Role == RoleOwner || m.Role == RoleAdmin
}

// CanManageOtherMember checks if this member can manage another member
func (m *BusinessMember) CanManageOtherMember(otherMember *BusinessMember) bool {
	if !m.IsActive || !m.Permissions.CanManageMembers {
		return false
	}

	// Owners can manage anyone except other owners
	if m.IsOwner() {
		return !otherMember.IsOwner()
	}

	// Admins can manage members and viewers
	if m.Role == RoleAdmin {
		return otherMember.Role == RoleMember || otherMember.Role == RoleViewer || otherMember.Role == RoleAccountant
	}

	return false
}

// GetDefaultPermissions returns default permissions for a given role
func GetDefaultPermissions(role MemberRole) MemberPermissions {
	switch role {
	case RoleOwner:
		return MemberPermissions{
			CanCreateListings:  true,
			CanEditListings:    true,
			CanDeleteListings:  true,
			CanPublishListings: true,
			CanManageMedia:     true,
			CanViewAnalytics:   true,
			CanManageMembers:   true,
			CanEditBusiness:    true,
			CanViewFinancials:  true,
		}
	case RoleAdmin:
		return MemberPermissions{
			CanCreateListings:  true,
			CanEditListings:    true,
			CanDeleteListings:  true,
			CanPublishListings: true,
			CanManageMedia:     true,
			CanViewAnalytics:   true,
			CanManageMembers:   true,
			CanEditBusiness:    false,
			CanViewFinancials:  true,
		}
	case RoleMember:
		return MemberPermissions{
			CanCreateListings:  true,
			CanEditListings:    true,
			CanDeleteListings:  false,
			CanPublishListings: false,
			CanManageMedia:     true,
			CanViewAnalytics:   false,
			CanManageMembers:   false,
			CanEditBusiness:    false,
			CanViewFinancials:  false,
		}
	case RoleViewer:
		return MemberPermissions{
			CanCreateListings:  false,
			CanEditListings:    false,
			CanDeleteListings:  false,
			CanPublishListings: false,
			CanManageMedia:     false,
			CanViewAnalytics:   true,
			CanManageMembers:   false,
			CanEditBusiness:    false,
			CanViewFinancials:  false,
		}
	case RoleAccountant:
		return MemberPermissions{
			CanCreateListings:  false,
			CanEditListings:    false,
			CanDeleteListings:  false,
			CanPublishListings: false,
			CanManageMedia:     false,
			CanViewAnalytics:   true,
			CanManageMembers:   false,
			CanEditBusiness:    false,
			CanViewFinancials:  true,
		}
	default:
		return MemberPermissions{}
	}
}
