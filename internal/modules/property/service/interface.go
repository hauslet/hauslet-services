package service

import (
	"context"
	"log/slog"

	"hauslet/internal/modules/auth/authorization"
	businessservice "hauslet/internal/modules/business/service"
	promotionservice "hauslet/internal/modules/promotions/service"
	"hauslet/internal/modules/property/domain"
	"hauslet/internal/modules/property/notification"
	"hauslet/internal/modules/property/repository"
	aiassist "hauslet/internal/platform/ai/assist"
	aiembeddings "hauslet/internal/platform/ai/embeddings"
	"hauslet/internal/platform/events"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/storage"
	"hauslet/internal/platform/xchange"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// PropertyService aggregates property and listing operations.
// This intentionally does not split property vs listing at the service layer.
type PropertyService interface {
	// Property lifecycle
	CreateProperty(ctx context.Context, p domain.Property) (*domain.Property, error)
	UpdateProperty(ctx context.Context, p domain.Property) (*domain.Property, error)
	PatchProperty(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Property, error)
	GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error)
	GetPropertiesByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Property, error)
	ListProperties(ctx context.Context, filter PropertyFilter, page Pagination) ([]domain.Property, int64, error)
	DeleteProperty(ctx context.Context, id uuid.UUID, hard bool) error

	// Listing lifecycle
	CreateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error)
	UpdateListing(ctx context.Context, l domain.Listing) (*domain.Listing, error)
	PatchListing(ctx context.Context, id uuid.UUID, updates map[string]any) (*domain.Listing, error)
	PatchShortletDetails(ctx context.Context, id uuid.UUID, updates map[string]any) error
	PatchRentalDetails(ctx context.Context, id uuid.UUID, updates map[string]any) error
	PatchSaleDetails(ctx context.Context, id uuid.UUID, updates map[string]any) error
	GetListingByID(ctx context.Context, id uuid.UUID, preloadMedia bool) (*domain.Listing, error)
	GetListingByPublicID(ctx context.Context, publicID string, preloadMedia bool) (*domain.Listing, error)
	GetListingBySlug(ctx context.Context, slug string, preloadMedia bool) (*domain.Listing, error)
	GetListingsByIDs(ctx context.Context, ids []uuid.UUID, preloadMedia bool) ([]domain.Listing, error)
	GetListingsByPropertyIDs(ctx context.Context, propertyIDs []uuid.UUID) ([]domain.Listing, error)
	ListListings(ctx context.Context, filter ListingFilter, page Pagination) ([]domain.Listing, int64, error)
	DeleteListing(ctx context.Context, id uuid.UUID, hard bool) error
	GetListingCompleteness(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID) (*domain.ListingCompleteness, error)
	PublishListingRequest(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID) error
	UnpublishListing(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID) (*domain.Listing, error)
	LocalizeListing(ctx context.Context, listing *domain.Listing)

	// Listing media
	UploadListingMedia(ctx context.Context, listingID uuid.UUID, media []domain.ListingMediaInput) ([]domain.ListingMediaResult, error)
	UpdateListingMedia(ctx context.Context, listingID uuid.UUID, mediaID uuid.UUID, updates domain.ListingMediaUpdateInput) error
	DeleteListingMedia(ctx context.Context, listingID uuid.UUID, media []domain.ListingMediaDeleteInput) error
	ListListingMedia(ctx context.Context, listingID uuid.UUID) ([]domain.ListingMedia, error)
	FinalizeListingMedia(ctx context.Context, data domain.FinalizedListingMedia) error
	SearchListings(ctx context.Context, filter ListingFilter, limit int) ([]domain.ScoredListing, error)
	FindSimilarListings(ctx context.Context, listingID uuid.UUID, limit int, minSimilarity float64) ([]domain.ScoredListing, error)
	GenerateListingEmbeddings(ctx context.Context, batchSize int) (int, error)

	// Composite operations
	CreatePropertyWithListing(ctx context.Context, p domain.Property, l domain.Listing) (*domain.Property, *domain.Listing, error)
	UpdateListingWithProperty(ctx context.Context, id uuid.UUID, listingUpdates map[string]any, propertyUpdates map[string]any, requesterID uuid.UUID) (*domain.Listing, error)

	// Event Handling
	SubscribeToVerificationEvents(ctx context.Context, subscriber *events.Subscriber) error

	// AI Features
	GenerateListingDescription(ctx context.Context, input domain.GenerateListingDescriptionInput) (string, error)
}

// BusinessAuthorizer defines permission checks for business-owned listings.
type BusinessAuthorizer interface {
	CanCreateListing(ctx context.Context, businessID uuid.UUID) error
	CanEditListing(ctx context.Context, businessID uuid.UUID) error
	CanDeleteListing(ctx context.Context, businessID uuid.UUID) error
	CanPublishListing(ctx context.Context, businessID uuid.UUID) error
}

type ProfileProvider interface {
	GetProfileData(ctx context.Context, userID string) (string, string, error)
}

// ModerationHooks defines callbacks for moderation-related events.
type ModerationHooks interface {
	EnqueueAIModeration(ctx context.Context, TargetID uuid.UUID, contentType, Payload string) error
}

// ServiceImpl implements the Service interface.
type ServiceImpl struct {
	repo                repository.Repository
	storage             *storage.R2Storage
	queue               *platformQueue.Client
	thumbnailSubject    string
	moderationHooks     ModerationHooks
	notificationService *notification.NotificationService
	profiles            ProfileProvider
	cache               redis.RedisClient
	embedding           *aiembeddings.Client
	aiAssist            aiassist.AssistClient
	log                 *slog.Logger
	embeddingGroup      singleflight.Group
	businessAuthorizer  BusinessAuthorizer
	businessService     businessservice.BusinessService
	subscriptionService promotionservice.SubscriptionService
	supplyGate          authorization.SupplyGate
	fx                  xchange.XChange
}

// NewPropertyService creates a new property service.
func NewPropertyService(repo repository.Repository,
	notificationService *notification.NotificationService,
	profiles ProfileProvider,
	storage *storage.R2Storage,
	queue *platformQueue.Client,
	thumbnailSubject string,
	moderationHooks ModerationHooks,
	cache redis.RedisClient,
	embedding *aiembeddings.Client,
	aiAssist aiassist.AssistClient,
	log *slog.Logger,
	fx xchange.XChange,
	businessAuthorizer BusinessAuthorizer,
	businessService businessservice.BusinessService,
	subscriptionService promotionservice.SubscriptionService,
	supplyGate authorization.SupplyGate,
) *ServiceImpl {
	return &ServiceImpl{
		repo:                repo,
		storage:             storage,
		queue:               queue,
		profiles:            profiles,
		thumbnailSubject:    thumbnailSubject,
		moderationHooks:     moderationHooks,
		notificationService: notificationService,
		cache:               cache,
		embedding:           embedding,
		aiAssist:            aiAssist,
		log:                 log,
		businessAuthorizer:  businessAuthorizer,
		businessService:     businessService,
		subscriptionService: subscriptionService,
		supplyGate:          supplyGate,
		fx:                  fx,
	}
}
