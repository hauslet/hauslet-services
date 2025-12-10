package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hauslet/config"
	propertyrepository "hauslet/internal/modules/property/repository"
	"hauslet/internal/platform/database"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/logger"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/platform/storage"
	"hauslet/internal/queue"
	jobs "hauslet/internal/queue/jobs/listing"

	"hauslet/internal/transport/worker"
	emailHandler "hauslet/internal/transport/worker/handlers/emails"
	listingHandler "hauslet/internal/transport/worker/handlers/listing"

	"github.com/go-pkgz/lgr"
)

const ShutdownTimeout = 30 * time.Second

func main() {
	cfg := config.Load()
	log := logger.New()

	if cfg.App.Env != "development" {
		log = logger.NewProduction()
	}

	log.Logf("INFO 🔧 Starting worker in %s mode", cfg.App.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, err := database.NewPostgresWithContext(ctx, &cfg.Storage.DB, cfg.App.Env)
	if err != nil {
		log.Logf("ERROR failed to connect to database: %v", err)
		return
	}
	defer database.Close(db)

	// Initialize storage
	storageClient, err := storage.NewR2S3Client(cfg.Storage.R2)
	if err != nil {
		log.Logf("ERROR failed to create R2 S3 client: %v", err)
		return
	}
	r2Storage := storage.NewR2Storage(storageClient, &cfg.Storage.R2)

	// Initialize queue client
	emailSubject := cfg.YAML.Queue.Subjects["email"]
	thumbnailSubject := cfg.YAML.Queue.Subjects["media_thumbnail"]
	cleanupSubject := cfg.YAML.Queue.Subjects["media_cleanup"]
	subjects := []string{emailSubject}
	if thumbnailSubject != "" {
		subjects = append(subjects, thumbnailSubject)
	}
	if cleanupSubject != "" {
		subjects = append(subjects, cleanupSubject)
	}
	queueClient, err := platformQueue.New(
		ctx,
		cfg.Infra.NATS.URL,
		cfg.YAML.Queue.StreamName,
		subjects, // All subjects we'll handle
	)
	if err != nil {
		log.Logf("ERROR Failed to initialize queue: %v", err)
		return
	}
	defer queueClient.Close()

	// Initialize email client
	emailClient := initializeEmailClient(cfg, log)

	// Setup handler registry
	registry := queue.NewRegistry()

	// Register email handler
	emailHandler := emailHandler.NewEmailHandler(emailClient, log, emailSubject)
	registry.Register(emailHandler)

	var propertyRepo propertyrepository.Repository
	if thumbnailSubject != "" || cleanupSubject != "" {
		propertyRepo = propertyrepository.NewPropertyRepository(db)
	}
	// Register listing media thumbnail handler
	if thumbnailSubject != "" {
		thumbnailHandler := listingHandler.NewListingMediaThumbnailHandler(propertyRepo, r2Storage, log, thumbnailSubject)
		registry.Register(thumbnailHandler)
	}

	// Register listing media cleanup handler
	if cleanupSubject != "" {
		cleanupHandler := listingHandler.NewListingMediaCleanupHandler(propertyRepo, r2Storage, log, cleanupSubject)
		registry.Register(cleanupHandler)
	}

	// TODO: Register other handlers as they're implemented
	// registry.Register(notificationHandler)
	// registry.Register(moderationHandler)

	log.Logf("INFO ✅ Registered %d job handlers", registry.HandlerCount())

	// Start worker processor
	processor := worker.NewProcessor(queueClient, registry, log, cfg)

	if err := processor.Start(ctx); err != nil {
		log.Logf("ERROR Failed to start processor: %v", err)
		return
	}

	// Optional periodic cleanup publisher
	if cleanupSubject != "" {
		go func() {
			ticker := time.NewTicker(15 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					job := jobs.ListingMediaCleanupJob{
						OlderThanMinutes: 120,
						Limit:            200,
					}
					if err := queueClient.Publish(ctx, cleanupSubject, job); err != nil {
						log.Logf("ERROR failed to publish cleanup job: %v", err)
					}
				}
			}
		}()
	}

	// Graceful shutdown
	handleShutdown(log, processor)
}

// handleShutdown manages graceful shutdown on signals
func handleShutdown(log *lgr.Logger, processor *worker.Processor) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Logf("WARN Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if err := processor.Stop(shutdownCtx); err != nil {
		log.Logf("ERROR Worker shutdown error: %v", err)
	} else {
		log.Logf("INFO Worker shutdown cleanly")
	}
}

// initializeEmailClient creates the appropriate email client based on environment
func initializeEmailClient(cfg *config.GlobalConfig, log *lgr.Logger) *email.Client {
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

	return email.New(emailSender)
}
