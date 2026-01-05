package service

import (
	"context"
	"time"

	"hauslet/internal/modules/business/domain"

	"github.com/google/uuid"
)

// BusinessService defines business-level operations for business management
type BusinessService interface {
	// Business CRUD
	CreateBusiness(ctx context.Context, input domain.CreateBusinessInput, creatorID uuid.UUID) (*domain.Business, error)
	GetBusiness(ctx context.Context, id uuid.UUID) (*domain.Business, error)
	GetBusinessBySlug(ctx context.Context, slug string) (*domain.Business, error)
	UpdateBusiness(ctx context.Context, id uuid.UUID, input domain.UpdateBusinessInput, updatedBy uuid.UUID) (*domain.Business, error)
	DeleteBusiness(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	ListUserBusinesses(ctx context.Context, userID uuid.UUID) ([]domain.Business, error)
	ListAllBusinesses(ctx context.Context, limit, offset int) ([]domain.Business, error)
	SearchBusinesses(ctx context.Context, query string, limit, offset int) ([]domain.Business, error)

	// Membership Management
	AddMember(ctx context.Context, businessID, userID, invitedBy uuid.UUID, role domain.MemberRole, customPermissions *domain.MemberPermissions) (*domain.BusinessMember, error)
	UpdateMemberRole(ctx context.Context, businessID, memberID uuid.UUID, newRole domain.MemberRole, updatedBy uuid.UUID) (*domain.BusinessMember, error)
	UpdateMemberPermissions(ctx context.Context, businessID, memberID uuid.UUID, permissions domain.MemberPermissions, updatedBy uuid.UUID) (*domain.BusinessMember, error)
	RemoveMember(ctx context.Context, businessID, memberID, removedBy uuid.UUID) error
	GetBusinessMembers(ctx context.Context, businessID uuid.UUID) ([]domain.BusinessMember, error)
	GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]domain.BusinessMember, error)
	GetMember(ctx context.Context, businessID, userID uuid.UUID) (*domain.BusinessMember, error)

	// Invitations
	InviteUser(ctx context.Context, businessID uuid.UUID, email string, role domain.MemberRole, invitedBy uuid.UUID, customPermissions *domain.MemberPermissions) (*domain.BusinessInvitation, error)
	AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*domain.BusinessMember, error)
	DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error
	RevokeInvitation(ctx context.Context, invitationID, revokedBy uuid.UUID) error
	GetBusinessInvitations(ctx context.Context, businessID uuid.UUID) ([]domain.BusinessInvitation, error)
	GetUserInvitations(ctx context.Context, email string) ([]domain.BusinessInvitation, error)

	// Permissions
	GetUserPermissions(ctx context.Context, userID, businessID uuid.UUID) (*domain.MemberPermissions, error)
	HasPermission(ctx context.Context, userID, businessID uuid.UUID, permission string) (bool, error)
	IsOwner(ctx context.Context, userID, businessID uuid.UUID) (bool, error)
	IsAdmin(ctx context.Context, userID, businessID uuid.UUID) (bool, error)

	// Verification
	SetVerificationStatus(ctx context.Context, businessID uuid.UUID, verified bool, verifiedAt *time.Time) error
}
