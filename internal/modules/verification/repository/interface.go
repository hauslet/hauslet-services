package repository

import (
	"context"

	"hauslet/internal/modules/verification/domain"

	"github.com/google/uuid"
)

// VerificationRepository handles persistence for verification aggregates
type VerificationRepository interface {
	// Session operations
	CreateSession(ctx context.Context, session *domain.VerificationSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.VerificationSession, error)
	GetSessionByUserID(ctx context.Context, userID uuid.UUID) (*domain.VerificationSession, error)
	UpdateSession(ctx context.Context, session *domain.VerificationSession) error
	ListExpiredSessions(ctx context.Context, limit int) ([]*domain.VerificationSession, error)

	// Attempt operations
	CreateAttempt(ctx context.Context, attempt *domain.VerificationAttempt) error
	GetAttemptByID(ctx context.Context, id uuid.UUID) (*domain.VerificationAttempt, error)
	GetAttemptByProviderSessionID(ctx context.Context, providerSessionID string) (*domain.VerificationAttempt, error)
	ListAttemptsBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.VerificationAttempt, error)
	UpdateAttempt(ctx context.Context, attempt *domain.VerificationAttempt) error

	// Evidence operations
	CreateEvidence(ctx context.Context, evidence *domain.Evidence) error
	GetEvidenceByID(ctx context.Context, id uuid.UUID) (*domain.Evidence, error)
	ListEvidenceBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.Evidence, error)
	ListEvidenceByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]*domain.Evidence, error)
	UpdateEvidence(ctx context.Context, evidence *domain.Evidence) error
	SoftDeleteEvidence(ctx context.Context, id uuid.UUID) error
}
