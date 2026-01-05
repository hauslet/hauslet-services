package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const (
	OTPLength     = 6
	OTPExpiration = 10 * time.Minute
	OTPKeyPrefix  = "otp:verify:"
)

// GenerateEmailOTP generates a 6-digit OTP code and stores it in Redis
func (s *AuthServiceImpl) GenerateEmailOTP(ctx context.Context, email string) (string, error) {
	// Generate a secure random 6-digit code
	code, err := generateSecureOTP()
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Store in Redis with 10-minute expiration
	key := OTPKeyPrefix + email
	err = s.redisClient.Set(ctx, key, code, OTPExpiration).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store OTP in Redis: %w", err)
	}

	s.log.Info("generated OTP", "email", email, "expires_in", OTPExpiration)
	return code, nil
}

// VerifyEmailOTP verifies the provided OTP code against the stored value
func (s *AuthServiceImpl) VerifyEmailOTP(ctx context.Context, email, code string) error {
	key := OTPKeyPrefix + email

	// Get stored OTP from Redis
	storedCode, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		// Key not found or expired
		return fmt.Errorf("OTP not found or expired")
	}

	// Compare codes
	if storedCode != code {
		return fmt.Errorf("invalid OTP code")
	}

	s.log.Info("OTP verified successfully", "email", email)
	return nil
}

// DeleteEmailOTP removes the OTP from Redis after successful verification
func (s *AuthServiceImpl) DeleteEmailOTP(ctx context.Context, email string) error {
	key := OTPKeyPrefix + email
	err := s.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete OTP from Redis: %w", err)
	}

	s.log.Info("deleted OTP", "email", email)
	return nil
}

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

	// Format as 6 digits with leading zeros (e.g., "004123")
	return fmt.Sprintf("%06d", n.Int64()), nil
}
