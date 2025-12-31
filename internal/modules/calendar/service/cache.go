package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hauslet/internal/modules/calendar/domain"
	"time"

	"github.com/google/uuid"
)

const (
	eventCachePrefix        = "calendar:event:"
	availabilityCachePrefix = "calendar:availability:"

	eventCacheTTL        = 1 * time.Hour
	availabilityCacheTTL = 15 * time.Minute
)

// --- Cache Key Generators ---

func eventCacheKey(eventID uuid.UUID) string {
	return fmt.Sprintf("%s%s", eventCachePrefix, eventID)
}

func availabilityCacheKey(listingID uuid.UUID, startTime, endTime time.Time) string {
	return fmt.Sprintf("%s%s:%s:%s", availabilityCachePrefix, listingID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
}

// --- Cache Operations ---

func (s *CalendarServiceImpl) cacheEvent(ctx context.Context, event *domain.CalendarEvent) {
	if s.cache == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to marshal event for cache: %v", err)
		}
		return
	}

	key := eventCacheKey(event.ID)
	if err := s.cache.Set(ctx, key, data, eventCacheTTL); err != nil {
		if s.log != nil {
			s.log.Warn("failed to cache event: %v", err)
		}
	}
}

func (s *CalendarServiceImpl) cacheAvailability(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, result *domain.AvailabilityResult) {
	if s.cache == nil {
		return
	}

	data, err := json.Marshal(result)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to marshal availability for cache: %v", err)
		}
		return
	}

	key := availabilityCacheKey(listingID, startTime, endTime)
	if err := s.cache.Set(ctx, key, data, availabilityCacheTTL); err != nil {
		if s.log != nil {
			s.log.Warn("failed to cache availability: %v", err)
		}
	}
}

func (s *CalendarServiceImpl) getCachedValue(ctx context.Context, key string, dest any) (bool, error) {
	if s.cache == nil {
		return false, nil
	}

	data, err := s.cache.Get(ctx, key).Bytes()
	if err != nil {
		return false, nil
	}

	if data == nil {
		return false, nil
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, err
	}

	return true, nil
}

func (s *CalendarServiceImpl) invalidateEventCache(ctx context.Context, eventID uuid.UUID) {
	if s.cache == nil {
		return
	}

	key := eventCacheKey(eventID)
	if err := s.cache.Del(ctx, key); err != nil {
		if s.log != nil {
			s.log.Warn("failed to invalidate event cache: %v", err)
		}
	}
}

func (s *CalendarServiceImpl) invalidateAvailabilityCache(ctx context.Context, listingID uuid.UUID) {
	if s.cache == nil {
		return
	}

	// Invalidate all availability cache keys for this listing
	pattern := fmt.Sprintf("%s%s:*", availabilityCachePrefix, listingID)
	keys, err := s.cache.Keys(ctx, pattern).Result()
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to find availability cache keys: %v", err)
		}
		return
	}
	if len(keys) > 0 {
		if err := s.cache.Del(ctx, keys...).Err(); err != nil {
			if s.log != nil {
				s.log.Warn("failed to invalidate availability cache: %v", err)
			}
		}
	}
}
