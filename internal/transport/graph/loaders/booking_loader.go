package loaders

import (
	"context"
	"sync"
	"time"

	"hauslet/internal/modules/booking/domain"
	bookingservice "hauslet/internal/modules/booking/service"

	"github.com/google/uuid"
)

// BookingLoader batches booking fetches by ID using a time-window
// dataloader pattern.
type BookingLoader struct {
	svc bookingservice.BookingService

	mu     sync.Mutex
	batch  *bookingBatch
	cache  map[uuid.UUID]*bookingResult
	window time.Duration
}

type bookingResult struct {
	booking *domain.Booking
}

type bookingBatch struct {
	keys   []uuid.UUID
	done   chan struct{}
	result map[uuid.UUID]*domain.Booking
	err    error
}

const (
	defaultBookingWindow = 2 * time.Millisecond
	maxBookingBatchSize  = 200
)

func NewBookingLoader(svc bookingservice.BookingService) *BookingLoader {
	return &BookingLoader{
		svc:    svc,
		cache:  make(map[uuid.UUID]*bookingResult),
		window: defaultBookingWindow,
	}
}

// Load returns a booking by ID. Concurrent calls within the batching
// window are automatically grouped into a single database query.
func (l *BookingLoader) Load(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	l.mu.Lock()
	if cached, ok := l.cache[id]; ok {
		l.mu.Unlock()
		return cached.booking, nil
	}

	b := l.getCurrentBatch(ctx, id)
	l.mu.Unlock()

	<-b.done

	if b.err != nil {
		return nil, b.err
	}

	if booking, ok := b.result[id]; ok {
		return booking, nil
	}
	return nil, nil
}

// getCurrentBatch returns the current batch, creating one if needed.
// Must be called with l.mu held.
func (l *BookingLoader) getCurrentBatch(ctx context.Context, id uuid.UUID) *bookingBatch {
	if l.batch == nil {
		l.batch = &bookingBatch{
			keys: make([]uuid.UUID, 0, 32),
			done: make(chan struct{}),
		}
		go l.dispatchAfterWindow(ctx)
	}

	l.batch.keys = append(l.batch.keys, id)
	b := l.batch

	if len(l.batch.keys) >= maxBookingBatchSize {
		l.batch = nil
		go l.dispatchBatch(ctx, b)
	}

	return b
}

func (l *BookingLoader) dispatchAfterWindow(ctx context.Context) {
	time.Sleep(l.window)

	l.mu.Lock()
	b := l.batch
	l.batch = nil
	l.mu.Unlock()

	if b != nil {
		l.dispatchBatch(ctx, b)
	}
}

func (l *BookingLoader) dispatchBatch(ctx context.Context, b *bookingBatch) {
	defer close(b.done)

	seen := make(map[uuid.UUID]bool, len(b.keys))
	unique := make([]uuid.UUID, 0, len(b.keys))
	for _, id := range b.keys {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	fetched, err := l.svc.GetBookingsByIDs(ctx, unique)
	if err != nil {
		b.err = err
		return
	}

	b.result = make(map[uuid.UUID]*domain.Booking, len(fetched))
	for i := range fetched {
		b.result[fetched[i].ID] = fetched[i]
	}

	l.mu.Lock()
	for _, id := range unique {
		if booking, ok := b.result[id]; ok {
			l.cache[id] = &bookingResult{booking: booking}
		} else {
			l.cache[id] = &bookingResult{booking: nil}
		}
	}
	l.mu.Unlock()
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
		if cached, ok := l.cache[id]; ok {
			result[i] = cached.booking
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
		byID[fetched[i].ID] = fetched[i]
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i, idx := range missingIdx {
		id := missing[i]
		if booking, ok := byID[id]; ok {
			result[idx] = booking
			l.cache[id] = &bookingResult{booking: booking}
		} else {
			l.cache[id] = &bookingResult{booking: nil}
		}
	}

	return result, nil
}
