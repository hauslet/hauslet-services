package loaders

import (
	"context"
	"sync"

	"hauslet/internal/property/domain"
	propertyservice "hauslet/internal/property/service"

	"github.com/google/uuid"
)

// ListingsByPropertyLoader batches listing fetches by property ID for a single request.
type ListingsByPropertyLoader struct {
	svc   propertyservice.Service
	mu    sync.Mutex
	cache map[uuid.UUID][]domain.Listing
}

func NewListingsByPropertyLoader(svc propertyservice.Service) *ListingsByPropertyLoader {
	return &ListingsByPropertyLoader{
		svc:   svc,
		cache: make(map[uuid.UUID][]domain.Listing),
	}
}

// Load returns listings for a property ID, using cached/batched lookups within the request.
func (l *ListingsByPropertyLoader) Load(ctx context.Context, propertyID uuid.UUID) ([]domain.Listing, error) {
	l.mu.Lock()
	if listings, ok := l.cache[propertyID]; ok {
		l.mu.Unlock()
		return listings, nil
	}
	l.mu.Unlock()

	result, err := l.LoadMany(ctx, []uuid.UUID{propertyID})
	if err != nil {
		return nil, err
	}
	if listings, ok := result[propertyID]; ok {
		return listings, nil
	}
	return []domain.Listing{}, nil
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
		if listings, ok := l.cache[propertyID]; ok {
			result[propertyID] = listings
		} else {
			missing = append(missing, propertyID)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	// Fetch all listings for the missing property IDs
	fetched, err := l.svc.GetListingsByPropertyIDs(ctx, missing)
	if err != nil {
		return nil, err
	}

	// Group listings by property ID
	byPropertyID := make(map[uuid.UUID][]domain.Listing)
	for _, listing := range fetched {
		byPropertyID[listing.PropertyID] = append(byPropertyID[listing.PropertyID], listing)
	}

	// Ensure all requested property IDs have an entry (even if empty)
	for _, propertyID := range missing {
		if _, ok := byPropertyID[propertyID]; !ok {
			byPropertyID[propertyID] = []domain.Listing{}
		}
	}

	l.mu.Lock()
	for propertyID, listings := range byPropertyID {
		l.cache[propertyID] = listings
		result[propertyID] = listings
	}
	l.mu.Unlock()

	return result, nil
}
