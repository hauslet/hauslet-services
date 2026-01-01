package repository

import (
	"context"
	"fmt"
	"hauslet/internal/modules/leads/repository/schema"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LeadRepositoryImpl implements the LeadRepository interface
type LeadRepositoryImpl struct {
	db *gorm.DB
}

// NewLeadRepository creates a new instance of Lead repository
func NewLeadRepository(db *gorm.DB) LeadRepository {
	return &LeadRepositoryImpl{db: db}
}

// CreateLead creates a new lead in the database
func (r *LeadRepositoryImpl) CreateLead(ctx context.Context, lead *schema.Lead) error {
	return r.db.WithContext(ctx).Create(lead).Error
}

// GetLeadByID retrieves a lead by its ID
func (r *LeadRepositoryImpl) GetLeadByID(ctx context.Context, id uuid.UUID) (*schema.Lead, error) {
	var lead schema.Lead
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&lead).Error
	if err != nil {
		return nil, err
	}
	return &lead, nil
}

// UpdateLead updates an existing lead
func (r *LeadRepositoryImpl) UpdateLead(ctx context.Context, lead *schema.Lead) error {
	return r.db.WithContext(ctx).Save(lead).Error
}

// DeleteLead soft deletes a lead
func (r *LeadRepositoryImpl) DeleteLead(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.Lead{}).Error
}

// ListLeadsByListing retrieves leads for a specific listing with filters and pagination
func (r *LeadRepositoryImpl) ListLeadsByListing(ctx context.Context, listingID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error) {
	page = ValidatePagination(page)

	query := r.db.WithContext(ctx).Model(&schema.Lead{}).Where("listing_id = ?", listingID)
	query = r.applyFilters(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count leads: %w", err)
	}

	var leads []*schema.Lead
	err := query.
		Order("created_at DESC").
		Limit(page.Limit).
		Offset(page.Offset).
		Find(&leads).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leads: %w", err)
	}

	return leads, total, nil
}

// ListLeadsByBusiness retrieves leads for a specific business with filters and pagination
func (r *LeadRepositoryImpl) ListLeadsByBusiness(ctx context.Context, businessID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error) {
	page = ValidatePagination(page)

	query := r.db.WithContext(ctx).Model(&schema.Lead{}).Where("business_id = ?", businessID)
	query = r.applyFilters(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count leads: %w", err)
	}

	var leads []*schema.Lead
	err := query.
		Order("created_at DESC").
		Limit(page.Limit).
		Offset(page.Offset).
		Find(&leads).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leads: %w", err)
	}

	return leads, total, nil
}

// ListLeadsByOwner retrieves leads for listings owned by a user
func (r *LeadRepositoryImpl) ListLeadsByOwner(ctx context.Context, ownerID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error) {
	page = ValidatePagination(page)

	// Join with listings table to find leads for owner's listings
	query := r.db.WithContext(ctx).Model(&schema.Lead{}).
		Joins("JOIN listings ON listings.id = leads.listing_id").
		Where("listings.owner_id = ?", ownerID)

	query = r.applyFilters(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count leads: %w", err)
	}

	var leads []*schema.Lead
	err := query.
		Order("leads.created_at DESC").
		Limit(page.Limit).
		Offset(page.Offset).
		Find(&leads).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leads: %w", err)
	}

	return leads, total, nil
}

// ListLeadsByAssignee retrieves leads assigned to a specific user
func (r *LeadRepositoryImpl) ListLeadsByAssignee(ctx context.Context, userID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error) {
	page = ValidatePagination(page)

	query := r.db.WithContext(ctx).Model(&schema.Lead{}).Where("assigned_to = ?", userID)
	query = r.applyFilters(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count leads: %w", err)
	}

	var leads []*schema.Lead
	err := query.
		Order("created_at DESC").
		Limit(page.Limit).
		Offset(page.Offset).
		Find(&leads).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list leads: %w", err)
	}

	return leads, total, nil
}

// CountLeadsByEmail counts leads from an email address since a specific time
func (r *LeadRepositoryImpl) CountLeadsByEmail(ctx context.Context, email string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&schema.Lead{}).
		Where("email = ? AND created_at > ?", email, since).
		Count(&count).Error

	return count, err
}

// CountLeadsByIP counts leads from an IP address since a specific time
func (r *LeadRepositoryImpl) CountLeadsByIP(ctx context.Context, ipAddress string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&schema.Lead{}).
		Where("ip_address = ? AND created_at > ?", ipAddress, since).
		Count(&count).Error

	return count, err
}

// CountLeadsByListing counts leads for a specific listing from an email since a specific time
func (r *LeadRepositoryImpl) CountLeadsByListing(ctx context.Context, listingID uuid.UUID, email string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&schema.Lead{}).
		Where("listing_id = ? AND email = ? AND created_at > ?", listingID, email, since).
		Count(&count).Error

	return count, err
}

// GetLeadByEmailAndListing retrieves a lead by email and listing within a time window
func (r *LeadRepositoryImpl) GetLeadByEmailAndListing(ctx context.Context, email string, listingID uuid.UUID, since time.Time) (*schema.Lead, error) {
	var lead schema.Lead
	err := r.db.WithContext(ctx).
		Where("email = ? AND listing_id = ? AND created_at > ?", email, listingID, since).
		Order("created_at DESC").
		First(&lead).Error

	if err != nil {
		return nil, err
	}
	return &lead, nil
}

// Transaction executes a function within a database transaction
func (r *LeadRepositoryImpl) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// applyFilters applies filter criteria to a query
func (r *LeadRepositoryImpl) applyFilters(query *gorm.DB, filter LeadFilter) *gorm.DB {
	// Filter by status
	if len(filter.Status) > 0 {
		query = query.Where("status IN ?", filter.Status)
	}

	// Filter by source
	if len(filter.Source) > 0 {
		query = query.Where("source IN ?", filter.Source)
	}

	// Filter by spam status
	if filter.IsSpam != nil {
		query = query.Where("is_spam = ?", *filter.IsSpam)
	}

	// Filter by assignment status
	if filter.Assigned != nil {
		if *filter.Assigned {
			query = query.Where("assigned_to IS NOT NULL")
		} else {
			query = query.Where("assigned_to IS NULL")
		}
	}

	// Filter by date range
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", *filter.DateTo)
	}

	// Search in name, email, or message
	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		searchPattern := "%" + strings.ToLower(*filter.SearchTerm) + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(message) LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	return query
}
