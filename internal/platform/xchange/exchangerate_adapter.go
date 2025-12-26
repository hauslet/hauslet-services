package xchange

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hauslet/internal/platform/redis"

	goredis "github.com/redis/go-redis/v9"
)

// ExchangeRateAdapter implements the XChange interface using exchangerate-api.com.
type ExchangeRateAdapter struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	cacheTTL   time.Duration
	cache      redis.RedisClient
}

// NewExchangeRateAdapter constructs an adapter with sane defaults.
func NewExchangeRateAdapter(apiKey, baseURL string, cache redis.RedisClient) *ExchangeRateAdapter {
	return &ExchangeRateAdapter{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cacheTTL: 30 * time.Minute,
		cache:    cache,
	}
}

// WithHTTPClient allows overriding the HTTP client (useful for tests).
func (a *ExchangeRateAdapter) WithHTTPClient(client *http.Client) *ExchangeRateAdapter {
	if client != nil {
		a.httpClient = client
	}
	return a
}

// ConvertCurrency converts an amount between two currencies using the remote rate.
func (a *ExchangeRateAdapter) ConvertCurrency(amount float64, fromCurrency, toCurrency string) (float64, error) {
	rate, err := a.GetExchangeRate(fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	return amount * rate, nil
}

// GetExchangeRate fetches the exchange rate between the provided currencies.
func (a *ExchangeRateAdapter) GetExchangeRate(fromCurrency, toCurrency string) (float64, error) {
	from := NormalizeCurrency(fromCurrency)
	to := NormalizeCurrency(toCurrency)

	if from == to {
		return 1, nil
	}

	cacheKey := fmt.Sprintf("fx:rate:%s:%s", from, to)
	if rate, ok := a.cachedRate(cacheKey); ok {
		return rate, nil
	}

	url := fmt.Sprintf("%s/%s/pair/%s/%s", a.baseURL, a.apiKey, from, to)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("fx: failed to build request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fx: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fx: upstream returned status %d", resp.StatusCode)
	}

	var pair pairResponse
	if err := json.NewDecoder(resp.Body).Decode(&pair); err != nil {
		return 0, fmt.Errorf("fx: failed to decode response: %w", err)
	}

	if pair.Result != "success" {
		if pair.ErrorType != "" {
			return 0, fmt.Errorf("fx: upstream error %s", pair.ErrorType)
		}
		return 0, fmt.Errorf("fx: upstream error")
	}

	if pair.ConversionRate <= 0 {
		return 0, fmt.Errorf("fx: invalid conversion rate %.4f", pair.ConversionRate)
	}

	a.storeRate(cacheKey, pair.ConversionRate)

	return pair.ConversionRate, nil
}

type pairResponse struct {
	Result         string  `json:"result"`
	ConversionRate float64 `json:"conversion_rate"`
	ErrorType      string  `json:"error-type"`
}

func (a *ExchangeRateAdapter) cachedRate(key string) (float64, bool) {
	if a.cache == nil || a.cacheTTL <= 0 {
		return 0, false
	}

	ctx := context.Background()
	val, err := a.cache.Get(ctx, key).Result()
	if err != nil {
		if err != goredis.Nil {
			return 0, false
		}
		return 0, false
	}

	rate, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false
	}

	return rate, true
}

func (a *ExchangeRateAdapter) storeRate(key string, rate float64) {
	if a.cache == nil || a.cacheTTL <= 0 {
		return
	}

	ctx := context.Background()
	a.cache.Set(ctx, key, rate, a.cacheTTL)
}
