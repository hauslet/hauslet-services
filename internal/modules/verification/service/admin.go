package service

import (
	"context"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// =======================
// Queries
// =======================

func (s *verificationService) ListAttempts(ctx context.Context, sessionID uuid.UUID, requesterID uuid.UUID) ([]*domain.VerificationAttempt, error) {
	// Authorize
	session, err := s.GetSession(ctx, sessionID, requesterID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListAttemptsBySessionID(ctx, session.ID)
}

func (s *verificationService) ExpireOldSessions(ctx context.Context) (int, error) {
	sessions, err := s.repo.ListExpiredSessions(ctx, 100)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, session := range sessions {
		if err := session.MarkExpired(); err != nil {
			continue
		}
		if err := s.repo.UpdateSession(ctx, session); err != nil {
			s.logger.Error("failed to expire session",
				"session_id", session.ID,
				"error", err,
			)
			continue
		}
		count++
	}

	s.logger.Info("expired old sessions", "count", count)
	return count, nil
}

// ApproveVerificationSession manually approves a verification session (Admin only)
func (s *verificationService) ApproveVerificationSession(ctx context.Context, sessionID uuid.UUID) error {
	// 1. Get Session
	// Note: Authentication/Authorization for 'Admin' role should be handled at the handler/resolver layer.
	// We use direct repo access here assuming caller has authorized this action.
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return domain.ErrSessionNotFound
	}

	// 2. Approve
	if err := session.Approve(); err != nil {
		return err
	}

	// 3. Update DB
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return err
	}

	// 4. Notify/Publish Event
	s.notifyVerificationSuccess(ctx, session)

	s.logger.Info("verification session manually approved",
		"session_id", session.ID,
		"type", session.Type,
	)

	return nil
}
