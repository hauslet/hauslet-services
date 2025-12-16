package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/business/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateBusiness creates a new business
func (s *BusinessServiceImpl) CreateBusiness(ctx context.Context, input domain.CreateBusinessInput, creatorID uuid.UUID) (*domain.Business, error) {
	// Validate input
	if input.Name == "" {
		return nil, fmt.Errorf("business name is required")
	}
	if input.DisplayName == "" {
		return nil, fmt.Errorf("display name is required")
	}
	if input.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Generate slug from name
	slug := generateSlug(input.Name)

	// Check if slug already exists
	exists, err := s.repo.SlugExists(ctx, slug)
	if err != nil {
		s.log.Logf("ERROR Failed to check slug existence: %v", err)
		return nil, fmt.Errorf("failed to check slug: %w", err)
	}
	if exists {
		// Append random suffix if slug exists
		slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])
	}

	// Create business domain model
	now := time.Now()
	business := &domain.Business{
		ID:                 uuid.New(),
		Slug:               slug,
		Name:               input.Name,
		DisplayName:        input.DisplayName,
		Description:        input.Description,
		BusinessType:       input.BusinessType,
		RegistrationNumber: input.RegistrationNumber,
		TaxID:              input.TaxID,
		LegalEntityType:    input.LegalEntityType,
		Email:              input.Email,
		PhoneNumbers:       input.PhoneNumbers,
		Website:            input.Website,
		Address:            input.Address,
		Location:           input.Location,
		LogoURL:            input.LogoURL,
		CoverImageURL:      input.CoverImageURL,
		BrandColor:         input.BrandColor,
		IsVerified:         false,
		IsActive:           true,
		BillingEmail:       input.BillingEmail,
		CreatedBy:          creatorID,
		MemberCount:        1, // Creator is the first member
		PropertyCount:      0,
		ListingCount:       0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Convert to schema
	schemaBusiness, err := domain.MapBusinessToSchema(business)
	if err != nil {
		s.log.Logf("ERROR Failed to map business to schema: %v", err)
		return nil, fmt.Errorf("failed to map business: %w", err)
	}

	// Create business and add creator as owner in a transaction
	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Create business
		if err := s.repo.CreateBusiness(ctx, schemaBusiness); err != nil {
			return fmt.Errorf("failed to create business: %w", err)
		}

		// Add creator as owner
		ownerMember := &domain.BusinessMember{
			ID:          uuid.New(),
			BusinessID:  business.ID,
			UserID:      creatorID,
			Role:        domain.RoleOwner,
			Permissions: domain.GetDefaultPermissions(domain.RoleOwner),
			IsActive:    true,
			JoinedAt:    now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		schemaMember, err := domain.MapBusinessMemberToSchema(ownerMember)
		if err != nil {
			return fmt.Errorf("failed to map member: %w", err)
		}

		if err := s.repo.AddMember(ctx, schemaMember); err != nil {
			return fmt.Errorf("failed to add owner member: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.Logf("ERROR Failed to create business: %v", err)
		return nil, err
	}

	s.log.Logf("INFO Business created: %s (ID: %s) by user %s", business.Name, business.ID, creatorID)

	// Fire-and-forget welcome email for the creator/business contact.
	if s.notifier != nil && business.Email != "" {
		creatorName := s.getProfileName(ctx, creatorID)
		if creatorName == "" {
			creatorName = business.DisplayName
		}
		if creatorName == "" {
			creatorName = business.Name
		}
		if err := s.notifier.SendBusinessCreatedEmail(ctx, business, creatorName, business.Email); err != nil {
			s.log.Logf("WARN Failed to send business created email for %s: %v", business.ID, err)
		}
	}

	return business, nil
}

// GetBusiness retrieves a business by ID
func (s *BusinessServiceImpl) GetBusiness(ctx context.Context, id uuid.UUID) (*domain.Business, error) {
	schemaBusiness, err := s.repo.GetBusinessByID(ctx, id)
	if err != nil {
		return nil, domain.ErrBusinessNotFound
	}

	return domain.MapBusinessFromSchema(schemaBusiness), nil
}

// GetBusinessBySlug retrieves a business by slug
func (s *BusinessServiceImpl) GetBusinessBySlug(ctx context.Context, slug string) (*domain.Business, error) {
	schemaBusiness, err := s.repo.GetBusinessBySlug(ctx, slug)
	if err != nil {
		return nil, domain.ErrBusinessNotFound
	}

	return domain.MapBusinessFromSchema(schemaBusiness), nil
}

// UpdateBusiness updates an existing business
func (s *BusinessServiceImpl) UpdateBusiness(ctx context.Context, id uuid.UUID, input domain.UpdateBusinessInput, updatedBy uuid.UUID) (*domain.Business, error) {
	// Get existing business
	business, err := s.GetBusiness(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if user has permission to update
	hasPermission, err := s.HasPermission(ctx, updatedBy, id, "CanEditBusiness")
	if err != nil {
		return nil, err
	}
	if !hasPermission {
		return nil, domain.ErrInsufficientPermissions
	}

	// Update fields
	if input.DisplayName != nil {
		business.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		business.Description = input.Description
	}
	if input.Email != nil {
		business.Email = *input.Email
	}
	if input.PhoneNumbers != nil {
		business.PhoneNumbers = input.PhoneNumbers
	}
	if input.Website != nil {
		business.Website = input.Website
	}
	if input.Address != nil {
		business.Address = *input.Address
	}
	if input.Location != nil {
		business.Location = input.Location
	}
	if input.LogoURL != nil {
		business.LogoURL = input.LogoURL
	}
	if input.CoverImageURL != nil {
		business.CoverImageURL = input.CoverImageURL
	}
	if input.BrandColor != nil {
		business.BrandColor = input.BrandColor
	}
	if input.BillingEmail != nil {
		business.BillingEmail = input.BillingEmail
	}

	business.UpdatedAt = time.Now()

	// Convert to schema and update
	schemaBusiness, err := domain.MapBusinessToSchema(business)
	if err != nil {
		return nil, fmt.Errorf("failed to map business: %w", err)
	}

	if err := s.repo.UpdateBusiness(ctx, schemaBusiness); err != nil {
		s.log.Logf("ERROR Failed to update business %s: %v", id, err)
		return nil, fmt.Errorf("failed to update business: %w", err)
	}

	s.log.Logf("INFO Business updated: %s by user %s", id, updatedBy)
	return business, nil
}

// DeleteBusiness deletes a business (soft delete)
func (s *BusinessServiceImpl) DeleteBusiness(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// Get business
	business, err := s.GetBusiness(ctx, id)
	if err != nil {
		return err
	}

	// Check if user is owner
	isOwner, err := s.IsOwner(ctx, deletedBy, id)
	if err != nil {
		return err
	}
	if !isOwner {
		return domain.ErrInsufficientPermissions
	}

	// Check if business can be deleted
	if !business.CanBeDeleted() {
		return fmt.Errorf("cannot delete business with existing properties or listings")
	}

	// Delete business
	if err := s.repo.DeleteBusiness(ctx, id); err != nil {
		s.log.Logf("ERROR Failed to delete business %s: %v", id, err)
		return fmt.Errorf("failed to delete business: %w", err)
	}

	s.log.Logf("INFO Business deleted: %s by user %s", id, deletedBy)
	return nil
}

// ListUserBusinesses lists all businesses a user is a member of
func (s *BusinessServiceImpl) ListUserBusinesses(ctx context.Context, userID uuid.UUID) ([]domain.Business, error) {
	// Get user memberships
	schemaMembers, err := s.repo.ListUserMemberships(ctx, userID)
	if err != nil {
		s.log.Logf("ERROR Failed to list user memberships: %v", err)
		return nil, fmt.Errorf("failed to list memberships: %w", err)
	}

	// Extract businesses from memberships
	businesses := make([]domain.Business, 0, len(schemaMembers))
	for _, member := range schemaMembers {
		business := domain.MapBusinessFromSchema(&member.Business)
		if business != nil {
			businesses = append(businesses, *business)
		}
	}

	return businesses, nil
}

// ListAllBusinesses lists all businesses with pagination
func (s *BusinessServiceImpl) ListAllBusinesses(ctx context.Context, limit, offset int) ([]domain.Business, error) {
	schemaBusinesses, err := s.repo.ListBusinesses(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list businesses: %w", err)
	}

	return domain.MapBusinessesFromSchema(schemaBusinesses), nil
}

// SearchBusinesses searches businesses by name
func (s *BusinessServiceImpl) SearchBusinesses(ctx context.Context, query string, limit, offset int) ([]domain.Business, error) {
	schemaBusinesses, err := s.repo.SearchBusinesses(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search businesses: %w", err)
	}

	return domain.MapBusinessesFromSchema(schemaBusinesses), nil
}

// generateSlug generates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	var result strings.Builder
	for _, char := range slug {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result.WriteRune(char)
		}
	}
	return result.String()
}
