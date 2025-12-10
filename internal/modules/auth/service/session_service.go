package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/auth/domain"
)

// Session management

func (s *AuthServiceImpl) GetUserSessions(ctx context.Context, userID string) ([]domain.Session, error) {
	sessions, err := s.repository.ListActiveSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	return sessions, nil
}

func (s *AuthServiceImpl) RevokeSession(ctx context.Context, sessionID string) error {
	if err := s.repository.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

func (s *AuthServiceImpl) RevokeAllUserSessions(ctx context.Context, userID string) error {
	if err := s.repository.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all user sessions: %w", err)
	}
	return nil
}

func (s *AuthServiceImpl) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	newExpiry := time.Now().Add(duration)
	if err := s.repository.UpdateSessionExpiry(ctx, sessionID, newExpiry); err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}
	return nil
}
