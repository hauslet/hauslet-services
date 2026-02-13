package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hauslet/internal/modules/auth/repository/schema"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	resetTokenPrefix = "reset:password:"
	resetTokenTTL    = 15 * time.Minute
	resetJWTTTL      = 5 * time.Minute
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

	if err := s.notifier.SendPasswordResetEmail(ctx, email, user.Name, token, int(resetTokenTTL.Minutes())); err != nil {
		s.log.Warn("failed to send password reset email", "email", email, "error", err)
	}

	return nil
}

// VerifyResetOTP validates the OTP and returns a short-lived signed JWT for the final reset step.
func (s *AuthServiceImpl) VerifyResetOTP(ctx context.Context, email, otp string) (string, error) {
	key := resetTokenPrefix + email
	stored, err := s.redisClient.Get(ctx, key).Result()
	if err != nil || stored == "" {
		return "", errors.New("invalid or expired OTP")
	}
	if stored != otp {
		return "", errors.New("invalid or expired OTP")
	}

	// OTP is valid — delete it so it can't be reused.
	_ = s.redisClient.Del(ctx, key).Err()

	// Look up the user to embed their ID in the token.
	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return "", errors.New("user not found")
	}

	// Issue a short-lived JWT carrying only the claims needed for the reset step.
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":     user.ID.String(),
		"email":   email,
		"purpose": "password_reset",
		"iat":     now.Unix(),
		"exp":     now.Add(resetJWTTTL).Unix(),
	}

	signedToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign reset token: %w", err)
	}

	return signedToken, nil
}

// ResetPassword verifies the signed reset JWT and sets a new password.
func (s *AuthServiceImpl) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	// Parse and verify the JWT.
	parsed, err := jwt.Parse(resetToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return errors.New("invalid or expired reset token")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("invalid token claims")
	}

	// Ensure the token was issued for password reset.
	if purpose, _ := claims["purpose"].(string); purpose != "password_reset" {
		return errors.New("invalid token purpose")
	}

	email, _ := claims["email"].(string)
	userID, _ := claims["sub"].(string)
	if email == "" || userID == "" {
		return errors.New("invalid token claims")
	}

	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil || user == nil || user.ID.String() != userID {
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

	if err := s.notifier.SendPasswordChangedEmail(ctx, email, user.Name); err != nil {
		s.log.Warn("failed to send password changed email", "email", email, "error", err)
	}

	return nil
}
