package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/review/domain"

	"github.com/google/uuid"
)

// ============================================================================
// Stats Query Resolvers
// ============================================================================

// ListingStats retrieves statistics for a listing
func (r *Resolver) ListingStats(ctx context.Context, listingID string) (*domain.ListingStats, error) {
	lid, err := uuid.Parse(listingID)
	if err != nil {
		r.log.Logf("ERROR invalid listing ID %s: %v", listingID, err)
		return nil, fmt.Errorf("invalid listing ID")
	}

	stats, err := r.reviewService.GetListingStats(ctx, lid)
	if err != nil {
		if err == domain.ErrStatsNotFound {
			return nil, nil
		}
		r.log.Logf("ERROR failed to get listing stats for %s: %v", listingID, err)
		return nil, err
	}

	return stats, nil
}

// HostStats retrieves statistics for a host
func (r *Resolver) HostStats(ctx context.Context, hostID string) (*domain.HostStats, error) {
	hid, err := uuid.Parse(hostID)
	if err != nil {
		r.log.Logf("ERROR invalid host ID %s: %v", hostID, err)
		return nil, fmt.Errorf("invalid host ID")
	}

	stats, err := r.reviewService.GetHostStats(ctx, hid)
	if err != nil {
		if err == domain.ErrStatsNotFound {
			return nil, nil
		}
		r.log.Logf("ERROR failed to get host stats for %s: %v", hostID, err)
		return nil, err
	}

	return stats, nil
}
