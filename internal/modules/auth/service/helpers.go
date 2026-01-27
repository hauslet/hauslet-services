package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	OTPLength        = 6
	backupCodeLength = 8
	charset          = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// generateSecureOTP generates a cryptographically secure 6-digit OTP
func generateSecureOTP() (string, error) {
	// Define the maximum value (exclusive) for a 6-digit number: 1,000,000
	// This gives us a range of 0 to 999999
	max := big.NewInt(1000000)

	// Generate a cryptographically secure random integer
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// In production, this usually indicates a system-level issue (entropy exhaustion)
		return "", fmt.Errorf("failed to generate secure random number: %w", err)
	}

	// Format as OTPLength digits with leading zeros (e.g., "004123")
	return fmt.Sprintf("%0*d", OTPLength, n.Int64()), nil
}

// generateBackupCodes now returns an error and handles hashing failures
func generateBackupCodes(count int) ([]string, []string, error) {
	codes := make([]string, count)
	hashes := make([]string, count)

	for i := 0; i < count; i++ {
		code, err := generateRandomCode(backupCodeLength)
		if err != nil {
			return nil, nil, err
		}

		codes[i] = code
		// Using a slightly lower cost for backup codes is common,
		// but DefaultCost (10) is a solid balance.
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hash code: %w", err)
		}
		hashes[i] = string(hash)
	}

	return codes, hashes, nil
}

// generateRandomCode uses a more robust method to avoid bias
func generateRandomCode(length int) (string, error) {
	var sb strings.Builder
	sb.Grow(length)

	max := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("entropy failure: %w", err)
		}
		sb.WriteByte(charset[idx.Int64()])
	}

	return sb.String(), nil
}
