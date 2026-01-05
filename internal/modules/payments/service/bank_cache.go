package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/platform/payment"

	"github.com/redis/go-redis/v9"
)

const bankListCacheTTL = 24 * time.Hour

type bankNameCache map[string]string

func (s *PaymentServiceImpl) bankCacheEnabled() bool {
	return s != nil && s.cache != nil
}

func bankListCacheKey(currency payment.Currency, country string) string {
	cur := strings.ToLower(strings.TrimSpace(currency.String()))
	if cur == "" {
		cur = "unknown"
	}
	region := strings.ToLower(strings.TrimSpace(country))
	if region == "" {
		region = "all"
	}
	return fmt.Sprintf("payments:banks:%s:%s", cur, region)
}

func normalizeBankCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func buildBankNameCache(banks []payment.Bank) bankNameCache {
	names := make(bankNameCache, len(banks))
	for _, bank := range banks {
		if bank.IsDeleted || !bank.Active {
			continue
		}
		code := normalizeBankCode(bank.Code)
		if code == "" || bank.Name == "" {
			continue
		}
		names[code] = bank.Name
	}
	return names
}

func (s *PaymentServiceImpl) getCachedBankNames(ctx context.Context, key string) (bankNameCache, bool) {
	if !s.bankCacheEnabled() {
		return nil, false
	}
	val, err := s.cache.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil && s.log != nil {
			s.log.Warn("bank cache get failed", "key", key, "error", err)
		}
		return nil, false
	}

	var names bankNameCache
	if err := json.Unmarshal([]byte(val), &names); err != nil {
		_ = s.cache.Del(ctx, key).Err()
		if s.log != nil {
			s.log.Warn("bank cache decode failed", "key", key, "error", err)
		}
		return nil, false
	}
	if len(names) == 0 {
		return nil, false
	}
	return names, true
}

func (s *PaymentServiceImpl) setCachedBankNames(ctx context.Context, key string, names bankNameCache) {
	if !s.bankCacheEnabled() || len(names) == 0 {
		return
	}
	bytes, err := json.Marshal(names)
	if err != nil {
		if s.log != nil {
			s.log.Warn("bank cache marshal failed", "key", key, "error", err)
		}
		return
	}
	if err := s.cache.Set(ctx, key, bytes, bankListCacheTTL).Err(); err != nil && s.log != nil {
		s.log.Warn("bank cache set failed", "key", key, "error", err)
	}
}
