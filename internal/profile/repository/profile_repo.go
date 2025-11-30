package repository

import (
	"context"
	"errors"
	"hauslet/internal/profile/repository/schema"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// CreateProfile creates a new profile record in the database.
func (r *ProfileRerpositoryImpl) CreateProfile(ctx context.Context, profile *schema.Profile) error {
	if profile == nil {
		return errors.New("profile cannot be nil")
	}

	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	if profile.UserID == uuid.Nil {
		return errors.New("user ID cannot be nil")
	}

	return r.db.WithContext(ctx).Create(profile).Error
}

// GetProfileByUserID retrieves a profile by the associated user ID.
func (r *ProfileRerpositoryImpl) GetProfileByUserID(ctx context.Context, userID string) (*schema.Profile, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid userID format")
	}

	var profile schema.Profile
	err = r.db.WithContext(ctx).
		Where("user_id = ?", parsedUserID).
		First(&profile).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &profile, nil
}

// ProfileExists checks if a profile exists for the given user ID.
func (r *ProfileRerpositoryImpl) ProfileExists(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return false, errors.New("invalid userID format")
	}

	var count int64
	err = r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateProfile updates an existing profile record in the database.
func (r *ProfileRerpositoryImpl) UpdateProfile(ctx context.Context, profile *schema.Profile) error {
	if profile == nil {
		return errors.New("profile cannot be nil")
	}

	if profile.ID == uuid.Nil {
		return errors.New("profile ID cannot be nil")
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("id = ?", profile.ID).
		Updates(profile).Error
}

// DeleteProfile performs a soft delete of a profile by user ID.
func (r *ProfileRerpositoryImpl) DeleteProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Where("user_id = ?", parsedUserID).
		Delete(&schema.Profile{}).Error
}

// HardDeleteProfile permanently removes a profile by user ID.
func (r *ProfileRerpositoryImpl) HardDeleteProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Unscoped().
		Where("user_id = ?", parsedUserID).
		Delete(&schema.Profile{}).Error
}

// GetDeletedProfiles retrieves all soft-deleted profiles.
func (r *ProfileRerpositoryImpl) GetDeletedProfiles(ctx context.Context) ([]*schema.Profile, error) {
	var profiles []*schema.Profile

	err := r.db.WithContext(ctx).
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Find(&profiles).Error

	if err != nil {
		return nil, err
	}

	return profiles, nil
}

// GetProfilesByUserIDs fetches profiles for the given user IDs.
func (r *ProfileRerpositoryImpl) GetProfilesByUserIDs(ctx context.Context, userIDs []string) ([]*schema.Profile, error) {
	if len(userIDs) == 0 {
		return []*schema.Profile{}, nil
	}

	parsed := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		parsedID, err := uuid.Parse(id)
		if err != nil {
			return nil, errors.New("invalid userID format in list")
		}
		parsed = append(parsed, parsedID)
	}

	var profiles []*schema.Profile
	if err := r.db.WithContext(ctx).
		Where("user_id IN ?", parsed).
		Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

// GetProfileByID retrieves a profile by its primary ID.
func (r *ProfileRerpositoryImpl) GetProfileByID(ctx context.Context, id string) (*schema.Profile, error) {
	if id == "" {
		return nil, errors.New("id cannot be empty")
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid id format")
	}

	var profile schema.Profile
	err = r.db.WithContext(ctx).
		Where("id = ?", parsedID).
		First(&profile).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &profile, nil
}

