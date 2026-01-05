package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/modules/verification/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VerificationRepo implements VerificationRepository using GORM
type VerificationRepo struct {
	db *gorm.DB
}

// NewVerificationRepo creates a new verification repository
func NewVerificationRepo(db *gorm.DB) *VerificationRepo {
	return &VerificationRepo{db: db}
}

// =======================
// Session operations
// =======================

func (r *VerificationRepo) CreateSession(ctx context.Context, session *domain.VerificationSession) error {
	sch := SessionToSchema(session)
	if err := r.db.WithContext(ctx).Create(sch).Error; err != nil {
		return fmt.Errorf("failed to create verification session: %w", err)
	}
	return nil
}

func (r *VerificationRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.VerificationSession, error) {
	var sch schema.VerificationSessionSchema
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}
	return SessionFromSchema(&sch)
}

func (r *VerificationRepo) GetSessionByUserID(ctx context.Context, userID uuid.UUID) (*domain.VerificationSession, error) {
	var sch schema.VerificationSessionSchema
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&sch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session by user ID: %w", err)
	}
	return SessionFromSchema(&sch)
}

func (r *VerificationRepo) UpdateSession(ctx context.Context, session *domain.VerificationSession) error {
	sch := SessionToSchema(session)
	result := r.db.WithContext(ctx).
		Model(&schema.VerificationSessionSchema{}).
		Where("id = ?", session.ID).
		Updates(sch)

	if result.Error != nil {
		return fmt.Errorf("failed to update session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *VerificationRepo) ListExpiredSessions(ctx context.Context, limit int) ([]*domain.VerificationSession, error) {
	var schemas []schema.VerificationSessionSchema
	err := r.db.WithContext(ctx).
		Where("expires_at < ? AND status NOT IN ?", time.Now(), []string{"approved", "rejected", "expired"}).
		Limit(limit).
		Find(&schemas).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list expired sessions: %w", err)
	}

	sessions := make([]*domain.VerificationSession, len(schemas))
	for i, sch := range schemas {
		session, err := SessionFromSchema(&sch)
		if err != nil {
			return nil, err
		}
		sessions[i] = session
	}
	return sessions, nil
}

// =======================
// Attempt operations
// =======================

func (r *VerificationRepo) CreateAttempt(ctx context.Context, attempt *domain.VerificationAttempt) error {
	sch := AttemptToSchema(attempt)
	if err := r.db.WithContext(ctx).Create(sch).Error; err != nil {
		return fmt.Errorf("failed to create verification attempt: %w", err)
	}
	return nil
}

func (r *VerificationRepo) GetAttemptByID(ctx context.Context, id uuid.UUID) (*domain.VerificationAttempt, error) {
	var sch schema.VerificationAttemptSchema
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAttemptNotFound
		}
		return nil, fmt.Errorf("failed to get attempt by ID: %w", err)
	}
	return AttemptFromSchema(&sch)
}

func (r *VerificationRepo) GetAttemptByProviderSessionID(ctx context.Context, providerSessionID string) (*domain.VerificationAttempt, error) {
	var sch schema.VerificationAttemptSchema
	err := r.db.WithContext(ctx).
		Where("provider_session_id = ?", providerSessionID).
		First(&sch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAttemptNotFound
		}
		return nil, fmt.Errorf("failed to get attempt by provider session ID: %w", err)
	}
	return AttemptFromSchema(&sch)
}

func (r *VerificationRepo) ListAttemptsBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.VerificationAttempt, error) {
	var schemas []schema.VerificationAttemptSchema
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&schemas).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list attempts by session ID: %w", err)
	}

	attempts := make([]*domain.VerificationAttempt, len(schemas))
	for i, sch := range schemas {
		attempt, err := AttemptFromSchema(&sch)
		if err != nil {
			return nil, err
		}
		attempts[i] = attempt
	}
	return attempts, nil
}

func (r *VerificationRepo) UpdateAttempt(ctx context.Context, attempt *domain.VerificationAttempt) error {
	sch := AttemptToSchema(attempt)
	result := r.db.WithContext(ctx).
		Model(&schema.VerificationAttemptSchema{}).
		Where("id = ?", attempt.ID).
		Updates(sch)

	if result.Error != nil {
		return fmt.Errorf("failed to update attempt: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrAttemptNotFound
	}
	return nil
}

// =======================
// Evidence operations
// =======================

func (r *VerificationRepo) CreateEvidence(ctx context.Context, evidence *domain.Evidence) error {
	sch := EvidenceToSchema(evidence)
	if err := r.db.WithContext(ctx).Create(sch).Error; err != nil {
		return fmt.Errorf("failed to create evidence: %w", err)
	}
	return nil
}

func (r *VerificationRepo) GetEvidenceByID(ctx context.Context, id uuid.UUID) (*domain.Evidence, error) {
	var sch schema.VerificationEvidenceSchema
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&sch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrEvidenceNotFound
		}
		return nil, fmt.Errorf("failed to get evidence by ID: %w", err)
	}
	return EvidenceFromSchema(&sch)
}

func (r *VerificationRepo) ListEvidenceBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.Evidence, error) {
	var schemas []schema.VerificationEvidenceSchema
	err := r.db.WithContext(ctx).
		Where("session_id = ? AND deleted_at IS NULL", sessionID).
		Order("created_at ASC").
		Find(&schemas).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list evidence by session ID: %w", err)
	}

	evidence := make([]*domain.Evidence, len(schemas))
	for i, sch := range schemas {
		ev, err := EvidenceFromSchema(&sch)
		if err != nil {
			return nil, err
		}
		evidence[i] = ev
	}
	return evidence, nil
}

func (r *VerificationRepo) ListEvidenceByAttemptID(ctx context.Context, attemptID uuid.UUID) ([]*domain.Evidence, error) {
	var schemas []schema.VerificationEvidenceSchema
	err := r.db.WithContext(ctx).
		Where("attempt_id = ? AND deleted_at IS NULL", attemptID).
		Order("created_at ASC").
		Find(&schemas).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list evidence by attempt ID: %w", err)
	}

	evidence := make([]*domain.Evidence, len(schemas))
	for i, sch := range schemas {
		ev, err := EvidenceFromSchema(&sch)
		if err != nil {
			return nil, err
		}
		evidence[i] = ev
	}
	return evidence, nil
}

func (r *VerificationRepo) UpdateEvidence(ctx context.Context, evidence *domain.Evidence) error {
	sch := EvidenceToSchema(evidence)
	result := r.db.WithContext(ctx).
		Model(&schema.VerificationEvidenceSchema{}).
		Where("id = ?", evidence.ID).
		Updates(sch)

	if result.Error != nil {
		return fmt.Errorf("failed to update evidence: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrEvidenceNotFound
	}
	return nil
}

func (r *VerificationRepo) SoftDeleteEvidence(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&schema.VerificationEvidenceSchema{}).
		Where("id = ?", id).
		Update("deleted_at", &now)

	if result.Error != nil {
		return fmt.Errorf("failed to soft delete evidence: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrEvidenceNotFound
	}
	return nil
}
