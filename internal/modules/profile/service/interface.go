package service

import (
	"context"
	"log/slog"
	"time"

	"hauslet/internal/modules/profile/domain"
	"hauslet/internal/modules/profile/notification"
	"hauslet/internal/modules/profile/repository"
	"hauslet/internal/platform/storage"

	"github.com/google/uuid"
)

// ProfileService defines business-level operations for profile management.
type ProfileService interface {
	// Core CRUD
	CreateProfile(ctx context.Context, profile domain.Profile) (*domain.Profile, error)
	GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	GetProfileByID(ctx context.Context, id string) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, profile domain.Profile) (*domain.Profile, error)
	PatchProfile(ctx context.Context, id string, updates map[string]any) (*domain.Profile, error)
	DeleteProfile(ctx context.Context, userID string) error // Soft delete

	// Discovery
	ListProfiles(ctx context.Context, limit, offset int) ([]domain.Profile, error)
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]domain.Profile, error)
	GetProfilesByUserType(ctx context.Context, userType string, limit, offset int) ([]domain.Profile, error)
	GetVerifiedProfiles(ctx context.Context, level string, limit, offset int) ([]domain.Profile, error)

	// Batch/Helpers
	GetProfilesByUserIDs(ctx context.Context, userIDs []string) ([]domain.Profile, error)
	ProfileExists(ctx context.Context, userID string) (bool, error)

	// Field-level operations
	SelectSupplyRoles(ctx context.Context, userID string, userTypes []domain.UserType) (*domain.Profile, error)
	AddBadge(ctx context.Context, userID string, badge domain.Badge) error
	RemoveBadge(ctx context.Context, userID string, badge domain.Badge) error
	AddTravelCompanion(ctx context.Context, userID string, companion domain.TravelCompanion) error
	DeleteTravelCompanion(ctx context.Context, userID string, companionID string) error
	UpdateTravelCompanion(ctx context.Context, userID string, companion domain.TravelCompanion) error
	UpdateTrustScore(ctx context.Context, userID string, score float64) error
	IncrementReviewStats(ctx context.Context, userID string, ratingDelta float64, reviewsDelta int) error
	SetVerificationStatus(ctx context.Context, userID string, level string, verified bool, verificationDate *time.Time) error
	UploadProfilePhoto(ctx context.Context, userID, filename string) (*domain.UploadResult, error)
	UploadTravelCompanionPhoto(ctx context.Context, CompanionID uuid.UUID, userID, filename string) (*domain.UploadResult, error)

	// Administrative
	RestoreProfile(ctx context.Context, userID string) error
	HardDeleteProfile(ctx context.Context, userID string) error
	GetDeletedProfiles(ctx context.Context) ([]domain.Profile, error)

	// Hooks
	CreateDefaultProfile(ctx context.Context, userID string, name string, birthDate *time.Time) error
}

// ModerationHooks defines callbacks for moderation-related events.
type ModerationHooks interface {
	EnqueueAIModeration(ctx context.Context, TargetID uuid.UUID, contentType, Payload string) error
}

// SubscriptionAdapter provides subscription operations for profile service.
// Following dependency inversion: interface defined where consumed, implemented in port/hooks.
type SubscriptionAdapter interface {
	// GetOrCreateFreeSubscription gets existing free subscription or creates one
	GetOrCreateFreeSubscription(ctx context.Context, userID uuid.UUID) error

	// HasActiveSubscription checks if user has any active subscription
	HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error)
}

// ProfileServiceImpl provides business-level operations for profiles.
type ProfileServiceImpl struct {
	repo                repository.ProfileRepository
	storage             *storage.R2Storage
	moderationHooks     ModerationHooks
	subscriptionAdapter SubscriptionAdapter // Optional: can be nil
	notificationService *notification.NotificationService
	log                 *slog.Logger
}

// NewProfileService creates a new profile service.
func NewProfileService(repo repository.ProfileRepository,
	storage *storage.R2Storage,
	moderationHooks ModerationHooks,
	subscriptionAdapter SubscriptionAdapter, // Optional: can be nil
	notificationService *notification.NotificationService,
	log *slog.Logger,
) ProfileService {
	return &ProfileServiceImpl{
		repo:                repo,
		storage:             storage,
		moderationHooks:     moderationHooks,
		subscriptionAdapter: subscriptionAdapter,
		notificationService: notificationService,
		log:                 log,
	}
}
