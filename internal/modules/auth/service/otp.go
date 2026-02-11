package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/auth/domain"
	"strings"
	"time"
)

const (
	OTPExpiration             = 10 * time.Minute
	OTPKeyPrefix              = "otp:verify:"
	PasswordlessOTPKeyPrefix  = "otp:passwordless:"
	PasswordlessOTPExpiration = 10 * time.Minute
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// GenerateEmailOTP generates a 6-digit OTP code and stores it in Redis
func (s *AuthServiceImpl) GenerateEmailOTP(ctx context.Context, email string) (string, error) {
	// Generate a secure random 6-digit code
	code, err := generateSecureOTP()
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	normalizedEmail := normalizeEmail(email)

	// Store in Redis with 10-minute expiration
	key := OTPKeyPrefix + normalizedEmail
	err = s.redisClient.Set(ctx, key, code, OTPExpiration).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store OTP in Redis: %w", err)
	}

	s.log.Info("generated OTP", "email", normalizedEmail, "expires_in", OTPExpiration)
	return code, nil
}

// VerifyEmailOTP verifies the provided OTP code against the stored value
func (s *AuthServiceImpl) VerifyEmailOTP(ctx context.Context, email, code string) error {
	normalizedEmail := normalizeEmail(email)
	key := OTPKeyPrefix + normalizedEmail

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

	s.log.Info("OTP verified successfully", "email", normalizedEmail)
	return nil
}

// DeleteEmailOTP removes the OTP from Redis after successful verification
func (s *AuthServiceImpl) DeleteEmailOTP(ctx context.Context, email string) error {
	normalizedEmail := normalizeEmail(email)
	key := OTPKeyPrefix + normalizedEmail
	err := s.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete OTP from Redis: %w", err)
	}

	s.log.Info("deleted OTP", "email", normalizedEmail)
	return nil
}

// ResendVerificationEmail generates a new OTP and sends the welcome email.
// Returns an error if the user doesn't exist, is already verified, or email fails.
func (s *AuthServiceImpl) ResendVerificationEmail(ctx context.Context, email string) error {
	// Get user by email
	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		// Don't reveal if user exists - caller should return generic success
		return fmt.Errorf("user not found")
	}

	// Check if email is already verified
	identities, err := s.repository.ListUserIdentitiesByUserID(ctx, user.ID.String())
	if err != nil {
		return fmt.Errorf("failed to list user identities: %w", err)
	}

	for _, identity := range identities {
		if identity.Provider == "password" && identity.Email == email && identity.EmailVerified {
			return fmt.Errorf("email is already verified")
		}
	}

	// Generate new OTP (overwrites old one in Redis)
	otpCode, err := s.GenerateEmailOTP(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Send welcome email with new OTP
	if err := s.notifier.SendWelcomeEmail(ctx, email, user.Name, otpCode); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	s.log.Info("resent verification email", "email", email)
	return nil
}

// VerifyAndActivateEmail verifies the OTP code and activates the user's email identity.
// This is a composite operation that:
// 1. Verifies the OTP code
// 2. Marks the email identity as verified
// 3. Cleans up the OTP from Redis
func (s *AuthServiceImpl) VerifyAndActivateEmail(ctx context.Context, email, code string) error {
	// Verify OTP
	if err := s.VerifyEmailOTP(ctx, email, code); err != nil {
		return fmt.Errorf("invalid or expired verification code")
	}

	// Update email verified status in UserIdentity
	if err := s.UpdateIdentityVerified(ctx, email); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	// Delete OTP from Redis (cleanup - don't fail if this errors)
	if err := s.DeleteEmailOTP(ctx, email); err != nil {
		s.log.Warn("failed to delete OTP after verification", "email", email, "error", err)
		// Continue - OTP will expire anyway
	}

	s.log.Info("email verified successfully", "email", email)
	return nil
}

// ============================================================================
// Passwordless Login OTP (separate from registration verification)
// ============================================================================

// GeneratePasswordlessOTP generates a 6-digit OTP for passwordless login and stores it in Redis.
func (s *AuthServiceImpl) GeneratePasswordlessOTP(ctx context.Context, email string) (string, error) {
	code, err := generateSecureOTP()
	if err != nil {
		return "", fmt.Errorf("failed to generate passwordless OTP: %w", err)
	}

	normalizedEmail := normalizeEmail(email)
	key := PasswordlessOTPKeyPrefix + normalizedEmail
	if err := s.redisClient.Set(ctx, key, code, PasswordlessOTPExpiration).Err(); err != nil {
		return "", fmt.Errorf("failed to store passwordless OTP in Redis: %w", err)
	}

	s.log.Info("generated passwordless OTP", "email", normalizedEmail, "expires_in", PasswordlessOTPExpiration)
	return code, nil
}

// VerifyPasswordlessOTP verifies the provided OTP code for passwordless login.
func (s *AuthServiceImpl) VerifyPasswordlessOTP(ctx context.Context, email, code string) error {
	normalizedEmail := normalizeEmail(email)
	key := PasswordlessOTPKeyPrefix + normalizedEmail

	storedCode, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("OTP not found or expired")
	}

	if storedCode != code {
		return fmt.Errorf("invalid OTP code")
	}

	s.log.Info("passwordless OTP verified", "email", normalizedEmail)
	return nil
}

// DeletePasswordlessOTP removes the passwordless OTP from Redis after successful verification.
func (s *AuthServiceImpl) DeletePasswordlessOTP(ctx context.Context, email string) error {
	normalizedEmail := normalizeEmail(email)
	key := PasswordlessOTPKeyPrefix + normalizedEmail
	if err := s.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete passwordless OTP: %w", err)
	}

	s.log.Info("deleted passwordless OTP", "email", normalizedEmail)
	return nil
}

// VerifyPasswordlessLogin verifies the passwordless OTP and returns the authenticated user.
// This is a composite operation that:
// 1. Verifies the OTP code
// 2. Looks up the user by email
// 3. Checks if the user is active
// 4. Cleans up the OTP from Redis
func (s *AuthServiceImpl) VerifyPasswordlessLogin(ctx context.Context, email, code string) (*domain.User, error) {
	// Verify OTP code
	if err := s.VerifyPasswordlessOTP(ctx, email, code); err != nil {
		return nil, fmt.Errorf("invalid or expired verification code")
	}

	// Look up user by email
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is deactivated")
	}

	// Delete OTP after successful verification (cleanup - don't fail if this errors)
	if err := s.DeletePasswordlessOTP(ctx, email); err != nil {
		s.log.Warn("failed to delete passwordless OTP after verification", "email", email, "error", err)
		// Continue - OTP will expire anyway
	}

	s.log.Info("passwordless login verified", "email", email, "user_id", user.ID)
	return user, nil
}
