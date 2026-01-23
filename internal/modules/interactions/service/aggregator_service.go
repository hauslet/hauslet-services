package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/modules/interactions/repository"
	"hauslet/internal/modules/interactions/repository/schema"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// AggregatorService handles aggregation of raw interactions into analytics
type AggregatorService interface {
	// AggregateHourly aggregates interactions for a specific hour
	AggregateHourly(ctx context.Context, periodStart time.Time) error

	// AggregateDaily aggregates interactions for a specific day
	AggregateDaily(ctx context.Context, periodStart time.Time) error

	// AggregateForAllEntities aggregates for all entities in the given period
	AggregateForAllEntities(ctx context.Context, periodType domain.PeriodType, periodStart time.Time) error
}

// AggregatorServiceImpl implements AggregatorService
type AggregatorServiceImpl struct {
	interactionRepo repository.InteractionRepository
	analyticsRepo   repository.AnalyticsRepository
	log             *slog.Logger
}

// NewAggregatorService creates a new aggregator service
func NewAggregatorService(
	interactionRepo repository.InteractionRepository,
	analyticsRepo repository.AnalyticsRepository,
	log *slog.Logger,
) AggregatorService {
	return &AggregatorServiceImpl{
		interactionRepo: interactionRepo,
		analyticsRepo:   analyticsRepo,
		log:             log,
	}
}

// AggregateForAllEntities aggregates interactions for all entities in the period
func (s *AggregatorServiceImpl) AggregateForAllEntities(
	ctx context.Context,
	periodType domain.PeriodType,
	periodStart time.Time,
) error {
	// Calculate period end based on type
	periodEnd := s.calculatePeriodEnd(periodStart, periodType)

	if s.log != nil {
		s.log.Info("starting aggregation",
			"period_type", periodType.String(),
			"period_start", periodStart,
			"period_end", periodEnd)
	}

	totalSuccess := 0
	totalError := 0
	batchSize := 1000
	offset := 0

	for {
		// Get unique entity combinations in batches
		entities, err := s.getUniqueEntities(ctx, periodStart, periodEnd, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to get unique entities: %w", err)
		}

		if len(entities) == 0 {
			break // No more entities
		}

		// Aggregate for each entity
		for _, entity := range entities {
			err := s.aggregateForEntity(ctx, entity.EntityType, entity.EntityID, periodType, periodStart, periodEnd)
			if err != nil {
				if s.log != nil {
					s.log.Error("failed to aggregate for entity",
						"error", err,
						"entity_type", entity.EntityType,
						"entity_id", entity.EntityID)
				}
				totalError++
				continue
			}
			totalSuccess++
		}

		offset += batchSize
	}

	if s.log != nil {
		s.log.Info("aggregation completed",
			"period_type", periodType.String(),
			"period_start", periodStart,
			"success_count", totalSuccess,
			"error_count", totalError)
	}

	return nil
}

// AggregateHourly aggregates interactions for a specific hour
func (s *AggregatorServiceImpl) AggregateHourly(ctx context.Context, periodStart time.Time) error {
	// Truncate to hour boundary
	hourStart := periodStart.Truncate(time.Hour)
	return s.AggregateForAllEntities(ctx, domain.PeriodHour, hourStart)
}

// AggregateDaily aggregates interactions for a specific day
func (s *AggregatorServiceImpl) AggregateDaily(ctx context.Context, periodStart time.Time) error {
	// Truncate to day boundary
	dayStart := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, periodStart.Location())
	return s.AggregateForAllEntities(ctx, domain.PeriodDay, dayStart)
}

// getUniqueEntities returns all unique entity combinations in the period
func (s *AggregatorServiceImpl) getUniqueEntities(ctx context.Context, start, end time.Time, limit, offset int) ([]repository.EntityKey, error) {
	// Delegate to repository method
	return s.interactionRepo.GetUniqueEntities(ctx, start, end, limit, offset)
}

// aggregateForEntity aggregates interactions for a specific entity
func (s *AggregatorServiceImpl) aggregateForEntity(
	ctx context.Context,
	entityType domain.EntityType,
	entityID uuid.UUID,
	periodType domain.PeriodType,
	periodStart, periodEnd time.Time,
) error {
	// Get aggregated metrics directly from DB
	aggregate, err := s.interactionRepo.GetAggregationMetrics(ctx, entityType, entityID, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to get aggregation metrics: %w", err)
	}

	// If no activity, we might still want to record a zero-record or skip.
	// Current logic: if no views, maybe skip?
	// But GetAggregationMetrics returns 0s if no rows match (count/sum returns 0 or null->0).
	// If ViewsTotal and ViewsUnique are 0, we can probably assume no interaction.
	if aggregate.ViewsTotal == 0 && aggregate.ViewsUnique == 0 && aggregate.SavesTotal == 0 {
		return nil // No significant interactions
	}

	// Populate metadata
	aggregate.ID = uuid.New()
	aggregate.EntityType = entityType.String()
	aggregate.EntityID = entityID
	aggregate.PeriodType = periodType.String()
	aggregate.PeriodStart = periodStart
	aggregate.CreatedAt = time.Now()
	aggregate.UpdatedAt = time.Now()

	// Calculate engagement score
	aggregate.EngagementScore = s.calculateEngagementScore(aggregate)

	// Upsert into aggregates table
	if err := s.analyticsRepo.UpsertAggregate(ctx, aggregate); err != nil {
		return fmt.Errorf("failed to upsert aggregate: %w", err)
	}

	if s.log != nil {
		s.log.Debug("aggregated entity",
			"entity_type", entityType.String(),
			"entity_id", entityID,
			"period_type", periodType.String(),
			"views_total", aggregate.ViewsTotal,
			"views_unique", aggregate.ViewsUnique)
	}

	return nil
}

// calculateEngagementScore calculates an engagement score based on metrics
func (s *AggregatorServiceImpl) calculateEngagementScore(agg *schema.InteractionAggregate) float64 {
	if agg.ViewsUnique == 0 {
		return 0
	}

	// Weighted scoring: saves=3, shares=2, contacts=5, bookings=10
	highValueActions := float64(
		agg.SavesTotal*3 +
			agg.SharesTotal*2 +
			agg.ContactsTotal*5 +
			agg.BookingRequests*10,
	)

	uniqueViews := float64(agg.ViewsUnique)

	// Normalize to 0-100 scale
	score := (highValueActions / uniqueViews) * 100
	if score > 100 {
		score = 100
	}

	return score
}

// calculatePeriodEnd calculates the end time based on period type
func (s *AggregatorServiceImpl) calculatePeriodEnd(start time.Time, periodType domain.PeriodType) time.Time {
	switch periodType {
	case domain.PeriodHour:
		return start.Add(time.Hour)
	case domain.PeriodDay:
		return start.AddDate(0, 0, 1)
	case domain.PeriodWeek:
		return start.AddDate(0, 0, 7)
	default:
		return start.Add(time.Hour)
	}
}
