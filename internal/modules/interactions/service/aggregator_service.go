package service

import (
	"context"
	"encoding/json"
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

	// Get all interactions for this period
	// Note: This is simplified - in production you'd want to batch this
	// by querying distinct entity_id values and processing in chunks

	// For now, we'll aggregate by querying the DB
	// In a high-traffic system, you'd use a more efficient approach
	if s.log != nil {
		s.log.Info("starting aggregation",
			"period_type", periodType.String(),
			"period_start", periodStart,
			"period_end", periodEnd)
	}

	// Get unique entity combinations from the period
	entities, err := s.getUniqueEntities(ctx, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to get unique entities: %w", err)
	}

	successCount := 0
	errorCount := 0

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
			errorCount++
			continue
		}
		successCount++
	}

	if s.log != nil {
		s.log.Info("aggregation completed",
			"period_type", periodType.String(),
			"period_start", periodStart,
			"success_count", successCount,
			"error_count", errorCount)
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
func (s *AggregatorServiceImpl) getUniqueEntities(ctx context.Context, start, end time.Time) ([]repository.EntityKey, error) {
	// Delegate to repository method
	return s.interactionRepo.GetUniqueEntities(ctx, start, end)
}

// aggregateForEntity aggregates interactions for a specific entity
func (s *AggregatorServiceImpl) aggregateForEntity(
	ctx context.Context,
	entityType domain.EntityType,
	entityID uuid.UUID,
	periodType domain.PeriodType,
	periodStart, periodEnd time.Time,
) error {
	// Get all interactions for this entity in the period
	interactions, err := s.interactionRepo.GetEntityInteractions(ctx, entityType, entityID, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to get entity interactions: %w", err)
	}

	if len(interactions) == 0 {
		return nil // No interactions to aggregate
	}

	// Calculate metrics
	aggregate := s.calculateMetrics(interactions, entityType, entityID, periodType, periodStart)

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

// calculateMetrics calculates aggregate metrics from raw interactions
func (s *AggregatorServiceImpl) calculateMetrics(
	interactions []*schema.Interaction,
	entityType domain.EntityType,
	entityID uuid.UUID,
	periodType domain.PeriodType,
	periodStart time.Time,
) *schema.InteractionAggregate {
	aggregate := &schema.InteractionAggregate{
		ID:          uuid.New(),
		EntityType:  entityType.String(),
		EntityID:    entityID,
		PeriodType:  periodType.String(),
		PeriodStart: periodStart,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Track unique sessions for unique view counting
	uniqueSessions := make(map[string]bool)
	var totalTimeOnPage int64

	for _, interaction := range interactions {
		// Skip bot interactions for most metrics
		if interaction.IsBot {
			continue
		}

		interactionType := domain.ParseInteractionType(interaction.InteractionType)

		switch interactionType {
		case domain.InteractionViewListing, domain.InteractionViewListingDetail:
			aggregate.ViewsTotal++
			uniqueSessions[interaction.SessionID] = true

		case domain.InteractionViewMedia:
			// Counted separately but not in total views

		case domain.InteractionViewMap:
			// Counted separately but not in total views

		case domain.InteractionSaveListing:
			aggregate.SavesTotal++

		case domain.InteractionUnsaveListing:
			aggregate.UnsavesTotal++

		case domain.InteractionShareListing:
			aggregate.SharesTotal++

		case domain.InteractionContactOwner:
			aggregate.ContactsTotal++

		case domain.InteractionBookingRequest:
			aggregate.BookingRequests++

		case domain.InteractionTimeMilestone:
			// Extract time spent from context if available
			if len(interaction.Context) > 0 {
				var contextMap map[string]interface{}
				if err := json.Unmarshal(interaction.Context, &contextMap); err == nil {
					if timeSpent, ok := contextMap["timeSpent"].(float64); ok {
						totalTimeOnPage += int64(timeSpent)
					}
				}
			}
		}
	}

	// Calculate unique views
	aggregate.ViewsUnique = int64(len(uniqueSessions))

	// Calculate average time on page
	if aggregate.ViewsTotal > 0 && totalTimeOnPage > 0 {
		aggregate.AvgTimeOnPageSec = int(totalTimeOnPage / aggregate.ViewsTotal)
	}

	// Calculate engagement score
	aggregate.EngagementScore = s.calculateEngagementScore(aggregate)

	return aggregate
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