// ListProfiles returns a paginated list of profiles.
func (r *ProfileRerpositoryImpl) ListProfiles(ctx context.Context, limit, offset int) ([]*schema.Profile, error) {
	var profiles []*schema.Profile

	query := r.db.WithContext(ctx).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

// GetProfilesByUserType returns profiles containing the provided user type.
func (r *ProfileRerpositoryImpl) GetProfilesByUserType(ctx context.Context, userType string, limit, offset int) ([]*schema.Profile, error) {
	if userType == "" {
		return nil, errors.New("userType cannot be empty")
	}

	var profiles []*schema.Profile
	query := r.db.WithContext(ctx).
		Where("? = ANY(user_types)", userType).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

// GetVerifiedProfiles returns profiles marked verified, optionally filtered by level.
func (r *ProfileRerpositoryImpl) GetVerifiedProfiles(ctx context.Context, level string, limit, offset int) ([]*schema.Profile, error) {
	var profiles []*schema.Profile

	query := r.db.WithContext(ctx).
		Where("id_verified = ?", true).
		Order("verification_date DESC")

	if level != "" {
		query = query.Where("verification_level = ?", level)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

// SearchProfiles performs a simple text search across profile fields.
func (r *ProfileRerpositoryImpl) SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*schema.Profile, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	searchTerm := "%" + query + "%"
	var profiles []*schema.Profile

	dbQuery := r.db.WithContext(ctx).
		Where(
			r.db.Where("bio ILIKE ?", searchTerm).
				Or("occupation ILIKE ?", searchTerm).
				Or("city ILIKE ?", searchTerm).
				Or("state ILIKE ?", searchTerm),
		).
		Order("created_at DESC")

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}
	if offset > 0 {
		dbQuery = dbQuery.Offset(offset)
	}

	if err := dbQuery.Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

// PatchProfile performs a partial update using the provided field map.
func (r *ProfileRerpositoryImpl) PatchProfile(ctx context.Context, id string, updates map[string]interface{}) error {
	if id == "" {
		return errors.New("id cannot be empty")
	}
	if len(updates) == 0 {
		return errors.New("updates cannot be empty")
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid id format")
	}

	normalized := make(map[string]interface{}, len(updates))
	for k, v := range updates {
		switch k {
		case "user_types", "phone_numbers", "skills", "languages", "interests", "hobbies", "badges":
			if slice, ok := v.([]string); ok {
				normalized[k] = pq.StringArray(slice)
				continue
			}
		}
		normalized[k] = v
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("id = ?", parsedID).
		Updates(normalized).Error
}

// IncrementReviewStats atomically adjusts rating and review counters.
func (r *ProfileRerpositoryImpl) IncrementReviewStats(ctx context.Context, userID string, ratingDelta float64, reviewsDelta int) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Updates(map[string]interface{}{
			"rating":        gorm.Expr("rating + ?", ratingDelta),
			"reviews_count": gorm.Expr("reviews_count + ?", reviewsDelta),
		}).Error
}

// SetVerificationStatus updates verification fields for a profile.
func (r *ProfileRerpositoryImpl) SetVerificationStatus(ctx context.Context, userID string, level string, verified bool, verificationDate *time.Time) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	var ts *time.Time
	if verificationDate != nil {
		ts = verificationDate
	} else if verified {
		now := time.Now()
		ts = &now
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Updates(map[string]interface{}{
			"id_verified":        verified,
			"verification_level": level,
			"verification_date":  ts,
		}).Error
}

// AddBadge appends a badge to the badges array if not already present.
func (r *ProfileRerpositoryImpl) AddBadge(ctx context.Context, userID string, badge string) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}
	if badge == "" {
		return errors.New("badge cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Updates(map[string]interface{}{
			"badges": gorm.Expr("CASE WHEN ? = ANY(badges) THEN badges ELSE array_append(badges, ?) END", badge, badge),
		}).Error
}

// RemoveBadge removes a badge from the badges array.
func (r *ProfileRerpositoryImpl) RemoveBadge(ctx context.Context, userID string, badge string) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}
	if badge == "" {
		return errors.New("badge cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Update("badges", gorm.Expr("array_remove(badges, ?)", badge)).Error
}

// UpdateTravelCompanions replaces the travel companions JSONB field.
func (r *ProfileRerpositoryImpl) UpdateTravelCompanions(ctx context.Context, userID string, companions []schema.TravelCompanionProfile) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Update("travel_companions", companions).Error
}

// UpdateTrustScore updates the trust score for a profile.
func (r *ProfileRerpositoryImpl) UpdateTrustScore(ctx context.Context, userID string, score float64) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Update("trust_score", score).Error
}

// RestoreProfile clears the soft-delete flag for a profile by user ID.
func (r *ProfileRerpositoryImpl) RestoreProfile(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid userID format")
	}

	return r.db.WithContext(ctx).
		Unscoped().
		Model(&schema.Profile{}).
		Where("user_id = ?", parsedUserID).
		Update("deleted_at", nil).Error
}
