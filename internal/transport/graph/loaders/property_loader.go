package loaders

import (
	"context"
	"sync"
	"time"

	"hauslet/internal/modules/property/domain"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// PropertyLoader batches property fetches by ID using a time-window
// dataloader pattern. When multiple goroutines (e.g., concurrent GraphQL
// field resolvers) call Load() within a short window, the loader collects
// all requested IDs and dispatches a single batch query.
type PropertyLoader struct {
	svc propertyservice.PropertyService

	mu     sync.Mutex
	batch  *propertyBatch
	cache  map[uuid.UUID]*propertyResult
	window time.Duration
}

type propertyResult struct {
	property *domain.Property
	err      error
}

type propertyBatch struct {
	keys   []uuid.UUID
	done   chan struct{}
	result map[uuid.UUID]*domain.Property
	err    error
}

const (
	defaultPropertyWindow = 2 * time.Millisecond
	maxPropertyBatchSize  = 200
)

func NewPropertyLoader(svc propertyservice.PropertyService) *PropertyLoader {
	return &PropertyLoader{
		svc:    svc,
		cache:  make(map[uuid.UUID]*propertyResult),
		window: defaultPropertyWindow,
	}
}

// Load returns a property by ID. Concurrent calls within the batching
// window are automatically grouped into a single database query.
func (l *PropertyLoader) Load(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	// Fast path: check cache
	l.mu.Lock()
	if cached, ok := l.cache[id]; ok {
		l.mu.Unlock()
		return cached.property, cached.err
	}

	b := l.getCurrentBatch(ctx, id)
	l.mu.Unlock()

	<-b.done

	if b.err != nil {
		return nil, b.err
	}

	if p, ok := b.result[id]; ok {
		return p, nil
	}
	return nil, nil
}

// getCurrentBatch returns the current batch, creating one if needed.
// Must be called with l.mu held.
func (l *PropertyLoader) getCurrentBatch(ctx context.Context, id uuid.UUID) *propertyBatch {
	if l.batch == nil {
		l.batch = &propertyBatch{
			keys: make([]uuid.UUID, 0, 32),
			done: make(chan struct{}),
		}
		go l.dispatchAfterWindow(ctx)
	}

	l.batch.keys = append(l.batch.keys, id)
	b := l.batch

	if len(l.batch.keys) >= maxPropertyBatchSize {
		l.batch = nil
		go l.dispatchBatch(ctx, b)
	}

	return b
}

func (l *PropertyLoader) dispatchAfterWindow(ctx context.Context) {
	time.Sleep(l.window)

	l.mu.Lock()
	b := l.batch
	l.batch = nil
	l.mu.Unlock()

	if b != nil {
		l.dispatchBatch(ctx, b)
	}
}

func (l *PropertyLoader) dispatchBatch(ctx context.Context, b *propertyBatch) {
	defer close(b.done)

	seen := make(map[uuid.UUID]bool, len(b.keys))
	unique := make([]uuid.UUID, 0, len(b.keys))
	for _, id := range b.keys {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	fetched, err := l.svc.GetPropertiesByIDs(ctx, unique)
	if err != nil {
		b.err = err
		return
	}

	b.result = make(map[uuid.UUID]*domain.Property, len(fetched))
	for i := range fetched {
		pCopy := fetched[i]
		b.result[pCopy.ID] = &pCopy
	}

	l.mu.Lock()
	for _, id := range unique {
		if p, ok := b.result[id]; ok {
			l.cache[id] = &propertyResult{property: p}
		} else {
			l.cache[id] = &propertyResult{property: nil}
		}
	}
	l.mu.Unlock()
}

// LoadMany fetches properties for the provided IDs using a single service call.
// This bypasses the time-window since the caller already has all IDs.
func (l *PropertyLoader) LoadMany(ctx context.Context, ids []uuid.UUID) ([]*domain.Property, error) {
	if len(ids) == 0 {
		return []*domain.Property{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.Property, len(ids))
	missing := make([]uuid.UUID, 0)
	missingIdx := make([]int, 0)

	for i, id := range ids {
		if cached, ok := l.cache[id]; ok {
			result[i] = cached.property
		} else {
			missing = append(missing, id)
			missingIdx = append(missingIdx, i)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	fetched, err := l.svc.GetPropertiesByIDs(ctx, missing)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]*domain.Property, len(fetched))
	for i := range fetched {
		pCopy := fetched[i]
		byID[pCopy.ID] = &pCopy
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i, idx := range missingIdx {
		id := missing[i]
		if p, ok := byID[id]; ok {
			result[idx] = p
			l.cache[id] = &propertyResult{property: p}
		} else {
			l.cache[id] = &propertyResult{property: nil}
		}
	}

	return result, nil
}
