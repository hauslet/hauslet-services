package server

import (
	"context"
	securitymiddleware "hauslet/cmd/api/server/middleware"
	"hauslet/config"
	"hauslet/internal/platform/breaker"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/evidence"
	"hauslet/internal/platform/kyc"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/ratelimit"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/sms"
	"hauslet/internal/platform/storage"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func NewHTTPServer(
	ctx context.Context,
	db *gorm.DB,
	rds *redis.RedisClient,
	log *slog.Logger,
	cfg *config.GlobalConfig,
	mC *email.Client,
	q *queue.Client,
	r2 *storage.R2Storage,
	kycClient *kyc.Client,
	smsClient *sms.Client,
	evidenceStore evidence.Store,
	rateLimiter ratelimit.Limiter,
	circuitBreaker breaker.CircuitBreaker,
) *http.Server {
	r := chi.NewRouter()

	// Middleware setup
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)
	r.Use(securitymiddleware.CORSMiddleware(&cfg.App))
	r.Use(chimiddleware.Heartbeat("/ping"))
	if cfg.App.Env == "production" {
		r.Use(securitymiddleware.SecurityHeaders)
	}

	registerHealthRoutes(r, db, rds)

	// Initialize application container
	container, err := NewContainer(ctx, InfrastructureDependencies{
		DB:             db,
		Redis:          rds,
		Queue:          q,
		R2:             r2,
		Logger:         log,
		Config:         cfg,
		EmailClient:    mC,
		KYC:            kycClient,
		SMS:            smsClient,
		Evidence:       evidenceStore,
		RateLimiter:    rateLimiter,
		CircuitBreaker: circuitBreaker,
	})
	if err != nil {
		log.Error("failed to initialize application container", "error", err)
		panic(err) // Panic is appropriate here as we can't continue without the container
	}

	// Set up routes using the initialized container
	setupRoutes(r, container, cfg)

	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return srv
}
