package loaders

import (
	"context"
	"sync"

	"hauslet/internal/modules/review/domain"
	reviewservice "hauslet/internal/modules/review/service"

	"github.com/google/uuid"
)

// ListingStatsLoader batches listing stats fetches for a single request.
type ListingStatsLoader struct {
	svc   reviewservice.ReviewService
	mu    sync.Mutex
	cache map[uuid.UUID]*domain.ListingStats
}

func NewListingStatsLoader(svc reviewservice.ReviewService) *ListingStatsLoader {
	return &ListingStatsLoader{
		svc:   svc,
		cache: make(map[uuid.UUID]*domain.ListingStats),
	}
}

// Load returns listing stats by ID, using cached/batched lookups within the request.
func (l *ListingStatsLoader) Load(ctx context.Context, listingID uuid.UUID) (*domain.ListingStats, error) {
	l.mu.Lock()
	if stats, ok := l.cache[listingID]; ok {
		l.mu.Unlock()
		return stats, nil
	}
	l.mu.Unlock()

	stats, err := l.LoadMany(ctx, []uuid.UUID{listingID})
	if err != nil {
		return nil, err
	}
	if len(stats) > 0 {
		return stats[0], nil
	}
	return nil, nil // Return nil if no stats found (no reviews yet)
}

// LoadMany fetches stats for multiple listings in a batch.
func (l *ListingStatsLoader) LoadMany(ctx context.Context, listingIDs []uuid.UUID) ([]*domain.ListingStats, error) {
	if len(listingIDs) == 0 {
		return []*domain.ListingStats{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.ListingStats, len(listingIDs))
	missing := make([]uuid.UUID, 0)
	missingIdx := make([]int, 0)

	for i, id := range listingIDs {
		if stats, ok := l.cache[id]; ok {
			result[i] = stats
		} else {
			missing = append(missing, id)
			missingIdx = append(missingIdx, i)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	// Fetch from service (batch)
	statsMap, err := l.svc.GetListingStatsBatch(ctx, missing)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Update cache
	for id, stats := range statsMap {
		l.cache[id] = stats
	}

	// Populate result
	for i, idx := range missingIdx {
		id := missing[i]
		if stats, ok := statsMap[id]; ok {
			result[idx] = stats
		}
		// If not found in map, it remains nil (meaning no stats exist for this listing)
		// We still cache the nil to avoid refetching
		if _, ok := statsMap[id]; !ok {
			l.cache[id] = nil
		}
	}

	return result, nil
}
