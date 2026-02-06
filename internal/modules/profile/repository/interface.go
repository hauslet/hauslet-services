package repository

import (
	"context"
	"hauslet/internal/modules/profile/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	// --- Core CRUD ---
	CreateProfile(ctx context.Context, profile *schema.Profile) error
	GetProfileByUserID(ctx context.Context, userID string) (*schema.Profile, error)
	GetProfileByID(ctx context.Context, id string) (*schema.Profile, error)
	UpdateProfile(ctx context.Context, profile *schema.Profile) error
	PatchProfile(ctx context.Context, id string, updates map[string]any) error
	DeleteProfile(ctx context.Context, userID string) error // Soft delete
	ProfileExists(ctx context.Context, userID string) (bool, error)
	GetProfilesByUserIDs(ctx context.Context, userIDs []string) ([]*schema.Profile, error)
	ListProfiles(ctx context.Context, limit, offset int) ([]*schema.Profile, error)
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*schema.Profile, error)
	GetProfilesByUserType(ctx context.Context, userType string, limit, offset int) ([]*schema.Profile, error)
	GetVerifiedProfiles(ctx context.Context, level string, limit, offset int) ([]*schema.Profile, error)
	UpdateTrustScore(ctx context.Context, userID string, score float64) error
	IncrementReviewStats(ctx context.Context, userID string, ratingDelta float64, reviewsDelta int) error
	UpdateModerationStatus(ctx context.Context, profileID uuid.UUID, status bool) error

	// Handling Verification
	SetVerificationStatus(ctx context.Context, userID string, level string, verified bool, verificationDate *time.Time) error

	// Handling Badges (Postgres Array Append/Remove)
	AddBadge(ctx context.Context, userID string, badge string) error
	RemoveBadge(ctx context.Context, userID string, badge string) error

	// Handling Travel Companions
	AddTravelCompanion(ctx context.Context, userID string, companion schema.TravelCompanion) error
	GetTravelCompanions(ctx context.Context, userID string) ([]*schema.TravelCompanion, error)
	GetTravelCompanionByID(ctx context.Context, userID string, companionID uuid.UUID) (*schema.TravelCompanion, error)
	UpdateTravelCompanion(ctx context.Context, userID string, companion schema.TravelCompanion) error
	DeleteTravelCompanion(ctx context.Context, userID string, companionID string) error

	// Host cancellation tracking
	CreateHostCancellationRecord(ctx context.Context, record *schema.HostCancellationRecord) error
	CountHostCancellationsSince(ctx context.Context, hostID uuid.UUID, since time.Time) (int, error)
	GetHostCancellationRecordByBookingID(ctx context.Context, bookingID uuid.UUID) (*schema.HostCancellationRecord, error)

	// --- Administrative ---
	RestoreProfile(ctx context.Context, userID string) error
	HardDeleteProfile(ctx context.Context, userID string) error
	GetDeletedProfiles(ctx context.Context) ([]*schema.Profile, error)
}

type ProfileRepositoryImpl struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &ProfileRepositoryImpl{db: db}
}
