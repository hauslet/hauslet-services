package service

import (
	"context"
	"fmt"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// =======================
// Session Management
// =======================

func (s *verificationService) CreateSession(ctx context.Context, req CreateSessionRequest) (*domain.VerificationSession, error) {
	// Check if user already has an active session of this type
	existing, err := s.repo.GetSessionByUserID(ctx, req.UserID)
	if err == nil && existing.Type == req.Type && !existing.Status.IsFinal() {
		return nil, domain.ErrSessionAlreadyExists
	}

	// Create new session
	session, err := domain.NewVerificationSession(
		req.UserID,
		req.Type,
		req.Tier,
		req.Data,
		req.Country,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set optional fields
	session.IPAddress = req.IPAddress
	session.UserAgent = req.UserAgent

	// Persist session
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	s.logger.Info("verification session created",
		"session_id", session.ID,
		"user_id", session.UserID,
		"type", session.Type,
		"tier", session.Tier,
		"country", session.Country,
	)

	return session, nil
}

func (s *verificationService) GetSession(ctx context.Context, sessionID uuid.UUID, requesterID uuid.UUID) (*domain.VerificationSession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Authorization: only the session owner can view
	if session.UserID != requesterID {
		return nil, domain.ErrUnauthorized
	}

	return session, nil
}

func (s *verificationService) GetSessionByUser(ctx context.Context, userID uuid.UUID, requesterID uuid.UUID, vType domain.VerificationType) (*domain.VerificationSession, error) {
	// Authorization: users can only view their own sessions
	if userID != requesterID {
		return nil, domain.ErrUnauthorized
	}

	// Get session and filter by type
	session, err := s.repo.GetSessionByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Ensure the session matches the requested type
	if session.Type != vType {
		return nil, domain.ErrSessionNotFound
	}

	return session, nil
}
