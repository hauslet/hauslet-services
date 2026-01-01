package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"hauslet/internal/modules/promotions/repository/schema"
)

// AgentSubscriptionRepo implements AgentSubscriptionRepository using GORM
type AgentSubscriptionRepo struct {
	db *gorm.DB
}

// NewAgentSubscriptionRepository creates a new agent subscription repository
func NewAgentSubscriptionRepository(db *gorm.DB) AgentSubscriptionRepository {
	return &AgentSubscriptionRepo{db: db}
}

// Create creates a new subscription
func (r *AgentSubscriptionRepo) Create(ctx context.Context, sub *schema.AgentSubscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

// GetByID retrieves a subscription by ID
func (r *AgentSubscriptionRepo) GetByID(ctx context.Context, id uuid.UUID) (*schema.AgentSubscription, error) {
	var sub schema.AgentSubscription
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&sub).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subscription by id: %w", err)
	}

	return &sub, nil
}

// GetActiveByUser retrieves the active subscription for a user
func (r *AgentSubscriptionRepo) GetActiveByUser(ctx context.Context, userID uuid.UUID) (*schema.AgentSubscription, error) {
	var sub schema.AgentSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND (status = ? OR status = ?) AND deleted_at IS NULL", userID, "active", "trial").
		First(&sub).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active subscription: %w", err)
	}

	return &sub, nil
}

// Update updates a subscription
func (r *AgentSubscriptionRepo) Update(ctx context.Context, sub *schema.AgentSubscription) error {
	sub.UpdatedAt = time.Now()
	err := r.db.WithContext(ctx).Save(sub).Error
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of a subscription
func (r *AgentSubscriptionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	err := r.db.WithContext(ctx).
		Model(&schema.AgentSubscription{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update subscription status: %w", err)
	}

	return nil
}

// ListDueForBilling lists subscriptions that are due for billing
func (r *AgentSubscriptionRepo) ListDueForBilling(ctx context.Context) ([]*schema.AgentSubscription, error) {
	now := time.Now()

	var subs []*schema.AgentSubscription
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_billing_date <= ? AND deleted_at IS NULL", "active", now).
		Find(&subs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions due for billing: %w", err)
	}

	return subs, nil
}

// ListByUser lists all subscriptions for a user (including inactive)
func (r *AgentSubscriptionRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*schema.AgentSubscription, error) {
	var subs []*schema.AgentSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Find(&subs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions by user: %w", err)
	}

	return subs, nil
}
