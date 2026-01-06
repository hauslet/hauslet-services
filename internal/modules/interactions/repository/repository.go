package repository

import (
	"context"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/modules/interactions/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InteractionRepository defines the interface for interaction data access
type InteractionRepository interface {
	// Create inserts a single interaction
	Create(ctx context.Context, interaction *schema.Interaction) error

	// BulkInsert inserts multiple interactions in a single transaction (for batch worker)
	BulkInsert(ctx context.Context, interactions []*schema.Interaction) error

	// GetByID retrieves an interaction by ID
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Interaction, error)

	// GetUserHistory retrieves recent interactions for a user
	GetUserHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*schema.Interaction, error)

	// GetEntityInteractions retrieves interactions for a specific entity
	GetEntityInteractions(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID, start, end time.Time) ([]*schema.Interaction, error)

	// UpdateSessionToUser updates session_id interactions to user_id when user logs in
	UpdateSessionToUser(ctx context.Context, sessionID string, userID uuid.UUID) error

	// CountBySessionAndEntity counts interactions by session and entity (for deduplication)
	CountBySessionAndEntity(ctx context.Context, sessionID string, entityType domain.EntityType, entityID uuid.UUID, interactionType domain.InteractionType, since time.Time) (int64, error)

	// GetUniqueEntities returns distinct entity combinations within a time range (for aggregation)
	GetUniqueEntities(ctx context.Context, start, end time.Time) ([]EntityKey, error)
}

// EntityKey represents a unique entity combination
type EntityKey struct {
	EntityType domain.EntityType
	EntityID   uuid.UUID
}

// InteractionRepositoryImpl implements InteractionRepository
type InteractionRepositoryImpl struct {
	db *gorm.DB
}

// NewInteractionRepository creates a new interaction repository
func NewInteractionRepository(db *gorm.DB) InteractionRepository {
	return &InteractionRepositoryImpl{db: db}
}

// Create inserts a single interaction
func (r *InteractionRepositoryImpl) Create(ctx context.Context, interaction *schema.Interaction) error {
	return r.db.WithContext(ctx).Create(interaction).Error
}

// BulkInsert inserts multiple interactions in a batch
func (r *InteractionRepositoryImpl) BulkInsert(ctx context.Context, interactions []*schema.Interaction) error {
	if len(interactions) == 0 {
		return nil
	}

	// Use CreateInBatches for efficient bulk insert
	return r.db.WithContext(ctx).CreateInBatches(interactions, 100).Error
}

// GetByID retrieves an interaction by ID
func (r *InteractionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*schema.Interaction, error) {
	var interaction schema.Interaction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&interaction).Error
	if err != nil {
		return nil, err
	}
	return &interaction, nil
}

// GetUserHistory retrieves recent interactions for a user
func (r *InteractionRepositoryImpl) GetUserHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*schema.Interaction, error) {
	var interactions []*schema.Interaction

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&interactions).Error

	if err != nil {
		return nil, err
	}

	return interactions, nil
}

// GetEntityInteractions retrieves interactions for a specific entity within a time range
func (r *InteractionRepositoryImpl) GetEntityInteractions(
	ctx context.Context,
	entityType domain.EntityType,
	entityID uuid.UUID,
	start, end time.Time,
) ([]*schema.Interaction, error) {
	var interactions []*schema.Interaction

	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ? AND created_at BETWEEN ? AND ?",
			entityType.String(), entityID, start, end).
		Order("created_at DESC").
		Find(&interactions).Error

	if err != nil {
		return nil, err
	}

	return interactions, nil
}

// UpdateSessionToUser updates all interactions with sessionID to have userID
// Called when a user logs in to attribute anonymous activity
func (r *InteractionRepositoryImpl) UpdateSessionToUser(ctx context.Context, sessionID string, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&schema.Interaction{}).
		Where("session_id = ? AND user_id IS NULL", sessionID).
		Update("user_id", userID).Error
}

// CountBySessionAndEntity counts interactions for deduplication
func (r *InteractionRepositoryImpl) CountBySessionAndEntity(
	ctx context.Context,
	sessionID string,
	entityType domain.EntityType,
	entityID uuid.UUID,
	interactionType domain.InteractionType,
	since time.Time,
) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&schema.Interaction{}).
		Where("session_id = ? AND entity_type = ? AND entity_id = ? AND interaction_type = ? AND created_at > ?",
			sessionID, entityType.String(), entityID, interactionType.String(), since).
		Count(&count).Error

	return count, err
}

// GetUniqueEntities returns distinct entity combinations within a time range
func (r *InteractionRepositoryImpl) GetUniqueEntities(ctx context.Context, start, end time.Time) ([]EntityKey, error) {
	var results []EntityKey

	// Use raw SQL for efficiency
	rows, err := r.db.WithContext(ctx).
		Table("interactions").
		Select("DISTINCT entity_type, entity_id").
		Where("created_at >= ? AND created_at < ? AND entity_id IS NOT NULL", start, end).
		Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entityTypeStr string
		var entityID uuid.UUID

		if err := rows.Scan(&entityTypeStr, &entityID); err != nil {
			continue
		}

		entityType := domain.ParseEntityType(entityTypeStr)
		if entityType.IsValid() {
			results = append(results, EntityKey{
				EntityType: entityType,
				EntityID:   entityID,
			})
		}
	}

	return results, nil
}
