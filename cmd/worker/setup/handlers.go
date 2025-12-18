package setup

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
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/queue"
	jobs "hauslet/internal/queue/jobs/listing"
	"hauslet/internal/transport/worker"
	emailHandler "hauslet/internal/transport/worker/handlers/emails"
	listingHandler "hauslet/internal/transport/worker/handlers/listing"
	moderationHandler "hauslet/internal/transport/worker/handlers/moderation"

	"github.com/go-pkgz/lgr"
)

// RegisterHandlers builds and registers all queue handlers based on config.
func RegisterHandlers(infra *Infrastructure, cfg *config.GlobalConfig, log *lgr.Logger) *queue.Registry {
	registry := queue.NewRegistry()
	qCfg := cfg.YAML.Queue.Subjects

	// Email handler
	emailH := emailHandler.NewEmailHandler(infra.Email, log, qCfg["email"])
	registry.Register(emailH)

	// Feature flags
	hasThumbnail := qCfg["media_thumbnail"] != ""
	hasCleanup := qCfg["media_cleanup"] != ""
	hasModeration := qCfg["ai_moderation"] != ""

	// Shared repos/services
	var propertyRepo propertyrepository.Repository
	var profileSvc profileservice.ProfileService
	var propertyProfileAdapter *profileport.PropertyProfileAdapter

	if hasThumbnail || hasCleanup || hasModeration {
		propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
		profileRepo := profilerepository.NewProfileRepository(infra.DB)
		profileSvc = profileservice.NewProfileService(profileRepo, infra.Storage)
		propertyProfileAdapter = profileport.NewPropertyProfileAdapter(profileSvc)
	}

	// Thumbnail handler
	if hasThumbnail {
		h := listingHandler.NewListingMediaThumbnailHandler(propertyRepo, infra.Storage, log, qCfg["media_thumbnail"])
		registry.Register(h)
	}

	// Cleanup handler
	if hasCleanup {
		h := listingHandler.NewListingMediaCleanupHandler(propertyRepo, infra.Storage, log, qCfg["media_cleanup"])
		registry.Register(h)
	}

	// AI moderation handler
	if hasModeration {
		moderationRepo := moderationrepository.NewModerationRepository(infra.DB)
		propertyNotificationService := propertynotification.NewNotificationService(infra.Email, nil, "", cfg.App.Client, log)
		propertyModerationCallback := propertyhooks.NewModerationPropertyAdapter(propertyRepo, propertyProfileAdapter, propertyNotificationService)
		modService := moderationservice.NewModerationService(moderationRepo,
			infra.AI, infra.Queue,
			qCfg["ai_moderation"],
			propertyModerationCallback, log)
		h := moderationHandler.NewAIModerationHandler(modService, log, qCfg["ai_moderation"])
		registry.Register(h)
	}

	return registry
}

// StartPeriodicCleanup initializes background tickers.
func StartPeriodicCleanup(ctx context.Context, queueClient *platformQueue.Client, cfg *config.GlobalConfig, log *lgr.Logger) {
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

// HandleShutdown manages graceful shutdown on signals.
func HandleShutdown(log *lgr.Logger, processor *worker.Processor) {
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
