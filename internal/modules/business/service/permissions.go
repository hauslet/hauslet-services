package service

import (
	"context"
	"hauslet/internal/modules/business/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetUserPermissions gets the permissions for a user in a business
func (s *BusinessServiceImpl) GetUserPermissions(ctx context.Context, userID, businessID uuid.UUID) (*domain.MemberPermissions, error) {
	member, err := s.repo.GetMember(ctx, businessID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrMemberNotFound
		}
		return nil, err
	}

	domainMember := domain.MapBusinessMemberFromSchema(member)
	return &domainMember.Permissions, nil
}

// HasPermission checks if a user has a specific permission in a business
func (s *BusinessServiceImpl) HasPermission(ctx context.Context, userID, businessID uuid.UUID, permission string) (bool, error) {
	member, err := s.repo.GetMember(ctx, businessID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	domainMember := domain.MapBusinessMemberFromSchema(member)
	return domainMember.HasPermission(permission), nil
}

// IsOwner checks if a user is an owner of a business
func (s *BusinessServiceImpl) IsOwner(ctx context.Context, userID, businessID uuid.UUID) (bool, error) {
	member, err := s.repo.GetMember(ctx, businessID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	domainMember := domain.MapBusinessMemberFromSchema(member)
	return domainMember.IsOwner(), nil
}

// IsAdmin checks if a user is an admin or owner of a business
func (s *BusinessServiceImpl) IsAdmin(ctx context.Context, userID, businessID uuid.UUID) (bool, error) {
	member, err := s.repo.GetMember(ctx, businessID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	domainMember := domain.MapBusinessMemberFromSchema(member)
	return domainMember.IsAdmin(), nil
}
