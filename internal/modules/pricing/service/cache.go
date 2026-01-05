package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hauslet/internal/modules/pricing/domain"
	"time"

	"github.com/google/uuid"
)

const (
	basePriceCachePrefix = "pricing:base:"
	breakdownCachePrefix = "pricing:breakdown:"

	basePriceCacheTTL = 24 * time.Hour
	breakdownCacheTTL = 1 * time.Hour
)

// --- Cache Key Generators ---

func basePriceCacheKey(listingID uuid.UUID) string {
	return fmt.Sprintf("%s%s", basePriceCachePrefix, listingID)
}

func breakdownCacheKey(listingID uuid.UUID, checkIn, checkOut time.Time) string {
	return fmt.Sprintf("%s%s:%s:%s", breakdownCachePrefix, listingID, checkIn.Format("2006-01-02"), checkOut.Format("2006-01-02"))
}

// --- Cache Operations ---

func (s *PricingServiceImpl) cacheBasePrice(ctx context.Context, listingID uuid.UUID, price float64, currency string) {
	if s.cache == nil {
		return
	}

	data := struct {
		Price    float64
		Currency string
	}{
		Price:    price,
		Currency: currency,
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to marshal base price for cache", "error", err)
		}
		return
	}

	key := basePriceCacheKey(listingID)
	if err := s.cache.Set(ctx, key, bytes, basePriceCacheTTL).Err(); err != nil {
		if s.log != nil {
			s.log.Warn("failed to cache base price", "error", err)
		}
	}
}

func (s *PricingServiceImpl) cachePriceBreakdown(ctx context.Context, breakdown *domain.PriceBreakdown) {
	if s.cache == nil {
		return
	}

	data, err := json.Marshal(breakdown)
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to marshal price breakdown for cache", "error", err)
		}
		return
	}

	key := breakdownCacheKey(breakdown.ListingID, breakdown.CheckIn, breakdown.CheckOut)
	if err := s.cache.Set(ctx, key, data, breakdownCacheTTL).Err(); err != nil {
		if s.log != nil {
			s.log.Warn("failed to cache price breakdown", "error", err)
		}
	}
}

func (s *PricingServiceImpl) getCachedValue(ctx context.Context, key string, dest any) (bool, error) {
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

func (s *PricingServiceImpl) invalidatePriceCaches(ctx context.Context, listingID uuid.UUID) {
	if s.cache == nil {
		return
	}

	// Invalidate base price
	baseKey := basePriceCacheKey(listingID)
	if err := s.cache.Del(ctx, baseKey).Err(); err != nil {
		if s.log != nil {
			s.log.Warn("failed to invalidate base price cache", "error", err)
		}
	}

	// Invalidate all breakdowns for this listing
	pattern := fmt.Sprintf("%s%s:*", breakdownCachePrefix, listingID)
	keys, err := s.cache.Keys(ctx, pattern).Result()
	if err != nil {
		if s.log != nil {
			s.log.Warn("failed to find breakdown cache keys", "error", err)
		}
		return
	}
	if len(keys) > 0 {
		if err := s.cache.Del(ctx, keys...).Err(); err != nil {
			if s.log != nil {
				s.log.Warn("failed to invalidate breakdown cache", "error", err)
			}
		}
	}
}
