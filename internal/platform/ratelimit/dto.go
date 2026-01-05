package ratelimit

import (
	"fmt"
	"time"
)

// KeyType represents the type of rate limit key
type KeyType string

const (
	KeyTypeUser    KeyType = "user"
	KeyTypeIP      KeyType = "ip"
	KeyTypePhone   KeyType = "phone"
	KeyTypeCountry KeyType = "country"
	KeyTypeEmail   KeyType = "email"
	KeyTypeListingEmail KeyType = "listing_email"
	KeyTypeListingIP    KeyType = "listing_ip"
)

// String returns the string representation
func (k KeyType) String() string {
	return string(k)
}

// LimitKey represents a rate limit key with its configuration
type LimitKey struct {
	Type   KeyType       // Type of key (user, ip, phone, country)
	Value  string        // Actual value (user_id, ip_address, phone_number, country_code)
	Limit  int64         // Maximum allowed requests
	Window time.Duration // Time window
}

// RedisKey returns the Redis key for this limit
func (k *LimitKey) RedisKey() string {
	return fmt.Sprintf("ratelimit:%s:%s", k.Type, k.Value)
}

// CheckResult represents the result of a rate limit check
type CheckResult struct {
	Allowed   bool          // Whether the request is allowed
	Key       LimitKey      // The rate limit key checked
	Current   int64         // Current request count
	Limit     int64         // Maximum allowed
	Remaining int64         // Remaining requests
	RetryAt   *time.Time    // When the limit resets (if rate limited)
	Window    time.Duration // Time window
}

// IsRateLimited returns true if the request is rate limited
func (r *CheckResult) IsRateLimited() bool {
	return !r.Allowed
}
