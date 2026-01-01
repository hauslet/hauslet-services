package repository

import (
	"context"
	"fmt"
	"hauslet/internal/modules/leads/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LeadEventRepositoryImpl implements the LeadEventRepository interface
type LeadEventRepositoryImpl struct {
	db *gorm.DB
}

// NewLeadEventRepository creates a new instance of LeadEvent repository
func NewLeadEventRepository(db *gorm.DB) LeadEventRepository {
	return &LeadEventRepositoryImpl{db: db}
}

// CreateLeadEvent creates a new lead event in the database
func (r *LeadEventRepositoryImpl) CreateLeadEvent(ctx context.Context, event *schema.LeadEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// ListLeadEvents retrieves events for a specific lead, ordered by creation date descending
func (r *LeadEventRepositoryImpl) ListLeadEvents(ctx context.Context, leadID uuid.UUID, limit int) ([]*schema.LeadEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	var events []*schema.LeadEvent
	err := r.db.WithContext(ctx).
		Where("lead_id = ?", leadID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list lead events: %w", err)
	}

	return events, nil
}
