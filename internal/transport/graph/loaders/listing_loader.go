package loaders

import (
	"context"
	"sync"
	"time"

	"hauslet/internal/modules/property/domain"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// ListingLoader batches listing fetches by ID using a time-window
// dataloader pattern.
type ListingLoader struct {
	svc propertyservice.PropertyService

	mu     sync.Mutex
	batch  *listingBatch
	cache  map[uuid.UUID]*listingResult
	window time.Duration
}

type listingResult struct {
	listing  *domain.Listing
	hasMedia bool // tracks whether this cached entry has media loaded
}

type listingBatch struct {
	keys         []uuid.UUID
	preloadMedia bool
	done         chan struct{}
	result       map[uuid.UUID]*domain.Listing
	err          error
}

const (
	defaultListingWindow = 2 * time.Millisecond
	maxListingBatchSize  = 200
)

func NewListingLoader(svc propertyservice.PropertyService) *ListingLoader {
	return &ListingLoader{
		svc:    svc,
		cache:  make(map[uuid.UUID]*listingResult),
		window: defaultListingWindow,
	}
}

// Load returns a listing by ID. Concurrent calls within the batching
// window are automatically grouped into a single database query.
func (l *ListingLoader) Load(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	return l.loadInternal(ctx, id, false)
}

// LoadWithMedia loads a listing with media preloaded.
func (l *ListingLoader) LoadWithMedia(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	return l.loadInternal(ctx, id, true)
}

func (l *ListingLoader) loadInternal(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error) {
	l.mu.Lock()
	if cached, ok := l.cache[id]; ok && (!preloadMedia || cached.hasMedia) {
		l.mu.Unlock()
		return cached.listing, nil
	}

	b := l.getCurrentBatch(ctx, id, preloadMedia)
	l.mu.Unlock()

	<-b.done

	if b.err != nil {
		return nil, b.err
	}

	if listing, ok := b.result[id]; ok {
		return listing, nil
	}
	return nil, nil
}

// getCurrentBatch returns the current batch, creating one if needed.
// Must be called with l.mu held.
func (l *ListingLoader) getCurrentBatch(ctx context.Context, id uuid.UUID, preloadMedia bool) *listingBatch {
	// If a batch exists but differs in preloadMedia, dispatch it first
	if l.batch != nil && l.batch.preloadMedia != preloadMedia {
		old := l.batch
		l.batch = nil
		go l.dispatchBatch(ctx, old)
	}

	if l.batch == nil {
		l.batch = &listingBatch{
			keys:         make([]uuid.UUID, 0, 32),
			preloadMedia: preloadMedia,
			done:         make(chan struct{}),
		}
		go l.dispatchAfterWindow(ctx)
	}

	l.batch.keys = append(l.batch.keys, id)
	b := l.batch

	if len(l.batch.keys) >= maxListingBatchSize {
		l.batch = nil
		go l.dispatchBatch(ctx, b)
	}

	return b
}

func (l *ListingLoader) dispatchAfterWindow(ctx context.Context) {
	time.Sleep(l.window)

	l.mu.Lock()
	b := l.batch
	l.batch = nil
	l.mu.Unlock()

	if b != nil {
		l.dispatchBatch(ctx, b)
	}
}

func (l *ListingLoader) dispatchBatch(ctx context.Context, b *listingBatch) {
	defer close(b.done)

	seen := make(map[uuid.UUID]bool, len(b.keys))
	unique := make([]uuid.UUID, 0, len(b.keys))
	for _, id := range b.keys {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	fetched, err := l.svc.GetListingsByIDs(ctx, unique, b.preloadMedia)
	if err != nil {
		b.err = err
		return
	}

	b.result = make(map[uuid.UUID]*domain.Listing, len(fetched))
	for i := range fetched {
		listingCopy := fetched[i]
		b.result[listingCopy.ID] = &listingCopy
	}

	l.mu.Lock()
	for _, id := range unique {
		if listing, ok := b.result[id]; ok {
			l.cache[id] = &listingResult{
				listing:  listing,
				hasMedia: b.preloadMedia || len(listing.Media) > 0,
			}
		} else {
			l.cache[id] = &listingResult{listing: nil}
		}
	}
	l.mu.Unlock()
}

// LoadMany fetches listings for the provided IDs using a single service call.
func (l *ListingLoader) LoadMany(ctx context.Context, ids []uuid.UUID) ([]*domain.Listing, error) {
	return l.loadManyInternal(ctx, ids, false)
}

// LoadManyWithMedia fetches listings with media preloaded.
func (l *ListingLoader) LoadManyWithMedia(ctx context.Context, ids []uuid.UUID) ([]*domain.Listing, error) {
	return l.loadManyInternal(ctx, ids, true)
}

func (l *ListingLoader) loadManyInternal(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]*domain.Listing, error) {
	if len(ids) == 0 {
		return []*domain.Listing{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.Listing, len(ids))
	missing := make([]uuid.UUID, 0)
	missingIdx := make([]int, 0)

	for i, id := range ids {
		if cached, ok := l.cache[id]; ok && (!preloadMedia || cached.hasMedia) {
			result[i] = cached.listing
		} else {
			missing = append(missing, id)
			missingIdx = append(missingIdx, i)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	fetched, err := l.svc.GetListingsByIDs(ctx, missing, preloadMedia)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]*domain.Listing, len(fetched))
	for i := range fetched {
		listingCopy := fetched[i]
		byID[listingCopy.ID] = &listingCopy
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i, idx := range missingIdx {
		id := missing[i]
		if listing, ok := byID[id]; ok {
			result[idx] = listing
			l.cache[id] = &listingResult{
				listing:  listing,
				hasMedia: preloadMedia || len(listing.Media) > 0,
			}
		} else {
			l.cache[id] = &listingResult{listing: nil}
		}
	}

	return result, nil
}
