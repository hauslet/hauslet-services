package repository

import (
	"context"
	"hauslet/internal/modules/business/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BusinessRepository defines storage operations for businesses
type BusinessRepository interface {
	// Business CRUD
	CreateBusiness(ctx context.Context, business *schema.Business) error
	GetBusinessByID(ctx context.Context, id uuid.UUID) (*schema.Business, error)
	GetBusinessBySlug(ctx context.Context, slug string) (*schema.Business, error)
	UpdateBusiness(ctx context.Context, business *schema.Business) error
	PatchBusiness(ctx context.Context, id uuid.UUID, updates map[string]any) error
	DeleteBusiness(ctx context.Context, id uuid.UUID) error // Soft delete
	HardDeleteBusiness(ctx context.Context, id uuid.UUID) error
	BusinessExists(ctx context.Context, id uuid.UUID) (bool, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	ListBusinesses(ctx context.Context, limit, offset int) ([]*schema.Business, error)
	ListBusinessesByCreator(ctx context.Context, creatorID uuid.UUID) ([]*schema.Business, error)
	SearchBusinesses(ctx context.Context, query string, limit, offset int) ([]*schema.Business, error)

	// Business Members CRUD
	AddMember(ctx context.Context, member *schema.BusinessMember) error
	GetMemberByID(ctx context.Context, memberID uuid.UUID) (*schema.BusinessMember, error)
	GetMember(ctx context.Context, businessID, userID uuid.UUID) (*schema.BusinessMember, error)
	UpdateMember(ctx context.Context, member *schema.BusinessMember) error
	PatchMember(ctx context.Context, memberID uuid.UUID, updates map[string]any) error
	RemoveMember(ctx context.Context, businessID, userID uuid.UUID) error
	ListBusinessMembers(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessMember, error)
	ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*schema.BusinessMember, error)
	GetActiveMemberCount(ctx context.Context, businessID uuid.UUID) (int, error)
	IsMember(ctx context.Context, businessID, userID uuid.UUID) (bool, error)
	GetOwners(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessMember, error)

	// Business Invitations CRUD
	CreateInvitation(ctx context.Context, invitation *schema.BusinessInvitation) error
	GetInvitationByID(ctx context.Context, id uuid.UUID) (*schema.BusinessInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (*schema.BusinessInvitation, error)
	UpdateInvitation(ctx context.Context, invitation *schema.BusinessInvitation) error
	PatchInvitation(ctx context.Context, id uuid.UUID, updates map[string]any) error
	DeleteInvitation(ctx context.Context, id uuid.UUID) error
	ListBusinessInvitations(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessInvitation, error)
	ListUserInvitations(ctx context.Context, email string) ([]*schema.BusinessInvitation, error)
	ListPendingInvitations(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessInvitation, error)
	HasPendingInvitation(ctx context.Context, businessID uuid.UUID, email string) (bool, error)

	// Transaction support
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// BusinessRepositoryImpl implements BusinessRepository
type BusinessRepositoryImpl struct {
	db *gorm.DB
}

// NewBusinessRepository creates a new business repository
func NewBusinessRepository(db *gorm.DB) BusinessRepository {
	return &BusinessRepositoryImpl{db: db}
}
