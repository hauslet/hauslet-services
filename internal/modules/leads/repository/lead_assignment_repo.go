package repository

import (
	"context"
	"fmt"
	"hauslet/internal/modules/leads/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LeadAssignmentRepositoryImpl implements the LeadAssignmentRepository interface
type LeadAssignmentRepositoryImpl struct {
	db *gorm.DB
}

// NewLeadAssignmentRepository creates a new instance of LeadAssignment repository
func NewLeadAssignmentRepository(db *gorm.DB) LeadAssignmentRepository {
	return &LeadAssignmentRepositoryImpl{db: db}
}

// CreateLeadAssignment creates a new lead assignment record in the database
func (r *LeadAssignmentRepositoryImpl) CreateLeadAssignment(ctx context.Context, assignment *schema.LeadAssignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

// ListLeadAssignments retrieves all assignments for a specific lead, ordered by assigned_at descending
func (r *LeadAssignmentRepositoryImpl) ListLeadAssignments(ctx context.Context, leadID uuid.UUID) ([]*schema.LeadAssignment, error) {
	var assignments []*schema.LeadAssignment
	err := r.db.WithContext(ctx).
		Where("lead_id = ?", leadID).
		Order("assigned_at DESC").
		Find(&assignments).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list lead assignments: %w", err)
	}

	return assignments, nil
}

// GetLatestAssignment retrieves the most recent assignment for a specific lead
func (r *LeadAssignmentRepositoryImpl) GetLatestAssignment(ctx context.Context, leadID uuid.UUID) (*schema.LeadAssignment, error) {
	var assignment schema.LeadAssignment
	err := r.db.WithContext(ctx).
		Where("lead_id = ?", leadID).
		Order("assigned_at DESC").
		First(&assignment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if no assignment exists
		}
		return nil, fmt.Errorf("failed to get latest assignment: %w", err)
	}

	return &assignment, nil
}
