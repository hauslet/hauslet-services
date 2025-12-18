package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hauslet/config"
	moderationrepository "hauslet/internal/modules/moderation/repository"
	moderationservice "hauslet/internal/modules/moderation/service"
	profileport "hauslet/internal/modules/profile/port/hooks"
	profilerepository "hauslet/internal/modules/profile/repository"
	profileservice "hauslet/internal/modules/profile/service"
	propertynotification "hauslet/internal/modules/property/notification"
	propertyhooks "hauslet/internal/modules/property/port/hooks"
	propertyrepository "hauslet/internal/modules/property/repository"
	"hauslet/internal/platform/ai"
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
	moderationHandler "hauslet/internal/transport/worker/handlers/moderation"

	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

const ShutdownTimeout = 30 * time.Second

// Infrastructure holds the core platform dependencies
type Infrastructure struct {
	DB      *gorm.DB
	Storage *storage.R2Storage
	AI      *ai.Client
	Email   *email.Client
	Queue   *platformQueue.Client
}

func main() {
	// 1. Setup Configuration and Logger
	cfg := config.Load()
	log := setupLogger(cfg.App.Env)
	log.Logf("INFO 🔧 Starting worker in %s mode", cfg.App.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize Core Infrastructure (DB, AI, Storage, Email)
	infra, err := initInfrastructure(ctx, cfg, log)
	if err != nil {
		log.Logf("CRITICAL failed to initialize infrastructure: %v", err)
		return
	}
	defer database.Close(infra.DB)

	// 3. Initialize Queue (NATS)
	// We do this separately because we need to calculate active subjects first
	queueSubjects := getActiveSubjects(cfg)
	qClient, err := platformQueue.New(ctx, cfg.Infra.NATS.URL, cfg.YAML.Queue.StreamName, queueSubjects)
	if err != nil {
		log.Logf("CRITICAL Failed to initialize queue: %v", err)
		return
	}
	infra.Queue = qClient
	defer infra.Queue.Close()

	// 4. Register Handlers (Wiring Services and Repositories)
	registry := queue.NewRegistry()
	registerHandlers(registry, infra, cfg, log)

	log.Logf("INFO ✅ Registered %d job handlers", registry.HandlerCount())

	// 5. Start Worker Processor
	processor := worker.NewProcessor(infra.Queue, registry, log, cfg)
	if err := processor.Start(ctx); err != nil {
		log.Logf("CRITICAL Failed to start processor: %v", err)
		return
	}

	// 6. Start Periodic Background Jobs
	startPeriodicCleanup(ctx, infra.Queue, cfg, log)

	// 7. Wait for Shutdown
	handleShutdown(log, processor)
}

// setupLogger initializes the logger based on environment
func setupLogger(env string) *lgr.Logger {
	if env != "development" {
		return logger.NewProduction()
	}
	return logger.New()
}

// initInfrastructure establishes connections to DB, Storage, AI, and Email
func initInfrastructure(ctx context.Context, cfg *config.GlobalConfig, log *lgr.Logger) (*Infrastructure, error) {
	// Database
	db, err := database.NewPostgresWithContext(ctx, &cfg.Storage.DB, cfg.App.Env)
	if err != nil {
		return nil, err
	}

	// Storage (R2)
	storageClient, err := storage.NewR2S3Client(cfg.Storage.R2)
	if err != nil {
		return nil, err
	}
	r2Storage := storage.NewR2Storage(storageClient, &cfg.Storage.R2)

	// AI (Gemini)
	aiProviderClient, err := ai.NewGeminiClient(ctx, cfg.Services.Gemini, r2Storage)
	if err != nil {
		return nil, err
	}
	if err := aiProviderClient.HealthCheck(ctx); err != nil {
		return nil, err
	}
	aiClient := ai.New(aiProviderClient)

	// Email
	emailClient := initializeEmailClient(cfg, log)

	return &Infrastructure{
		DB:      db,
		Storage: r2Storage,
		AI:      aiClient,
		Email:   emailClient,
	}, nil
}

// getActiveSubjects builds the list of NATS subjects needed based on config
func getActiveSubjects(cfg *config.GlobalConfig) []string {
	subjects := []string{cfg.YAML.Queue.Subjects["email"]}

	if s := cfg.YAML.Queue.Subjects["media_thumbnail"]; s != "" {
		subjects = append(subjects, s)
	}
	if s := cfg.YAML.Queue.Subjects["media_cleanup"]; s != "" {
		subjects = append(subjects, s)
	}
	if s := cfg.YAML.Queue.Subjects["ai_moderation"]; s != "" {
		subjects = append(subjects, s)
	}
	return subjects
}

// registerHandlers handles Dependency Injection and wires up the worker registry
func registerHandlers(registry *queue.Registry, infra *Infrastructure, cfg *config.GlobalConfig, log *lgr.Logger) {
	qCfg := cfg.YAML.Queue.Subjects

	// 1. Email Handler
	emailH := emailHandler.NewEmailHandler(infra.Email, log, qCfg["email"])
	registry.Register(emailH)

	// Check if we need domain services (Property/Profile)
	hasThumbnail := qCfg["media_thumbnail"] != ""
	hasCleanup := qCfg["media_cleanup"] != ""
	hasModeration := qCfg["ai_moderation"] != ""

	// Initialize Domain Repositories if needed
	var propertyRepo propertyrepository.Repository
	var profileService profileservice.ProfileService
	var propertyProfileAdapter *profileport.PropertyProfileAdapter

	if hasThumbnail || hasCleanup || hasModeration {
		propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
		profileRepo := profilerepository.NewProfileRepository(infra.DB)
		profileService = profileservice.NewProfileService(profileRepo, infra.Storage)
		propertyProfileAdapter = profileport.NewPropertyProfileAdapter(profileService)
	}

	// 2. Listing Media Thumbnail Handler
	if hasThumbnail {
		h := listingHandler.NewListingMediaThumbnailHandler(propertyRepo, infra.Storage, log, qCfg["media_thumbnail"])
		registry.Register(h)
	}

	// 3. Listing Media Cleanup Handler
	if hasCleanup {
		h := listingHandler.NewListingMediaCleanupHandler(propertyRepo, infra.Storage, log, qCfg["media_cleanup"])
		registry.Register(h)
	}

	// 4. AI Moderation Handler
	if hasModeration {
		moderationRepo := moderationrepository.NewModerationRepository(infra.DB)

		propertyNotificationService := propertynotification.NewNotificationService(infra.Email, nil, "", cfg.App.Client, log)
		propertyModerationCallback := propertyhooks.NewModerationPropertyAdapter(propertyRepo, propertyProfileAdapter, propertyNotificationService)

		modService := moderationservice.NewModerationService(moderationRepo, infra.AI, infra.Queue, qCfg["ai_moderation"], propertyModerationCallback)

		h := moderationHandler.NewAIModerationHandler(modService, log, qCfg["ai_moderation"])
		registry.Register(h)
	}
}

// startPeriodicCleanup initializes background tickers
func startPeriodicCleanup(ctx context.Context, queueClient *platformQueue.Client, cfg *config.GlobalConfig, log *lgr.Logger) {
	cleanupSubject := cfg.YAML.Queue.Subjects["media_cleanup"]
	if cleanupSubject == "" {
		return
	}

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
