package service

import (
	"context"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/modules/interactions/repository"
	"hauslet/internal/modules/interactions/repository/schema"
	"log/slog"

	"github.com/google/uuid"
)

// ReaderServiceImpl implements ReaderService
type ReaderServiceImpl struct {
	interactionRepo repository.InteractionRepository
	analyticsRepo   repository.AnalyticsRepository
	log             *slog.Logger
}

// NewReaderService creates a new reader service
func NewReaderService(
	interactionRepo repository.InteractionRepository,
	analyticsRepo repository.AnalyticsRepository,
	log *slog.Logger,
) ReaderService {
	return &ReaderServiceImpl{
		interactionRepo: interactionRepo,
		analyticsRepo:   analyticsRepo,
		log:             log,
	}
}

// GetListingAnalytics retrieves analytics for a listing
func (s *ReaderServiceImpl) GetListingAnalytics(ctx context.Context, listingID uuid.UUID, days int) (*domain.ListingAnalytics, error) {
	period := domain.NewPeriod(days)

	// Get aggregates for the period
	aggregates, err := s.analyticsRepo.GetAggregates(
		ctx,
		domain.EntityTypeListing,
		listingID,
		domain.PeriodDay,
		period.StartDate,
		period.EndDate,
	)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get aggregates", "error", err, "listing_id", listingID)
		}
		return nil, err
	}

	// Calculate totals and metrics
	analytics := &domain.ListingAnalytics{
		ListingID: listingID,
		Period:    period,
	}

	for _, agg := range aggregates {
		domainAgg := schema.MapAggregateToDomain(agg)

		analytics.TotalViews += domainAgg.ViewsTotal
		analytics.UniqueViews += domainAgg.ViewsUnique
		analytics.TotalSaves += domainAgg.SavesTotal
		analytics.TotalShares += domainAgg.SharesTotal
		analytics.TotalContacts += domainAgg.ContactsTotal
		analytics.BookingRequests += domainAgg.BookingRequests
	}

	// Calculate net saves
	var totalUnsaves int64
	for _, agg := range aggregates {
		totalUnsaves += agg.UnsavesTotal
	}
	analytics.NetSaves = analytics.TotalSaves - totalUnsaves

	// Calculate conversion rate
	if analytics.UniqueViews > 0 {
		highValueActions := float64(analytics.TotalContacts + analytics.BookingRequests)
		analytics.ConversionRate = (highValueActions / float64(analytics.UniqueViews)) * 100

		engagementActions := float64(analytics.TotalSaves + analytics.TotalShares + analytics.TotalContacts + analytics.BookingRequests)
		analytics.EngagementRate = (engagementActions / float64(analytics.UniqueViews)) * 100
	}

	// Calculate average time on page (weighted average)
	totalTime := 0
	totalViews := 0
	for _, agg := range aggregates {
		if agg.ViewsTotal > 0 {
			totalTime += agg.AvgTimeOnPageSec * int(agg.ViewsTotal)
			totalViews += int(agg.ViewsTotal)
		}
	}
	if totalViews > 0 {
		analytics.AvgTimeOnPage = totalTime / totalViews
	}

	// Calculate trends (compare to previous period)
	s.calculateTrends(ctx, analytics, listingID, period)

	return analytics, nil
}

// GetUserHistory retrieves a user's recent interaction history
func (s *ReaderServiceImpl) GetUserHistory(ctx context.Context, userID uuid.UUID, limit int) (*domain.UserActivityHistory, error) {
	schemaInteractions, err := s.interactionRepo.GetUserHistory(ctx, userID, limit)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to get user history", "error", err, "user_id", userID)
		}
		return nil, err
	}

	interactions := schema.MapInteractionsToDomain(schemaInteractions)

	return &domain.UserActivityHistory{
		UserID:       userID,
		Interactions: interactions,
		TotalCount:   len(interactions),
	}, nil
}

// calculateTrends compares current period to previous period
func (s *ReaderServiceImpl) calculateTrends(ctx context.Context, analytics *domain.ListingAnalytics, listingID uuid.UUID, currentPeriod domain.Period) {
	// Get previous period aggregates
	previousStart := currentPeriod.StartDate.AddDate(0, 0, -currentPeriod.Days)

	previousAggregates, err := s.analyticsRepo.GetAggregates(
		ctx,
		domain.EntityTypeListing,
		listingID,
		domain.PeriodDay,
		previousStart,
		currentPeriod.StartDate,
	)
	if err != nil {
		return // Silently fail, trends are optional
	}

	var previousViews, previousSaves int64
	for _, agg := range previousAggregates {
		previousViews += agg.ViewsTotal
		previousSaves += agg.SavesTotal
	}

	// Calculate percentage changes
	if previousViews > 0 {
		analytics.ViewsTrend = ((float64(analytics.TotalViews) - float64(previousViews)) / float64(previousViews)) * 100
	}

	if previousSaves > 0 {
		analytics.SavesTrend = ((float64(analytics.TotalSaves) - float64(previousSaves)) / float64(previousSaves)) * 100
	}

	// Engagement trend (simplified)
	if previousViews > 0 {
		previousEngagement := float64(previousSaves) / float64(previousViews)
		currentEngagement := analytics.EngagementRate / 100
		if previousEngagement > 0 {
			analytics.EngagementTrend = ((currentEngagement - previousEngagement) / previousEngagement) * 100
		}
	}
}
