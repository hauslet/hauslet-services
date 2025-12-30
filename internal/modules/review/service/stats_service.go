package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/review/domain"

	"github.com/google/uuid"
)

// ================================================================
// STATISTICS OPERATIONS
// ================================================================

// GetListingStats retrieves cached statistics for a listing
func (s *ReviewServiceImpl) GetListingStats(ctx context.Context, listingID uuid.UUID) (*domain.ListingStats, error) {
	schemaStats, err := s.statsRepo.GetListingStats(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get listing stats: %w", err)
	}

	return domain.MapListingStatsFromSchema(schemaStats), nil
}

// GetHostStats retrieves cached statistics for a host
func (s *ReviewServiceImpl) GetHostStats(ctx context.Context, hostID uuid.UUID) (*domain.HostStats, error) {
	schemaStats, err := s.statsRepo.GetHostStats(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("failed to get host stats: %w", err)
	}

	return domain.MapHostStatsFromSchema(schemaStats), nil
}

// RecalculateListingStats recalculates and caches statistics for a listing
func (s *ReviewServiceImpl) RecalculateListingStats(ctx context.Context, listingID uuid.UUID) error {
	if err := s.statsRepo.RecalculateStats(ctx, listingID); err != nil {
		return fmt.Errorf("failed to recalculate listing stats: %w", err)
	}

	s.log.Logf("[INFO] recalculated stats for listing %s", listingID)

	return nil
}

// RecalculateHostStats recalculates and caches statistics for a host
func (s *ReviewServiceImpl) RecalculateHostStats(ctx context.Context, hostID uuid.UUID) error {
	if err := s.statsRepo.RecalculateStats(ctx, hostID); err != nil {
		return fmt.Errorf("failed to recalculate host stats: %w", err)
	}

	s.log.Logf("[INFO] recalculated stats for host %s", hostID)

	return nil
}

// RefreshAllStats recalculates statistics for all listings and hosts
func (s *ReviewServiceImpl) RefreshAllStats(ctx context.Context) error {
	// This is a maintenance operation that would need to:
	// 1. Get all listing IDs
	// 2. Recalculate stats for each
	// For now, we'll return an error indicating it's not implemented
	// TODO: Implement when we have a way to iterate over all listings/hosts

	s.log.Logf("[WARN] RefreshAllStats not yet implemented")
	return fmt.Errorf("RefreshAllStats not yet implemented")
}
