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

	// GetUniqueEntities returns distinct entity combinations within a time range (for aggregation) with pagination
	GetUniqueEntities(ctx context.Context, start, end time.Time, limit, offset int) ([]EntityKey, error)

	// GetAggregationMetrics calculates metrics using SQL aggregation for efficiency
	GetAggregationMetrics(ctx context.Context, entityType domain.EntityType, entityID uuid.UUID, start, end time.Time) (*schema.InteractionAggregate, error)
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
func (r *InteractionRepositoryImpl) GetUniqueEntities(ctx context.Context, start, end time.Time, limit, offset int) ([]EntityKey, error) {
	var results []EntityKey

	// Use raw SQL for efficiency
	rows, err := r.db.WithContext(ctx).
		Table("interactions").
		Select("DISTINCT entity_type, entity_id").
		Where("created_at >= ? AND created_at < ? AND entity_id IS NOT NULL", start, end).
		Limit(limit).
		Offset(offset).
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

// GetAggregationMetrics calculates metrics using SQL aggregation
func (r *InteractionRepositoryImpl) GetAggregationMetrics(
	ctx context.Context,
	entityType domain.EntityType,
	entityID uuid.UUID,
	start, end time.Time,
) (*schema.InteractionAggregate, error) {
	var result schema.InteractionAggregate

	// Query breakdown:
	// 1. Filter by entity and time range
	// 2. Filter out bots
	// 3. Count various interaction types
	// 4. Count unique sessions (views_unique)
	// 5. Sum time spent from context (assuming JSONB)

	// Note: gorm:"type:jsonb" implies PostgreSQL.
	// The operator ->> returns text, which we cast to numeric/float.

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN interaction_type IN ('view_listing', 'view_listing_detail') THEN 1 ELSE 0 END), 0) as views_total,
			COUNT(DISTINCT CASE WHEN interaction_type IN ('view_listing', 'view_listing_detail') THEN session_id END) as views_unique,
			COALESCE(SUM(CASE WHEN interaction_type = 'save_listing' THEN 1 ELSE 0 END), 0) as saves_total,
			COALESCE(SUM(CASE WHEN interaction_type = 'unsave_listing' THEN 1 ELSE 0 END), 0) as unsaves_total,
			COALESCE(SUM(CASE WHEN interaction_type = 'share_listing' THEN 1 ELSE 0 END), 0) as shares_total,
			COALESCE(SUM(CASE WHEN interaction_type = 'contact_owner' THEN 1 ELSE 0 END), 0) as contacts_total,
			COALESCE(SUM(CASE WHEN interaction_type = 'booking_request' THEN 1 ELSE 0 END), 0) as booking_requests,
			COALESCE(SUM(CASE 
				WHEN interaction_type = 'time_milestone' 
				THEN COALESCE((context->>'timeSpent')::numeric, 0) 
				ELSE 0 
			END), 0) as total_time_seconds
		FROM interactions
		WHERE 
			entity_type = ? AND 
			entity_id = ? AND 
			created_at >= ? AND 
			created_at < ? AND 
			is_bot = false
	`

	// Using a temporary struct to scan the results including total_time_seconds which isn't in InteractionAggregate directly (it has AvgTimeOnPageSec)
	type AggResult struct {
		ViewsTotal       int64
		ViewsUnique      int64
		SavesTotal       int64
		UnsavesTotal     int64
		SharesTotal      int64
		ContactsTotal    int64
		BookingRequests  int64
		TotalTimeSeconds float64
	}

	var raw AggResult
	err := r.db.WithContext(ctx).Raw(query, entityType.String(), entityID, start, end).Scan(&raw).Error
	if err != nil {
		return nil, err
	}

	// Map to schema.InteractionAggregate
	result.ViewsTotal = raw.ViewsTotal
	result.ViewsUnique = raw.ViewsUnique
	result.SavesTotal = raw.SavesTotal
	result.UnsavesTotal = raw.UnsavesTotal
	result.SharesTotal = raw.SharesTotal
	result.ContactsTotal = raw.ContactsTotal
	result.BookingRequests = raw.BookingRequests

	// Calculate Average Time On Page immediately if possible, or leave it to caller
	if raw.ViewsTotal > 0 {
		result.AvgTimeOnPageSec = int(raw.TotalTimeSeconds / float64(raw.ViewsTotal))
	}

	return &result, nil
}
