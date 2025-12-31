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

	s.log.Info(" generated OTP for %s (expires in %v)", email, OTPExpiration)
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

	s.log.Info(" OTP verified successfully for", "email", email)
	return nil
}

// DeleteEmailOTP removes the OTP from Redis after successful verification
func (s *AuthServiceImpl) DeleteEmailOTP(ctx context.Context, email string) error {
	key := OTPKeyPrefix + email
	err := s.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete OTP from Redis: %w", err)
	}

	s.log.Info(" deleted OTP for", "email", email)
	return nil
}

// generateSecureOTP generates a cryptographically secure 6-digit OTP
func generateSecureOTP() (string, error) {
	// Generate a random number between 100000 and 999999
	min := int64(100000)
	max := int64(999999)

	n, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return "", err
	}

	code := n.Int64() + min
	return fmt.Sprintf("%06d", code), nil
}
