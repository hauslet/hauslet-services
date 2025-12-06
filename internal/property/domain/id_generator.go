package domain

import (
	"crypto/rand"
	"fmt"
)

const (
	publicIDPrefix = "H"
	publicIDLength = 8
	// Exclude confusing characters: 0, O, 1, I
	publicIDChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// GeneratePropertyPublicID generates a random public ID in format H + 7 alphanumeric chars.
// Example: H18U3AD8
func GeneratePropertyPublicID() (string, error) {
	b := make([]byte, publicIDLength-1) // -1 because first char is always 'H'
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	result := make([]byte, publicIDLength)
	result[0] = publicIDPrefix[0]

	for i := 1; i < publicIDLength; i++ {
		result[i] = publicIDChars[int(b[i-1])%len(publicIDChars)]
	}

	return string(result), nil
}
