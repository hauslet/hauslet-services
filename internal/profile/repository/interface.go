package repository

import (
	"context"
	"hauslet/internal/profile/repository/schema"
	"time"

	"gorm.io/gorm"
)

type ProfileRepository interface {
	// --- Core CRUD ---
	CreateProfile(ctx context.Context, profile *schema.Profile) error
	GetProfileByUserID(ctx context.Context, userID string) (*schema.Profile, error)
	GetProfileByID(ctx context.Context, id string) (*schema.Profile, error)
	UpdateProfile(ctx context.Context, profile *schema.Profile) error
	PatchProfile(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteProfile(ctx context.Context, userID string) error // Soft delete
	ProfileExists(ctx context.Context, userID string) (bool, error)

	// --- Performance / Batching ---
	// Essential for resolving lists of users (e.g. "Reviews" section)
	GetProfilesByUserIDs(ctx context.Context, userIDs []string) ([]*schema.Profile, error)

	// --- Search & Discovery ---
	ListProfiles(ctx context.Context, limit, offset int) ([]*schema.Profile, error)
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*schema.Profile, error)
	GetProfilesByUserType(ctx context.Context, userType string, limit, offset int) ([]*schema.Profile, error)
	GetVerifiedProfiles(ctx context.Context, level string, limit, offset int) ([]*schema.Profile, error)

	// --- Specific Field Updates (Atomic Operations) ---
	// Handling Review aggregates
	IncrementReviewStats(ctx context.Context, userID string, ratingDelta float64, reviewsDelta int) error

	// Handling Verification
	SetVerificationStatus(ctx context.Context, userID string, level string, verified bool, verificationDate *time.Time) error

	// Handling Badges (Postgres Array Append/Remove)
	AddBadge(ctx context.Context, userID string, badge string) error
	RemoveBadge(ctx context.Context, userID string, badge string) error

	// Handling JSONB
	UpdateTravelCompanions(ctx context.Context, userID string, companions []schema.TravelCompanionProfile) error

	// Handling Reputation
	UpdateTrustScore(ctx context.Context, userID string, score float64) error

	// --- Administrative ---
	RestoreProfile(ctx context.Context, userID string) error
	HardDeleteProfile(ctx context.Context, userID string) error
	GetDeletedProfiles(ctx context.Context) ([]*schema.Profile, error)
}

type ProfileRerpositoryImpl struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &ProfileRerpositoryImpl{db: db}
}
