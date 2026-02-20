package loaders

import (
	"context"
	"sync"
	"time"

	"hauslet/internal/modules/profile/domain"
	profileservice "hauslet/internal/modules/profile/service"
)

// ProfileLoader batches profile fetches by userID using a time-window
// dataloader pattern.
type ProfileLoader struct {
	svc profileservice.ProfileService

	mu     sync.Mutex
	batch  *profileBatch
	cache  map[string]*profileResult
	window time.Duration
}

type profileResult struct {
	profile *domain.Profile
}

type profileBatch struct {
	keys   []string
	done   chan struct{}
	result map[string]*domain.Profile
	err    error
}

const (
	defaultProfileWindow = 2 * time.Millisecond
	maxProfileBatchSize  = 200
)

func NewProfileLoader(svc profileservice.ProfileService) *ProfileLoader {
	return &ProfileLoader{
		svc:    svc,
		cache:  make(map[string]*profileResult),
		window: defaultProfileWindow,
	}
}

// Load returns a profile by userID. Concurrent calls within the batching
// window are automatically grouped into a single database query.
func (l *ProfileLoader) Load(ctx context.Context, userID string) (*domain.Profile, error) {
	l.mu.Lock()
	if cached, ok := l.cache[userID]; ok {
		l.mu.Unlock()
		return cached.profile, nil
	}

	b := l.getCurrentBatch(ctx, userID)
	l.mu.Unlock()

	<-b.done

	if b.err != nil {
		return nil, b.err
	}

	if p, ok := b.result[userID]; ok {
		return p, nil
	}
	return nil, nil
}

// getCurrentBatch returns the current batch, creating one if needed.
// Must be called with l.mu held.
func (l *ProfileLoader) getCurrentBatch(ctx context.Context, id string) *profileBatch {
	if l.batch == nil {
		l.batch = &profileBatch{
			keys: make([]string, 0, 32),
			done: make(chan struct{}),
		}
		go l.dispatchAfterWindow(ctx)
	}

	l.batch.keys = append(l.batch.keys, id)
	b := l.batch

	if len(l.batch.keys) >= maxProfileBatchSize {
		l.batch = nil
		go l.dispatchBatch(ctx, b)
	}

	return b
}

func (l *ProfileLoader) dispatchAfterWindow(ctx context.Context) {
	time.Sleep(l.window)

	l.mu.Lock()
	b := l.batch
	l.batch = nil
	l.mu.Unlock()

	if b != nil {
		l.dispatchBatch(ctx, b)
	}
}

func (l *ProfileLoader) dispatchBatch(ctx context.Context, b *profileBatch) {
	defer close(b.done)

	seen := make(map[string]bool, len(b.keys))
	unique := make([]string, 0, len(b.keys))
	for _, id := range b.keys {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	fetched, err := l.svc.GetProfilesByUserIDs(ctx, unique)
	if err != nil {
		b.err = err
		return
	}

	b.result = make(map[string]*domain.Profile, len(fetched))
	for i := range fetched {
		pCopy := fetched[i]
		b.result[pCopy.UserID] = &pCopy
	}

	l.mu.Lock()
	for _, id := range unique {
		if p, ok := b.result[id]; ok {
			l.cache[id] = &profileResult{profile: p}
		} else {
			l.cache[id] = &profileResult{profile: nil}
		}
	}
	l.mu.Unlock()
}

// LoadMany fetches profiles for the provided user IDs using a single service call.
// This bypasses the time-window since the caller already has all IDs.
func (l *ProfileLoader) LoadMany(ctx context.Context, userIDs []string) ([]*domain.Profile, error) {
	if len(userIDs) == 0 {
		return []*domain.Profile{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.Profile, len(userIDs))
	missing := make([]string, 0)
	missingIdx := make([]int, 0)

	for i, id := range userIDs {
		if cached, ok := l.cache[id]; ok {
			result[i] = cached.profile
		} else {
			missing = append(missing, id)
			missingIdx = append(missingIdx, i)
		}
	}
	l.mu.Unlock()

	if len(missing) == 0 {
		return result, nil
	}

	fetched, err := l.svc.GetProfilesByUserIDs(ctx, missing)
	if err != nil {
		return nil, err
	}

	byUserID := make(map[string]*domain.Profile, len(fetched))
	for i := range fetched {
		pCopy := fetched[i]
		byUserID[pCopy.UserID] = &pCopy
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for i, idx := range missingIdx {
		id := missing[i]
		if p, ok := byUserID[id]; ok {
			result[idx] = p
			l.cache[id] = &profileResult{profile: p}
		} else {
			l.cache[id] = &profileResult{profile: nil}
		}
	}

	return result, nil
}
