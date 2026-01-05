package loaders

import (
	"context"
	"sync"

	"hauslet/internal/modules/property/domain"
	propertyservice "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

// ListingLoader batches listing fetches by ID for a single request.
type ListingLoader struct {
	svc   propertyservice.PropertyService
	mu    sync.Mutex
	cache map[uuid.UUID]*domain.Listing
}

func NewListingLoader(svc propertyservice.PropertyService) *ListingLoader {
	return &ListingLoader{
		svc:   svc,
		cache: make(map[uuid.UUID]*domain.Listing),
	}
}

// Load returns a listing by ID, using cached/batched lookups within the request.
func (l *ListingLoader) Load(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	l.mu.Lock()
	if listing, ok := l.cache[id]; ok {
		l.mu.Unlock()
		return listing, nil
	}
	l.mu.Unlock()

	listings, err := l.LoadMany(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	if len(listings) > 0 {
		return listings[0], nil
	}
	return nil, nil
}

// LoadMany fetches listings for the provided IDs using a single service call.
// Note: Media is not preloaded by default. Use LoadWithMedia for media.
func (l *ListingLoader) LoadMany(ctx context.Context, ids []uuid.UUID) ([]*domain.Listing, error) {
	return l.loadManyInternal(ctx, ids, false)
}

// LoadWithMedia loads a listing with media preloaded.
func (l *ListingLoader) LoadWithMedia(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	listings, err := l.LoadManyWithMedia(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	if len(listings) > 0 {
		return listings[0], nil
	}
	return nil, nil
}

// LoadManyWithMedia fetches listings with media preloaded.
func (l *ListingLoader) LoadManyWithMedia(ctx context.Context, ids []uuid.UUID) ([]*domain.Listing, error) {
	return l.loadManyInternal(ctx, ids, true)
}

// loadManyInternal is the internal implementation that handles preloadMedia flag.
func (l *ListingLoader) loadManyInternal(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]*domain.Listing, error) {
	if len(ids) == 0 {
		return []*domain.Listing{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.Listing, len(ids))
	missing := make([]uuid.UUID, 0)
	missingIdx := make([]int, 0)

	for i, id := range ids {
		// Only use cache if we don't need media or if cached listing has media
		if listing, ok := l.cache[id]; ok && (!preloadMedia || len(listing.Media) > 0) {
			result[i] = listing
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
	for id, listing := range byID {
		l.cache[id] = listing
	}
	for i, idx := range missingIdx {
		if listing, ok := byID[missing[i]]; ok {
			result[idx] = listing
		}
	}
	l.mu.Unlock()

	return result, nil
}
