package loaders

import (
	"context"
	"sync"
	"time"

	"hauslet/internal/modules/review/domain"
	reviewservice "hauslet/internal/modules/review/service"

	"github.com/google/uuid"
)

// ListingStatsLoader batches listing stats fetches using a time-window
// dataloader pattern. When multiple goroutines (e.g., concurrent GraphQL
// field resolvers) call Load() within a short window, the loader collects
// all requested IDs and dispatches a single batch query.
//
// This solves the N+1 problem: instead of ~50 DB connections for ~50
// listings in a homeFeed query, we use 1 connection for 1 batch query.
type ListingStatsLoader struct {
	svc reviewservice.ReviewService

	// --- Batching state ---
	mu     sync.Mutex
	batch  *statsBatch                // current batch being assembled
	cache  map[uuid.UUID]*statsResult // per-request result cache
	window time.Duration              // how long to wait for more IDs before dispatching
}

// statsResult holds the result for a single listing ID.
type statsResult struct {
	stats *domain.ListingStats
	err   error
}

// statsBatch represents a batch of listing IDs being collected.
//
// Lifecycle:
// 1. First Load() call creates the batch and starts a timer
// 2. Subsequent Load() calls within the window add their IDs to the batch
// 3. When the timer fires OR the batch reaches maxBatchSize, dispatch() runs
// 4. dispatch() fetches all IDs in one DB call and fans out results to waiters
type statsBatch struct {
	keys   []uuid.UUID
	done   chan struct{} // closed when the batch result is ready
	result map[uuid.UUID]*domain.ListingStats
	err    error
}

const (
	// defaultStatsWindow is how long to wait for concurrent Load() calls
	// before dispatching. 2ms covers the typical goroutine scheduling
	// jitter in gqlgen's parallel resolver execution.
	defaultStatsWindow = 2 * time.Millisecond

	// maxStatsBatchSize forces a dispatch if we collect this many IDs
	// before the window expires. Prevents unbounded memory use.
	maxStatsBatchSize = 200
)

func NewListingStatsLoader(svc reviewservice.ReviewService) *ListingStatsLoader {
	return &ListingStatsLoader{
		svc:    svc,
		cache:  make(map[uuid.UUID]*statsResult),
		window: defaultStatsWindow,
	}
}

// Load returns listing stats by ID. Concurrent calls within the batching
// window are automatically grouped into a single database query.
func (l *ListingStatsLoader) Load(ctx context.Context, listingID uuid.UUID) (*domain.ListingStats, error) {
	// Fast path: check cache first
	l.mu.Lock()
	if cached, ok := l.cache[listingID]; ok {
		l.mu.Unlock()
		return cached.stats, cached.err
	}

	// Join or create a batch
	b := l.getCurrentBatch(ctx, listingID)
	l.mu.Unlock()

	// Wait for the batch to complete
	<-b.done

	// Read result
	if b.err != nil {
		return nil, b.err
	}

	stats := b.result[listingID]
	return stats, nil
}

// getCurrentBatch returns the current batch, creating one if needed.
// Must be called with l.mu held.
func (l *ListingStatsLoader) getCurrentBatch(ctx context.Context, id uuid.UUID) *statsBatch {
	if l.batch == nil {
		l.batch = &statsBatch{
			keys: make([]uuid.UUID, 0, 32),
			done: make(chan struct{}),
		}
		// Start the timer to dispatch this batch
		go l.dispatchAfterWindow(ctx)
	}

	l.batch.keys = append(l.batch.keys, id)

	b := l.batch

	// If we've hit the max batch size, dispatch immediately
	if len(l.batch.keys) >= maxStatsBatchSize {
		l.batch = nil // next Load() will create a new batch
		go l.dispatchBatch(ctx, b)
	}

	return b
}

// dispatchAfterWindow waits for the batching window, then dispatches.
func (l *ListingStatsLoader) dispatchAfterWindow(ctx context.Context) {
	time.Sleep(l.window)

	l.mu.Lock()
	b := l.batch
	l.batch = nil // next Load() will create a new batch
	l.mu.Unlock()

	if b != nil {
		l.dispatchBatch(ctx, b)
	}
}

// dispatchBatch executes the actual batched DB query and fans out results.
func (l *ListingStatsLoader) dispatchBatch(ctx context.Context, b *statsBatch) {
	defer close(b.done)

	// Deduplicate IDs
	seen := make(map[uuid.UUID]bool, len(b.keys))
	unique := make([]uuid.UUID, 0, len(b.keys))
	for _, id := range b.keys {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	// Single batch query for all collected IDs
	statsMap, err := l.svc.GetListingStatsBatch(ctx, unique)
	if err != nil {
		b.err = err
		return
	}

	b.result = statsMap

	// Populate cache for future Load() calls in this request
	l.mu.Lock()
	for _, id := range unique {
		if stats, ok := statsMap[id]; ok {
			l.cache[id] = &statsResult{stats: stats}
		} else {
			// Cache "not found" to avoid refetching
			l.cache[id] = &statsResult{stats: nil}
		}
	}
	l.mu.Unlock()
}

// LoadMany fetches stats for multiple listings in a batch.
// This is a convenience method that pre-populates the batch with all IDs.
func (l *ListingStatsLoader) LoadMany(ctx context.Context, listingIDs []uuid.UUID) ([]*domain.ListingStats, error) {
	if len(listingIDs) == 0 {
		return []*domain.ListingStats{}, nil
	}

	// For LoadMany, we bypass the time-window and do a direct batch fetch,
	// since the caller already has all the IDs.
	l.mu.Lock()
	result := make([]*domain.ListingStats, len(listingIDs))
	missing := make([]uuid.UUID, 0)
	missingIdx := make([]int, 0)

	for i, id := range listingIDs {
		if cached, ok := l.cache[id]; ok {
			result[i] = cached.stats
		} else {
			missing = append(missing, id)
			missingIdx = append(missingIdx, i)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	// Single batch query
	statsMap, err := l.svc.GetListingStatsBatch(ctx, missing)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i, idx := range missingIdx {
		id := missing[i]
		if stats, ok := statsMap[id]; ok {
			result[idx] = stats
			l.cache[id] = &statsResult{stats: stats}
		} else {
			l.cache[id] = &statsResult{stats: nil}
		}
	}

	return result, nil
}
