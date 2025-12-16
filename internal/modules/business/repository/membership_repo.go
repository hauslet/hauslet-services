package repository

import (
	"context"
	"hauslet/internal/modules/business/repository/schema"

	"github.com/google/uuid"
)

// AddMember adds a new member to a business
func (r *BusinessRepositoryImpl) AddMember(ctx context.Context, member *schema.BusinessMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

// GetMemberByID retrieves a member by their member ID
func (r *BusinessRepositoryImpl) GetMemberByID(ctx context.Context, memberID uuid.UUID) (*schema.BusinessMember, error) {
	var member schema.BusinessMember
	err := r.db.WithContext(ctx).Where("id = ?", memberID).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// GetMember retrieves a member by business ID and user ID
func (r *BusinessRepositoryImpl) GetMember(ctx context.Context, businessID, userID uuid.UUID) (*schema.BusinessMember, error) {
	var member schema.BusinessMember
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND user_id = ?", businessID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// UpdateMember updates an existing member
func (r *BusinessRepositoryImpl) UpdateMember(ctx context.Context, member *schema.BusinessMember) error {
	return r.db.WithContext(ctx).Save(member).Error
}

// RemoveMember removes a member from a business
func (r *BusinessRepositoryImpl) RemoveMember(ctx context.Context, businessID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("business_id = ? AND user_id = ?", businessID, userID).
		Delete(&schema.BusinessMember{}).Error
}

// ListBusinessMembers lists all members of a business
func (r *BusinessRepositoryImpl) ListBusinessMembers(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessMember, error) {
	var members []*schema.BusinessMember
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Order("joined_at ASC").
		Find(&members).Error
	return members, err
}

// ListUserMemberships lists all businesses a user is a member of
func (r *BusinessRepositoryImpl) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*schema.BusinessMember, error) {
	var members []*schema.BusinessMember
	err := r.db.WithContext(ctx).
		Preload("Business").
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("joined_at DESC").
		Find(&members).Error
	return members, err
}

// IsMember checks if a user is a member of a business
func (r *BusinessRepositoryImpl) IsMember(ctx context.Context, businessID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&schema.BusinessMember{}).
		Where("business_id = ? AND user_id = ? AND is_active = ?", businessID, userID, true).
		Count(&count).Error
	return count > 0, err
}

// GetOwners retrieves all owners of a business
func (r *BusinessRepositoryImpl) GetOwners(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessMember, error) {
	var owners []*schema.BusinessMember
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND role = ? AND is_active = ?", businessID, schema.RoleOwner, true).
		Find(&owners).Error
	return owners, err
}
