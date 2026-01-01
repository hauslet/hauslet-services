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
		r.log.Error("invalid listing ID", "listing_id", listingID, "error", err)
		return nil, fmt.Errorf("invalid listing ID")
	}

	stats, err := r.reviewService.GetListingStats(ctx, lid)
	if err != nil {
		if err == domain.ErrStatsNotFound {
			return nil, nil
		}
		r.log.Error("failed to get listing stats", "listing_id", listingID, "error", err)
		return nil, err
	}

	return stats, nil
}

// HostStats retrieves statistics for a host
func (r *Resolver) HostStats(ctx context.Context, hostID string) (*domain.HostStats, error) {
	hid, err := uuid.Parse(hostID)
	if err != nil {
		r.log.Error("invalid host ID", "host_id", hostID, "error", err)
		return nil, fmt.Errorf("invalid host ID")
	}

	stats, err := r.reviewService.GetHostStats(ctx, hid)
	if err != nil {
		if err == domain.ErrStatsNotFound {
			return nil, nil
		}
		r.log.Error("failed to get host stats", "host_id", hostID, "error", err)
		return nil, err
	}

	return stats, nil
}
