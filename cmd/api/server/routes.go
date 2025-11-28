package server

import (
	"context"
	"hauslet/config"
	"hauslet/internal/auth/port"
	"hauslet/internal/auth/repository"
	"hauslet/internal/auth/service"
	authsession "hauslet/internal/auth/session"
	"hauslet/platform/redis"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

func setupRoutes(r chi.Router, ctx context.Context, db *gorm.DB, rds *redis.RedisClient, log *lgr.Logger, cfg *config.GlobalConfig) {
	// Initialize auth service
	sessionStore := authsession.NewSessionStore(*rds)
	authRepo := repository.NewAuthRepositoryImpl(db, sessionStore)
	authService := service.NewAuthService(&cfg.Auth, authRepo, log)

	// Initialize auth HTTP handler with context
	authHTTP := port.NewHTTPHandler(ctx, authService, log)

	// Setup auth routes
	authHTTP.SetupRoutes(r)

}
