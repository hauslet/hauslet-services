package repository

import (
	"context"
	"hauslet/internal/modules/business/repository/schema"

	"github.com/google/uuid"
)

// CreateInvitation creates a new business invitation
func (r *BusinessRepositoryImpl) CreateInvitation(ctx context.Context, invitation *schema.BusinessInvitation) error {
	return r.db.WithContext(ctx).Create(invitation).Error
}

// GetInvitationByID retrieves an invitation by ID
func (r *BusinessRepositoryImpl) GetInvitationByID(ctx context.Context, id uuid.UUID) (*schema.BusinessInvitation, error) {
	var invitation schema.BusinessInvitation
	err := r.db.WithContext(ctx).
		Preload("Business").
		Where("id = ?", id).
		First(&invitation).Error
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}

// GetInvitationByToken retrieves an invitation by token
func (r *BusinessRepositoryImpl) GetInvitationByToken(ctx context.Context, token string) (*schema.BusinessInvitation, error) {
	var invitation schema.BusinessInvitation
	err := r.db.WithContext(ctx).
		Preload("Business").
		Where("token = ?", token).
		First(&invitation).Error
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}

// UpdateInvitation updates an existing invitation
func (r *BusinessRepositoryImpl) UpdateInvitation(ctx context.Context, invitation *schema.BusinessInvitation) error {
	return r.db.WithContext(ctx).Save(invitation).Error
}

// PatchInvitation updates specific fields of an invitation
func (r *BusinessRepositoryImpl) PatchInvitation(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&schema.BusinessInvitation{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// DeleteInvitation deletes an invitation
func (r *BusinessRepositoryImpl) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&schema.BusinessInvitation{}).Error
}

// ListBusinessInvitations lists all invitations for a business
func (r *BusinessRepositoryImpl) ListBusinessInvitations(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessInvitation, error) {
	var invitations []*schema.BusinessInvitation
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

// ListUserInvitations lists all invitations for a user email
func (r *BusinessRepositoryImpl) ListUserInvitations(ctx context.Context, email string) ([]*schema.BusinessInvitation, error) {
	var invitations []*schema.BusinessInvitation
	err := r.db.WithContext(ctx).
		Preload("Business").
		Where("email = ?", email).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

// ListPendingInvitations lists all pending invitations for a business
func (r *BusinessRepositoryImpl) ListPendingInvitations(ctx context.Context, businessID uuid.UUID) ([]*schema.BusinessInvitation, error) {
	var invitations []*schema.BusinessInvitation
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND status = ?", businessID, schema.InvitationPending).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

// HasPendingInvitation checks if a pending invitation exists for a business and email
func (r *BusinessRepositoryImpl) HasPendingInvitation(ctx context.Context, businessID uuid.UUID, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&schema.BusinessInvitation{}).
		Where("business_id = ? AND email = ? AND status = ?", businessID, email, schema.InvitationPending).
		Count(&count).Error
	return count > 0, err
}
