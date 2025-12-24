package server

import (
	"context"
	"hauslet/config"
	authhttp "hauslet/internal/modules/auth/port/http"
	authrepository "hauslet/internal/modules/auth/repository"
	"hauslet/internal/modules/auth/service"
	authsession "hauslet/internal/modules/auth/session"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	businessnotification "hauslet/internal/modules/business/notification"
	businessrepository "hauslet/internal/modules/business/repository"
	businessservice "hauslet/internal/modules/business/service"
	moderationhooks "hauslet/internal/modules/moderation/port/hooks"
	moderationrepository "hauslet/internal/modules/moderation/repository"
	moderationservice "hauslet/internal/modules/moderation/service"
	profilenotification "hauslet/internal/modules/profile/notification"
	profileport "hauslet/internal/modules/profile/port/hooks"
	profilerepository "hauslet/internal/modules/profile/repository"
	profileservice "hauslet/internal/modules/profile/service"
	propertynotification "hauslet/internal/modules/property/notification"
	propertyhooks "hauslet/internal/modules/property/port/hooks"
	propertyhttp "hauslet/internal/modules/property/port/http"
	propertyrepository "hauslet/internal/modules/property/repository"
	propertyservice "hauslet/internal/modules/property/service"
	wishlistrepository "hauslet/internal/modules/wishlist/repository"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	aiembeddings "hauslet/internal/platform/ai/embeddings"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/storage"
	"hauslet/internal/transport/graph"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

func setupRoutes(r chi.Router,
	ctx context.Context,
	db *gorm.DB, rds *redis.RedisClient,
	log *lgr.Logger,
	cfg *config.GlobalConfig,
	mC *email.Client,
	q *queue.Client,
	r2 *storage.R2Storage) {
	// Initialize auth service
	sessionStore := authsession.NewSessionStore(*rds)
	authRepo := authrepository.NewAuthRepository(db, sessionStore)
	emailSubject := cfg.YAML.Queue.Subjects["email"]

	// Initialize moderation service (AI client not needed on API path; only enqueue/persist).
	aiModerationSubject := cfg.YAML.Queue.Subjects["ai_moderation"]
	moderationRepo := moderationrepository.NewModerationRepository(db)
	moderationSvc := moderationservice.NewModerationService(moderationRepo, nil, q, aiModerationSubject, nil, nil, log)
	moderationAdapter := moderationhooks.NewModerationAdapter(moderationSvc)

	// Initialize profile service
	profileRepo := profilerepository.NewProfileRepository(db)
	profileNotificationService := profilenotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	profileService := profileservice.NewProfileService(profileRepo, r2, moderationAdapter, profileNotificationService, log)
	authProfileAdapter := profileport.NewAuthHooksAdapter(profileService, cfg.Storage.R2.CDNHost)
	businessProfileAdapter := profileport.NewBusinessProfileAdapter(profileService)

	// Initialize business service (before property service to enable business adapter)
	businessRepo := businessrepository.NewBusinessRepository(db)
	businessnotificationService := businessnotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	businessService := businessservice.NewBusinessService(
		businessRepo,
		businessnotificationService,
		businessProfileAdapter,
		log,
	)
	businessMW := businessmiddleware.NewMiddleware(businessService, log)

	// Initialize property service with business adapter and moderation hooks
	propertyRepo := propertyrepository.NewPropertyRepository(db)
	thumbnailSubject := cfg.YAML.Queue.Subjects["media_thumbnail"]
	propertyNotificationService := propertynotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	propertyProfileAdapter := profileport.NewPropertyProfileAdapter(profileService)

	var embeddingClient *aiembeddings.Client
	if provider, err := aiembeddings.NewGeminiProvider(ctx, cfg.Services.Gemini); err != nil {
		log.Logf("WARN failed to initialize embedding client: %v", err)
	} else {
		embeddingClient = aiembeddings.New(provider)
	}

	propertyService := propertyservice.NewPropertyService(
		propertyRepo,
		propertyNotificationService,
		propertyProfileAdapter,
		r2, q,
		thumbnailSubject,
		moderationAdapter,
		*rds,
		embeddingClient,
		log,
		businessmiddleware.NewPropertyAuthHelper(businessService),
		businessService,
	)

	// Initialize wishlist service
	wishlistRepo := wishlistrepository.NewWishlistRepository(db)
	wishListListAdapter := propertyhooks.NewWishlistHooksAdapter(propertyService)
	wishlistService := wishlistservice.NewWishlistService(wishlistRepo, *rds, wishListListAdapter, log)

	// Initialize auth service
	authService := service.NewAuthService(
		&cfg.Auth,
		authRepo,
		log, mC, *rds, q,
		emailSubject,
		authProfileAdapter,
	)

	// Initialize auth HTTP handler with context
	authHTTP := authhttp.NewHTTPHandler(ctx, authService, log)

	// Setup auth routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		authHTTP.SetupRoutesWithRateLimiting(r, *rds)
	} else {
		authHTTP.SetupRoutes(r)
	}

	// Initialize property HTTP handler with context
	propertyHTTP := propertyhttp.NewHTTPHandler(ctx, propertyService, log)

	// Setup property routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			propertyHTTP.SetupRoutesWithRateLimiting(r, authService, *rds, businessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			propertyHTTP.SetupRoutes(r, authService, businessMW)
		})
	}

	// Setup GraphQL routes
	graph.SetupGraphQL(r, authService, profileService, propertyService, businessService, wishlistService, businessMW.Auth.WithTenantSlug, cfg, log)
}
