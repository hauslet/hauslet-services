package repository

import (
	"context"
	"fmt"

	"hauslet/internal/modules/discovery/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DiscoveryRepositoryImpl implements the DiscoveryRepository interface
type DiscoveryRepositoryImpl struct {
	db *gorm.DB
}

// NewDiscoveryRepository creates a new discovery repository
func NewDiscoveryRepository(db *gorm.DB) *DiscoveryRepositoryImpl {
	return &DiscoveryRepositoryImpl{
		db: db,
	}
}

// SaveSearchHistory saves a search history record
func (r *DiscoveryRepositoryImpl) SaveSearchHistory(ctx context.Context, history *schema.SearchHistory) error {
	if err := r.db.WithContext(ctx).Create(history).Error; err != nil {
		return fmt.Errorf("failed to save search history: %w", err)
	}
	return nil
}

// GetRecentSearches retrieves recent searches for a user
func (r *DiscoveryRepositoryImpl) GetRecentSearches(ctx context.Context, userID uuid.UUID, limit int) ([]*schema.SearchHistory, error) {
	var searches []*schema.SearchHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&searches).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent searches: %w", err)
	}

	return searches, nil
}

// GetPopularSearches retrieves popular search queries
func (r *DiscoveryRepositoryImpl) GetPopularSearches(ctx context.Context, limit int) ([]string, error) {
	var queries []string

	// Get most common search queries from the last 30 days
	err := r.db.WithContext(ctx).
		Model(&schema.SearchHistory{}).
		Select("query, COUNT(*) as count").
		Where("query != '' AND created_at > NOW() - INTERVAL '30 days'").
		Group("query").
		Order("count DESC").
		Limit(limit).
		Pluck("query", &queries).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get popular searches: %w", err)
	}

	return queries, nil
}

// SaveUserPreferences saves user preferences
func (r *DiscoveryRepositoryImpl) SaveUserPreferences(ctx context.Context, prefs *schema.UserPreferences) error {
	// Upsert: Update if exists, insert if not
	err := r.db.WithContext(ctx).
		Where("user_id = ?", prefs.UserID).
		Assign(prefs).
		FirstOrCreate(prefs).Error

	if err != nil {
		return fmt.Errorf("failed to save user preferences: %w", err)
	}

	return nil
}

// GetUserPreferences retrieves user preferences
func (r *DiscoveryRepositoryImpl) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*schema.UserPreferences, error) {
	var prefs schema.UserPreferences
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&prefs).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No preferences found
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}

	return &prefs, nil
}

// UpdateUserPreferences updates specific fields in user preferences
func (r *DiscoveryRepositoryImpl) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	err := r.db.WithContext(ctx).
		Model(&schema.UserPreferences{}).
		Where("user_id = ?", userID).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	return nil
}
