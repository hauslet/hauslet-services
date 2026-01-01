package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/modules/wishlist/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	wishlistCacheTTL     = 5 * time.Minute
	wishlistItemCacheTTL = 5 * time.Minute
	userListsCacheTTL    = 3 * time.Minute
)

func (s *WishlistServiceImpl) cacheEnabled() bool {
	return s != nil && s.cache != nil
}

func (s *WishlistServiceImpl) getCachedValue(ctx context.Context, key string, dest any) (bool, error) {
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

func (s *WishlistServiceImpl) setCachedValue(ctx context.Context, key string, ttl time.Duration, value any) {
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

func (s *WishlistServiceImpl) invalidateCache(ctx context.Context, keys ...string) {
	if !s.cacheEnabled() || len(keys) == 0 {
		return
	}
	_ = s.cache.Del(ctx, keys...).Err()
}

// Cache key generators

func wishlistCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("wishlist:%s", id.String())
}

func wishlistItemsCacheKey(wishlistID uuid.UUID, limit, offset int) string {
	return fmt.Sprintf("wishlist:%s:items:limit:%d:offset:%d", wishlistID.String(), limit, offset)
}

func userWishlistsCacheKey(userID uuid.UUID, limit, offset int) string {
	return fmt.Sprintf("user:%s:wishlists:limit:%d:offset:%d", userID.String(), limit, offset)
}

func wishlistItemCountCacheKey(wishlistID uuid.UUID) string {
	return fmt.Sprintf("wishlist:%s:count", wishlistID.String())
}

func listingInWishlistCacheKey(wishlistID, listingID uuid.UUID) string {
	return fmt.Sprintf("wishlist:%s:listing:%s:exists", wishlistID.String(), listingID.String())
}

// Cache setters

func (s *WishlistServiceImpl) cacheWishlist(ctx context.Context, w *domain.Wishlist) {
	if w == nil {
		return
	}
	s.setCachedValue(ctx, wishlistCacheKey(w.ID), wishlistCacheTTL, w)
}

func (s *WishlistServiceImpl) cacheWishlistItems(ctx context.Context, wishlistID uuid.UUID, items []*domain.WishlistItem, limit, offset int) {
	if items == nil {
		return
	}
	s.setCachedValue(ctx, wishlistItemsCacheKey(wishlistID, limit, offset), wishlistItemCacheTTL, items)
}

func (s *WishlistServiceImpl) cacheUserWishlists(ctx context.Context, userID uuid.UUID, wishlists []*domain.Wishlist, limit, offset int) {
	if wishlists == nil {
		return
	}
	s.setCachedValue(ctx, userWishlistsCacheKey(userID, limit, offset), userListsCacheTTL, wishlists)
}

func (s *WishlistServiceImpl) cacheItemCount(ctx context.Context, wishlistID uuid.UUID, count int64) {
	s.setCachedValue(ctx, wishlistItemCountCacheKey(wishlistID), wishlistCacheTTL, count)
}

func (s *WishlistServiceImpl) cacheListingExists(ctx context.Context, wishlistID, listingID uuid.UUID, exists bool) {
	s.setCachedValue(ctx, listingInWishlistCacheKey(wishlistID, listingID), wishlistCacheTTL, exists)
}

// Cache invalidators

func (s *WishlistServiceImpl) invalidateWishlistCache(ctx context.Context, id uuid.UUID) {
	if id == uuid.Nil {
		return
	}
	s.invalidateCache(ctx, wishlistCacheKey(id))
}

func (s *WishlistServiceImpl) invalidateWishlistItemsCache(ctx context.Context, wishlistID uuid.UUID) {
	if wishlistID == uuid.Nil {
		return
	}
	// Invalidate all pagination variants by pattern
	// Note: This is a simple approach. For production, consider using Redis SCAN or key patterns
	s.invalidateCache(ctx,
		wishlistItemCountCacheKey(wishlistID),
		// Invalidate common pagination variants
		wishlistItemsCacheKey(wishlistID, 0, 0),
		wishlistItemsCacheKey(wishlistID, 10, 0),
		wishlistItemsCacheKey(wishlistID, 20, 0),
		wishlistItemsCacheKey(wishlistID, 50, 0),
	)
}

func (s *WishlistServiceImpl) invalidateUserWishlistsCache(ctx context.Context, userID uuid.UUID) {
	if userID == uuid.Nil {
		return
	}
	// Invalidate common pagination variants
	s.invalidateCache(ctx,
		userWishlistsCacheKey(userID, 0, 0),
		userWishlistsCacheKey(userID, 10, 0),
		userWishlistsCacheKey(userID, 20, 0),
		userWishlistsCacheKey(userID, 50, 0),
	)
}

func (s *WishlistServiceImpl) invalidateListingExistsCache(ctx context.Context, wishlistID, listingID uuid.UUID) {
	if wishlistID == uuid.Nil || listingID == uuid.Nil {
		return
	}
	s.invalidateCache(ctx, listingInWishlistCacheKey(wishlistID, listingID))
}

// Composite invalidation for write operations

func (s *WishlistServiceImpl) invalidateWishlistAndRelated(ctx context.Context, wishlistID, userID uuid.UUID) {
	s.invalidateWishlistCache(ctx, wishlistID)
	s.invalidateWishlistItemsCache(ctx, wishlistID)
	s.invalidateUserWishlistsCache(ctx, userID)
}
