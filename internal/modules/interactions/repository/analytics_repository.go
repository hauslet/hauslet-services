package repository

import (
	"context"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/modules/interactions/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AnalyticsRepository defines the interface for analytics/aggregate data access
type AnalyticsRepository interface {
	// UpsertAggregate creates or updates an aggregate
	UpsertAggregate(ctx context.Context, aggregate *schema.InteractionAggregate) error

	// GetAggregates retrieves aggregates for an entity within a time range
	GetAggregates(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID, periodType domain.PeriodType, start, end time.Time) ([]*schema.InteractionAggregate, error)

	// GetAggregatePrevious retrieves the aggregate for the previous period (for trend calculation)
	GetAggregatePrevious(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID, periodType domain.PeriodType, periodStart time.Time) (*schema.InteractionAggregate, error)
}

// AnalyticsRepositoryImpl implements AnalyticsRepository
type AnalyticsRepositoryImpl struct {
	db *gorm.DB
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &AnalyticsRepositoryImpl{db: db}
}

// UpsertAggregate creates or updates an aggregate using ON CONFLICT
func (r *AnalyticsRepositoryImpl) UpsertAggregate(ctx context.Context, aggregate *schema.InteractionAggregate) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "entity_type"},
			{Name: "entity_id"},
			{Name: "period_type"},
			{Name: "period_start"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"views_total",
			"views_unique",
			"saves_total",
			"unsaves_total",
			"shares_total",
			"contacts_total",
			"booking_requests",
			"avg_time_on_page_sec",
			"engagement_score",
			"updated_at",
		}),
	}).Create(aggregate).Error
}

// GetAggregates retrieves aggregates for an entity within a time range
func (r *AnalyticsRepositoryImpl) GetAggregates(
	ctx context.Context,
	entityType domain.EntityType,
	entityID uuid.UUID,
	periodType domain.PeriodType,
	start, end time.Time,
) ([]*schema.InteractionAggregate, error) {
	var aggregates []*schema.InteractionAggregate

	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ? AND period_type = ? AND period_start BETWEEN ? AND ?",
			entityType.String(), entityID, periodType.String(), start, end).
		Order("period_start ASC").
		Find(&aggregates).Error

	if err != nil {
		return nil, err
	}

	return aggregates, nil
}

// GetAggregatePrevious retrieves the aggregate for the previous period
func (r *AnalyticsRepositoryImpl) GetAggregatePrevious(
	ctx context.Context,
	entityType domain.EntityType,
	entityID uuid.UUID,
	periodType domain.PeriodType,
	periodStart time.Time,
) (*schema.InteractionAggregate, error) {
	var aggregate schema.InteractionAggregate

	// Calculate previous period start based on period type
	var previousStart time.Time
	switch periodType {
	case domain.PeriodHour:
		previousStart = periodStart.Add(-1 * time.Hour)
	case domain.PeriodDay:
		previousStart = periodStart.AddDate(0, 0, -1)
	case domain.PeriodWeek:
		previousStart = periodStart.AddDate(0, 0, -7)
	default:
		previousStart = periodStart.AddDate(0, 0, -1)
	}

	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ? AND period_type = ? AND period_start = ?",
			entityType.String(), entityID, periodType.String(), previousStart).
		First(&aggregate).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No previous period data
		}
		return nil, err
	}

	return &aggregate, nil
}
