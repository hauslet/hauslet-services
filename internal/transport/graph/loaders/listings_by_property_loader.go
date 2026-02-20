package loaders

import (
	"context"
	"sync"
	"time"

	"hauslet/internal/modules/property/domain"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// ListingsByPropertyLoader batches listing fetches by property ID using
// a time-window dataloader pattern.
type ListingsByPropertyLoader struct {
	svc propertyservice.PropertyService

	mu     sync.Mutex
	batch  *listingsByPropertyBatch
	cache  map[uuid.UUID]*listingsByPropertyResult
	window time.Duration
}

type listingsByPropertyResult struct {
	listings []domain.Listing
}

type listingsByPropertyBatch struct {
	keys   []uuid.UUID
	done   chan struct{}
	result map[uuid.UUID][]domain.Listing
	err    error
}

const (
	defaultListingsByPropertyWindow = 2 * time.Millisecond
	maxListingsByPropertyBatchSize  = 200
)

func NewListingsByPropertyLoader(svc propertyservice.PropertyService) *ListingsByPropertyLoader {
	return &ListingsByPropertyLoader{
		svc:    svc,
		cache:  make(map[uuid.UUID]*listingsByPropertyResult),
		window: defaultListingsByPropertyWindow,
	}
}

// Load returns listings for a property ID. Concurrent calls within the
// batching window are automatically grouped into a single database query.
func (l *ListingsByPropertyLoader) Load(ctx context.Context, propertyID uuid.UUID) ([]domain.Listing, error) {
	l.mu.Lock()
	if cached, ok := l.cache[propertyID]; ok {
		l.mu.Unlock()
		return cached.listings, nil
	}

	b := l.getCurrentBatch(ctx, propertyID)
	l.mu.Unlock()

	<-b.done

	if b.err != nil {
		return nil, b.err
	}

	if listings, ok := b.result[propertyID]; ok {
		return listings, nil
	}
	return []domain.Listing{}, nil
}

// getCurrentBatch returns the current batch, creating one if needed.
// Must be called with l.mu held.
func (l *ListingsByPropertyLoader) getCurrentBatch(ctx context.Context, id uuid.UUID) *listingsByPropertyBatch {
	if l.batch == nil {
		l.batch = &listingsByPropertyBatch{
			keys: make([]uuid.UUID, 0, 32),
			done: make(chan struct{}),
		}
		go l.dispatchAfterWindow(ctx)
	}

	l.batch.keys = append(l.batch.keys, id)
	b := l.batch

	if len(l.batch.keys) >= maxListingsByPropertyBatchSize {
		l.batch = nil
		go l.dispatchBatch(ctx, b)
	}

	return b
}

func (l *ListingsByPropertyLoader) dispatchAfterWindow(ctx context.Context) {
	time.Sleep(l.window)

	l.mu.Lock()
	b := l.batch
	l.batch = nil
	l.mu.Unlock()

	if b != nil {
		l.dispatchBatch(ctx, b)
	}
}

func (l *ListingsByPropertyLoader) dispatchBatch(ctx context.Context, b *listingsByPropertyBatch) {
	defer close(b.done)

	seen := make(map[uuid.UUID]bool, len(b.keys))
	unique := make([]uuid.UUID, 0, len(b.keys))
	for _, id := range b.keys {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	fetched, err := l.svc.GetListingsByPropertyIDs(ctx, unique)
	if err != nil {
		b.err = err
		return
	}

	// Group listings by property ID
	b.result = make(map[uuid.UUID][]domain.Listing)
	for _, listing := range fetched {
		b.result[listing.PropertyID] = append(b.result[listing.PropertyID], listing)
	}

	// Ensure all requested property IDs have an entry (even if empty)
	for _, propertyID := range unique {
		if _, ok := b.result[propertyID]; !ok {
			b.result[propertyID] = []domain.Listing{}
		}
	}

	l.mu.Lock()
	for propertyID, listings := range b.result {
		l.cache[propertyID] = &listingsByPropertyResult{listings: listings}
	}
	l.mu.Unlock()
}

// LoadMany fetches listings for the provided property IDs using a single service call.
func (l *ListingsByPropertyLoader) LoadMany(ctx context.Context, propertyIDs []uuid.UUID) (map[uuid.UUID][]domain.Listing, error) {
	if len(propertyIDs) == 0 {
		return map[uuid.UUID][]domain.Listing{}, nil
	}

	l.mu.Lock()
	result := make(map[uuid.UUID][]domain.Listing, len(propertyIDs))
	missing := make([]uuid.UUID, 0)

	for _, propertyID := range propertyIDs {
		if cached, ok := l.cache[propertyID]; ok {
			result[propertyID] = cached.listings
		} else {
			missing = append(missing, propertyID)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	fetched, err := l.svc.GetListingsByPropertyIDs(ctx, missing)
	if err != nil {
		return nil, err
	}

	byPropertyID := make(map[uuid.UUID][]domain.Listing)
	for _, listing := range fetched {
		byPropertyID[listing.PropertyID] = append(byPropertyID[listing.PropertyID], listing)
	}

	for _, propertyID := range missing {
		if _, ok := byPropertyID[propertyID]; !ok {
			byPropertyID[propertyID] = []domain.Listing{}
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for propertyID, listings := range byPropertyID {
		l.cache[propertyID] = &listingsByPropertyResult{listings: listings}
		result[propertyID] = listings
	}

	return result, nil
}
