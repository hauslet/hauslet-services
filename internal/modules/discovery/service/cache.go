package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	shortletPreviewCacheTTL = 15 * time.Minute
)

func (s *ServiceImpl) cacheEnabled() bool {
	return s != nil && s.cache != nil
}

func (s *ServiceImpl) getCachedValue(ctx context.Context, key string, dest any) (bool, error) {
	if !s.cacheEnabled() {
		return false, nil
	}

	val, err := s.cache.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		_ = s.cache.Del(ctx, key).Err()
		return false, nil
	}

	return true, nil
}

func (s *ServiceImpl) setCachedValue(ctx context.Context, key string, ttl time.Duration, value any) {
	if !s.cacheEnabled() {
		return
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		if s.log != nil {
			s.log.Warn("cache marshal failed", "key", key, "error", err)
		}
		return
	}

	if err := s.cache.Set(ctx, key, bytes, ttl).Err(); err != nil && s.log != nil {
		s.log.Warn("cache set failed", "key", key, "error", err)
	}
}

func shortletPreviewCacheKey(
	listingID uuid.UUID,
	guestCount, nights int,
	startDate time.Time,
) string {
	return fmt.Sprintf(
		"discovery:home:shortlet_preview:%s:g:%d:n:%d:start:%s",
		listingID.String(),
		guestCount,
		nights,
		startDate.Format("2006-01-02"),
	)
}
