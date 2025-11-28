package server

import (
	"context"
	securitymiddleware "hauslet/cmd/api/server/middleware"
	"hauslet/config"
	"hauslet/platform/redis"
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
) *http.Server {
	r := chi.NewRouter()

	// Middleware setup
	r.Use(securitymiddleware.SecurityHeaders)
	r.Use(securitymiddleware.CORSMiddleware(&cfg.App))
	r.Use(chimiddleware.Heartbeat("/health"))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Set up routes
	setupRoutes(r, ctx, db, log, cfg)

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Logf("INFO 🌐 HTTP server ready %s, env %s", srv.Addr, cfg.App.Env)
	return srv
}
