package server

import (
	"context"
	"hauslet/config"
	authhttp "hauslet/internal/auth/port/http"
	authrepository "hauslet/internal/auth/repository"
	"hauslet/internal/auth/service"
	authsession "hauslet/internal/auth/session"
	"hauslet/internal/graph"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	profileport "hauslet/internal/profile/port/hooks"
	profilerepository "hauslet/internal/profile/repository"
	profileservice "hauslet/internal/profile/service"
	propertyrepository "hauslet/internal/property/repository"
	propertyservice "hauslet/internal/property/service"

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
	q *queue.Client) {
	// Initialize auth service
	sessionStore := authsession.NewSessionStore(*rds)
	authRepo := authrepository.NewAuthRepositoryImpl(db, sessionStore)

	// Initialize profile service
	profileRepo := profilerepository.NewProfileRepository(db)
	profileService := profileservice.NewProfileService(profileRepo)
	profileHooks := profileport.NewAuthHooksAdapter(profileService)

	// Initialize property service
	propertyRepo := propertyrepository.NewPropertyRepository(db)
	propertyService := propertyservice.NewPropertyService(propertyRepo)

	authService := service.NewAuthService(&cfg.Auth,
		authRepo, log, mC, *rds, q, cfg.YAML.Queue.Subjects["email"], profileHooks)
	// Initialize auth HTTP handler with context
	authHTTP := authhttp.NewHTTPHandler(ctx, authService, log)

	// Setup auth routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		authHTTP.SetupRoutesWithRateLimiting(r, *rds)
	} else {
		authHTTP.SetupRoutes(r)
	}

	// Setup GraphQL routes
	graph.SetupGraphQL(r, authService, profileService, propertyService, &cfg.App, log)
}
