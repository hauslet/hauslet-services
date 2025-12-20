package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/modules/property/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	propertyCacheTTL = 5 * time.Minute
	listingCacheTTL  = 5 * time.Minute
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
		_ = s.cache.Del(ctx, key).Err() // drop corrupt cache
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
			s.log.Logf("WARN cache marshal failed for key=%s: %v", key, err)
		}
		return
	}
	if err := s.cache.Set(ctx, key, bytes, ttl).Err(); err != nil && s.log != nil {
		s.log.Logf("WARN cache set failed for key=%s: %v", key, err)
	}
}

func (s *ServiceImpl) invalidateCache(ctx context.Context, keys ...string) {
	if !s.cacheEnabled() || len(keys) == 0 {
		return
	}
	_ = s.cache.Del(ctx, keys...).Err()
}

func propertyCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("property:%s", id.String())
}

func listingIDCacheKey(id uuid.UUID, preloadMedia bool) string {
	return fmt.Sprintf("listing:%s:media:%t", id.String(), preloadMedia)
}

func listingSlugCacheKey(slug string, preloadMedia bool) string {
	return fmt.Sprintf("listing:slug:%s:media:%t", slug, preloadMedia)
}

func listingPublicIDCacheKey(publicID string, preloadMedia bool) string {
	return fmt.Sprintf("listing:public:%s:media:%t", publicID, preloadMedia)
}

func (s *ServiceImpl) cacheProperty(ctx context.Context, p *domain.Property) {
	if p == nil {
		return
	}
	s.setCachedValue(ctx, propertyCacheKey(p.ID), propertyCacheTTL, p)
}

func (s *ServiceImpl) cacheListing(ctx context.Context, l *domain.Listing, preloadMedia bool, slug string, publicID string) {
	if l == nil {
		return
	}

	s.setCachedValue(ctx, listingIDCacheKey(l.ID, preloadMedia), listingCacheTTL, l)
	if slug != "" {
		s.setCachedValue(ctx, listingSlugCacheKey(slug, preloadMedia), listingCacheTTL, l)
	}
	if publicID != "" {
		s.setCachedValue(ctx, listingPublicIDCacheKey(publicID, preloadMedia), listingCacheTTL, l)
	}
}

func (s *ServiceImpl) invalidatePropertyCache(ctx context.Context, id uuid.UUID) {
	if id == uuid.Nil {
		return
	}
	s.invalidateCache(ctx, propertyCacheKey(id))
}

func (s *ServiceImpl) invalidateListingCache(ctx context.Context, id uuid.UUID, slug string, publicID string) {
	if id == uuid.Nil && slug == "" && publicID == "" {
		return
	}

	keys := make([]string, 0, 6)
	if id != uuid.Nil {
		keys = append(keys, listingIDCacheKey(id, true), listingIDCacheKey(id, false))
	}
	if slug != "" {
		keys = append(keys, listingSlugCacheKey(slug, true), listingSlugCacheKey(slug, false))
	}
	if publicID != "" {
		keys = append(keys, listingPublicIDCacheKey(publicID, true), listingPublicIDCacheKey(publicID, false))
	}

	s.invalidateCache(ctx, keys...)
}

// getPropertyPublicID fetches the property's public ID for cache invalidation.
func (s *ServiceImpl) getPropertyPublicID(ctx context.Context, propertyID uuid.UUID) string {
	if propertyID == uuid.Nil {
		return ""
	}
	p, err := s.repo.GetPropertyByID(ctx, propertyID)
	if err != nil || p == nil {
		return ""
	}
	return p.PublicID
}
