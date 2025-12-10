package loaders

import (
	"context"
	"sync"

	"hauslet/internal/modules/profile/domain"
	profileservice "hauslet/internal/modules/profile/service"
)

// ProfileLoader batches profile fetches by userID for a single request.
type ProfileLoader struct {
	svc   profileservice.ProfileService
	mu    sync.Mutex
	cache map[string]*domain.Profile
}

func NewProfileLoader(svc profileservice.ProfileService) *ProfileLoader {
	return &ProfileLoader{
		svc:   svc,
		cache: make(map[string]*domain.Profile),
	}
}

// Load returns a profile by userID, using cached/batched lookups within the request.
func (l *ProfileLoader) Load(ctx context.Context, userID string) (*domain.Profile, error) {
	l.mu.Lock()
	if p, ok := l.cache[userID]; ok {
		l.mu.Unlock()
		return p, nil
	}
	l.mu.Unlock()

	profiles, err := l.LoadMany(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	if len(profiles) > 0 {
		return profiles[0], nil
	}
	return nil, nil
}

// LoadMany fetches profiles for the provided user IDs using a single service call.
func (l *ProfileLoader) LoadMany(ctx context.Context, userIDs []string) ([]*domain.Profile, error) {
	if len(userIDs) == 0 {
		return []*domain.Profile{}, nil
	}

	l.mu.Lock()
	result := make([]*domain.Profile, len(userIDs))
	missing := make([]string, 0)
	missingIdx := make([]int, 0)

	for i, id := range userIDs {
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
	for uid, p := range byUserID {
		l.cache[uid] = p
	}
	for i, idx := range missingIdx {
		if p, ok := byUserID[missing[i]]; ok {
			result[idx] = p
		}
	}
	l.mu.Unlock()

	return result, nil
}
