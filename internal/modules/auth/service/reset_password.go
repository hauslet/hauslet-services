package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hauslet/internal/modules/auth/repository/schema"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	resetTokenPrefix = "reset:password:"
	resetTokenTTL    = 15 * time.Minute
)

// RequestPasswordReset issues a reset token and emails it if the user exists.
func (s *AuthServiceImpl) RequestPasswordReset(ctx context.Context, email string) error {
	// Do not leak existence. Fetch user; if missing, return nil after short path.
	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil
	}

	token, err := generateSecureOTP()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	key := resetTokenPrefix + email
	if err := s.redisClient.Set(ctx, key, token, resetTokenTTL).Err(); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	if err := s.SendPasswordResetEmail(ctx, email, user.Name, token, int(resetTokenTTL.Minutes())); err != nil {
		s.log.Warn("failed to send password reset email", "email", email, "error", err)
	}

	return nil
}

// ResetPassword validates the reset token and sets a new password.
func (s *AuthServiceImpl) ResetPassword(ctx context.Context, email, token, newPassword string) error {
	key := resetTokenPrefix + email
	stored, err := s.redisClient.Get(ctx, key).Result()
	if err != nil || stored == "" {
		return errors.New("invalid or expired reset token")
	}
	if stored != token {
		return errors.New("invalid or expired reset token")
	}

	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	hashedStr := string(hashedPassword)

	identities, err := s.repository.ListUserIdentitiesByUserID(ctx, user.ID.String())
	if err != nil {
		return fmt.Errorf("failed to get identities: %w", err)
	}

	var passwordIdentity *schema.UserIdentity
	for _, id := range identities {
		if id.Provider == "password" {
			passwordIdentity = &id
			break
		}
	}

	if passwordIdentity == nil {
		passwordIdentity = &schema.UserIdentity{
			ID:            uuid.New(),
			UserID:        user.ID,
			Provider:      "password",
			ProviderID:    user.ID.String(),
			Email:         email,
			EmailVerified: true,
			PasswordHash:  &hashedStr,
		}
		if err := s.repository.CreateUserIdentity(ctx, passwordIdentity); err != nil {
			return fmt.Errorf("failed to create password identity: %w", err)
		}
	} else {
		passwordIdentity.PasswordHash = &hashedStr
		passwordIdentity.EmailVerified = true
		if err := s.repository.UpdateUserIdentity(ctx, passwordIdentity); err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
	}

	// Cleanup token
	_ = s.redisClient.Del(ctx, key).Err()

	if err := s.SendPasswordChangedEmail(ctx, email, user.Name); err != nil {
		s.log.Warn("failed to send password changed email", "email", email, "error", err)
	}

	return nil
}
