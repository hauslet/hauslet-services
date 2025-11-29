package server

import (
	"context"
	securitymiddleware "hauslet/cmd/api/server/middleware"
	"hauslet/config"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

func NewHTTPServer(
	ctx context.Context,
	db *gorm.DB,
	rds *redis.RedisClient,
	log *lgr.Logger,
	cfg *config.GlobalConfig,
	mC *email.Client,
	q *queue.Client,
) *http.Server {
	r := chi.NewRouter()

	// Middleware setup
	// Heartbeat must be first to intercept before any other middleware
	r.Use(chimiddleware.Heartbeat("/ping"))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)
	r.Use(securitymiddleware.SecurityHeaders)
	r.Use(securitymiddleware.CORSMiddleware(&cfg.App))

	// Set up routes
	setupRoutes(r, ctx, db, rds, log, cfg, mC, q)

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return srv
}
