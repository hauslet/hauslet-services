package loaders

import (
	"context"
	"sync"

	"hauslet/internal/modules/booking/domain"
	bookingservice "hauslet/internal/modules/booking/service"

	"github.com/google/uuid"
)

// BookingLoader batches booking fetches by ID for a single request.
type BookingLoader struct {
	svc   bookingservice.BookingService
	mu    sync.Mutex
	cache map[uuid.UUID]*domain.Booking
}

func NewBookingLoader(svc bookingservice.BookingService) *BookingLoader {
	return &BookingLoader{
		svc:   svc,
		cache: make(map[uuid.UUID]*domain.Booking),
	}
}

// Load returns a booking by ID, using cached/batched lookups within the request.
func (l *BookingLoader) Load(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	l.mu.Lock()
	if booking, ok := l.cache[id]; ok {
		l.mu.Unlock()
		return booking, nil
	}
	l.mu.Unlock()

	bookings, err := l.LoadMany(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	if len(bookings) > 0 {
		return bookings[0], nil
	}
	return nil, nil
}

// LoadMany fetches bookings for the provided IDs using a single service call.
func (l *BookingLoader) LoadMany(ctx context.Context, ids []uuid.UUID) ([]*domain.Booking, error) {
	if len(ids) == 0 {
		return []*domain.Booking{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.Booking, len(ids))
	missing := make([]uuid.UUID, 0)
	missingIdx := make([]int, 0)

	for i, id := range ids {
		if booking, ok := l.cache[id]; ok {
			result[i] = booking
		} else {
			missing = append(missing, id)
			missingIdx = append(missingIdx, i)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	fetched, err := l.svc.GetBookingsByIDs(ctx, missing)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]*domain.Booking, len(fetched))
	for i := range fetched {
		bookingCopy := fetched[i]
		byID[bookingCopy.ID] = bookingCopy
	}

	l.mu.Lock()
	for id, booking := range byID {
		l.cache[id] = booking
	}
	for i, idx := range missingIdx {
		if booking, ok := byID[missing[i]]; ok {
			result[idx] = booking
		}
	}
	l.mu.Unlock()

	return result, nil
}
