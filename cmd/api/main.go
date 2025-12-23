package main

import (
	"context"
	"hauslet/cmd/api/server"
	"hauslet/config"
	authSchema "hauslet/internal/modules/auth/repository/schema"
	businessSchema "hauslet/internal/modules/business/repository/schema"
	moderationSchema "hauslet/internal/modules/moderation/repository/schema"
	profileSchema "hauslet/internal/modules/profile/repository/schema"
	propertySchema "hauslet/internal/modules/property/repository/schema"
	wishlistSchema "hauslet/internal/modules/wishlist/repository/schema"
	"hauslet/internal/platform/database"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/logger"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-pkgz/lgr"
)

const (
	ShutdownTimeout = 60 * time.Second
	InitTimeout     = 30 * time.Second
)

func main() {
	cfg := config.Load()
	log := logger.New()
	if cfg.App.Env != "development" {
		log = logger.NewProduction()
	}

	log.Logf("INFO 🚀 Starting Hauslet server in %s mode", cfg.App.Env)

	// Context for initialization steps
	initCtx, cancelInit := context.WithTimeout(context.Background(), InitTimeout)
	defer cancelInit()

	// Connect to the database
	db, err := database.NewPostgresWithContext(initCtx, &cfg.Storage.DB, cfg.App.Env)
	if err != nil {
		log.Logf("ERROR failed to connect to database: %v", err)
		return
	}
	defer database.Close(db)

	log.Logf("INFO ✅ Database connected successfully")

	// Run migrations
	if err := database.RunMigrations(db, log,
		&authSchema.User{},
		&authSchema.UserIdentity{},
		&profileSchema.Profile{},
		&profileSchema.TravelCompanion{},
		&propertySchema.Property{},
		&propertySchema.Listing{},
		&propertySchema.ListingMedia{},
		&businessSchema.Business{},
		&businessSchema.BusinessMember{},
		&businessSchema.BusinessInvitation{},
		&moderationSchema.Moderation{},
		&wishlistSchema.Wishlist{},
		&wishlistSchema.WishlistItem{},
	); err != nil {
		log.Logf("ERROR failed to run migrations: %v", err)
		return
	}
	log.Logf("INFO ✅ Database migrations completed")

	// Initialize Redis
	err = redis.InitRedis(&cfg.Storage.Redis, initCtx)
	if err != nil {
		log.Logf("ERROR failed to initialize Redis: %v", err)
		return
	}
	defer redis.CloseRedis()

	// Initialize Redis client and ping
	redisClient, err := redis.GetRedis()
	if err != nil {
		log.Logf("ERROR failed to get Redis client: %v", err)
		return
	}
	_, err = redisClient.Ping(initCtx).Result()
	if err != nil {
		log.Logf("ERROR failed to connect to Redis: %v", err)
		return
	}
	log.Logf("INFO ✅ Redis connected successfully")

	storageClient, err := storage.NewR2S3Client(cfg.Storage.R2)
	if err != nil {
		log.Logf("ERROR failed to create R2 S3 client: %v", err)
		return
	}
	r2Storage := storage.NewR2Storage(storageClient, &cfg.Storage.R2)
	log.Logf("INFO ✅ Cloudflare R2 storage client initialized")

	// Define email sender based on configuration
	var emailSender email.Sender
	switch cfg.App.Env {
	case "development", "testing":
		emailSender = email.NewSMTPAdapter(
			cfg.Services.Email.SMTP.Host,
			log,
			cfg.Services.Email.SMTP.Port,
			cfg.Services.Email.SMTP.User,
			cfg.Services.Email.SMTP.Pass,
			cfg.Services.Email.From,
		)
		log.Logf("INFO ✅ SMTP email adapter initialized")
	default:
		emailSender = email.NewResendAdapter(
			cfg.Services.Email.Resend.APIKey,
			cfg.Services.Email.From,
		)
		log.Logf("INFO ✅ Resend email adapter initialized")
	}

	mailClient := email.New(emailSender)
	log.Logf("INFO ✅ Email client initialized")

	// Initialize NATS queue (optional fallback to direct send on failure)
	var queueClient *queue.Client
	queueSubjects := []string{cfg.YAML.Queue.Subjects["email"]}
	if thumbSub := cfg.YAML.Queue.Subjects["media_thumbnail"]; thumbSub != "" {
		queueSubjects = append(queueSubjects, thumbSub)
	}
	if cleanupSub := cfg.YAML.Queue.Subjects["media_cleanup"]; cleanupSub != "" {
		queueSubjects = append(queueSubjects, cleanupSub)
	}
	if aiModSub := cfg.YAML.Queue.Subjects["ai_moderation"]; aiModSub != "" {
		queueSubjects = append(queueSubjects, aiModSub)
	}
	if q, err := queue.New(initCtx, cfg.Infra.NATS.URL, cfg.YAML.Queue.StreamName, queueSubjects); err != nil {
		log.Logf("WARN ⚠️ failed to initialize NATS queue, direct send will be used: %v", err)
	} else {
		queueClient = q
		defer queueClient.Close()
		log.Logf("INFO ✅ NATS queue initialized")
	}

	// Create and start the HTTP server
	srv := server.NewHTTPServer(initCtx, db, &redisClient, log, cfg, mailClient, queueClient, r2Storage)
	handleServerLifecycle(srv, log)
}

// handleServerLifecycle manages the lifecycle of the HTTP server
func handleServerLifecycle(srv *http.Server, log *lgr.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Logf("INFO 🌐 HTTP server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Logf("ERROR Server error: %v", err)

	case <-stop:
		log.Logf("WARN Shutdown signal received. Shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Logf("ERROR HTTP server shutdown error: %v", err)
		} else {
			log.Logf("INFO HTTP server shutdown cleanly")
		}
	}

	log.Logf("INFO Graceful shutdown complete.")
}
