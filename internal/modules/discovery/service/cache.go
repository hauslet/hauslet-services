package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"hauslet/internal/modules/discovery/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	shortletPreviewCacheTTL = 15 * time.Minute
	homeFeedSectionCacheTTL = 2 * time.Minute
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

func homeFeedSectionCacheKey(
	sectionType domain.FeedSectionType,
	options FeedOptions,
	userID *uuid.UUID,
) string {
	userKey := "anon"
	if userID != nil {
		userKey = userID.String()
	}

	return fmt.Sprintf(
		"discovery:home:section:v1:%s:limit:%d:city:%s:state:%s:loc:%s:user:%s",
		sectionType.String(),
		options.Limit,
		normalizeCacheText(options.City),
		normalizeCacheText(options.State),
		locationCacheKey(options.Location),
		userKey,
	)
}

func normalizeCacheText(value *string) string {
	if value == nil {
		return "_"
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return "_"
	}
	return strings.ToLower(trimmed)
}

func locationCacheKey(location *LocationFilter) string {
	if location == nil {
		return "_"
	}

	return strings.Join([]string{
		strconv.FormatFloat(location.Latitude, 'f', 6, 64),
		strconv.FormatFloat(location.Longitude, 'f', 6, 64),
		strconv.FormatFloat(location.RadiusKm, 'f', 3, 64),
	}, ",")
}
