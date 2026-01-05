package utils

import (
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min+1)
}

func RandomInt64(min, max int64) int64 {
	if min >= max {
		return min
	}
	return min + rand.Int63n(max-min+1)
}

func RandomBool() bool {
	return rand.Intn(2) == 1
}

func RandomBoolWithProbability(probability float64) bool {
	return rand.Float64() < probability
}

func RandomChoice[T any](slice []T) T {
	if len(slice) == 0 {
		var zero T
		return zero
	}
	return slice[rand.Intn(len(slice))]
}

func RandomUniqueChoices[T any](slice []T, n int) []T {
	if n >= len(slice) {
		result := make([]T, len(slice))
		copy(result, slice)
		Shuffle(result)
		return result
	}
	result := make([]T, 0, n)
	used := make(map[int]bool)
	for len(result) < n {
		idx := rand.Intn(len(slice))
		if !used[idx] {
			used[idx] = true
			result = append(result, slice[idx])
		}
	}
	return result
}

func Shuffle[T any](slice []T) {
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func RandomDigits(length int) string {
	const digits = "0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = digits[rand.Intn(len(digits))]
	}
	return string(b)
}

func RandomDateInRange(minDays, maxDays int) time.Time {
	days := RandomInt(minDays, maxDays)
	return time.Now().AddDate(0, 0, days)
}

func RandomPastDate(maxDaysAgo int) time.Time {
	return RandomDateInRange(-maxDaysAgo, 0)
}

func RandomEmail(name string) string {
	providers := []string{"gmail.com", "yahoo.com", "outlook.com"}
	provider := RandomChoice(providers)
	clean := strings.ToLower(strings.ReplaceAll(name, " ", "."))
	suffix := RandomDigits(2)
	return clean + suffix + "@" + provider
}

func RandomPhoneNG() string {
	prefixes := []string{"803", "806", "810", "813", "816", "903", "906", "912"}
	prefix := RandomChoice(prefixes)
	suffix := RandomDigits(7)
	return "+234" + prefix + suffix
}
