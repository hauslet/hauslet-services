package loaders

import (
	"context"
	"sync"

	"hauslet/internal/modules/property/domain"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// PropertyLoader batches property fetches by ID for a single request.
type PropertyLoader struct {
	svc   propertyservice.Service
	mu    sync.Mutex
	cache map[uuid.UUID]*domain.Property
}

func NewPropertyLoader(svc propertyservice.Service) *PropertyLoader {
	return &PropertyLoader{
		svc:   svc,
		cache: make(map[uuid.UUID]*domain.Property),
	}
}

// Load returns a property by ID, using cached/batched lookups within the request.
func (l *PropertyLoader) Load(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	l.mu.Lock()
	if p, ok := l.cache[id]; ok {
		l.mu.Unlock()
		return p, nil
	}
	l.mu.Unlock()

	properties, err := l.LoadMany(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	if len(properties) > 0 {
		return properties[0], nil
	}
	return nil, nil
}

// LoadMany fetches properties for the provided IDs using a single service call.
func (l *PropertyLoader) LoadMany(ctx context.Context, ids []uuid.UUID) ([]*domain.Property, error) {
	if len(ids) == 0 {
		return []*domain.Property{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.Property, len(ids))
	missing := make([]uuid.UUID, 0)
	missingIdx := make([]int, 0)

	for i, id := range ids {
		if p, ok := l.cache[id]; ok {
			result[i] = p
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
	for id, p := range byID {
		l.cache[id] = p
	}
	for i, idx := range missingIdx {
		if p, ok := byID[missing[i]]; ok {
			result[idx] = p
		}
	}
	l.mu.Unlock()

	return result, nil
}
