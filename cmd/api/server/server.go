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
	redisClient *redis.RedisClient,
	logger *slog.Logger,
	cfg *config.GlobalConfig,
	email *email.Client,
	queue *queue.Client,
	r2 *storage.R2Storage,
	kyc *kyc.Client,
	sms *sms.Client,
	evidence evidence.Store,
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

	registerHealthRoutes(r, db, redisClient)

	// Initialize application container
	container, err := NewContainer(ctx, InfrastructureDependencies{
		DB:             db,
		Redis:          redisClient,
		Queue:          queue,
		R2:             r2,
		Logger:         logger,
		Config:         cfg,
		EmailClient:    email,
		KYC:            kyc,
		SMS:            sms,
		Evidence:       evidence,
		RateLimiter:    rateLimiter,
		CircuitBreaker: circuitBreaker,
	})
	if err != nil {
		logger.Error("failed to initialize application container", "error", err)
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
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	return srv
}
