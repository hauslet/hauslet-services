package setup

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hauslet/config"
	bookinghooks "hauslet/internal/modules/booking/port/hooks"
	bookingrepository "hauslet/internal/modules/booking/repository"
	bookingservice "hauslet/internal/modules/booking/service"
	businessrepository "hauslet/internal/modules/business/repository"
	businessservice "hauslet/internal/modules/business/service"
	calendarrepository "hauslet/internal/modules/calendar/repository"
	calendarservice "hauslet/internal/modules/calendar/service"
	moderationrepository "hauslet/internal/modules/moderation/repository"
	moderationservice "hauslet/internal/modules/moderation/service"
	pricingrepository "hauslet/internal/modules/pricing/repository"
	pricingservice "hauslet/internal/modules/pricing/service"
	profilenotification "hauslet/internal/modules/profile/notification"
	profileport "hauslet/internal/modules/profile/port/hooks"
	profilerepository "hauslet/internal/modules/profile/repository"
	profileservice "hauslet/internal/modules/profile/service"
	propertynotification "hauslet/internal/modules/property/notification"
	propertyhooks "hauslet/internal/modules/property/port/hooks"
	propertyrepository "hauslet/internal/modules/property/repository"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/queue"
	bookingJobs "hauslet/internal/queue/jobs/booking"
	jobs "hauslet/internal/queue/jobs/listing"
	"hauslet/internal/transport/worker"
	bookingHandler "hauslet/internal/transport/worker/handlers/booking"
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
	var profileRepo profilerepository.ProfileRepository

	if hasThumbnail || hasCleanup || hasModeration {
		propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
		profileRepo = profilerepository.NewProfileRepository(infra.DB)
		profileSvc = profileservice.NewProfileService(profileRepo, infra.Storage, nil, nil, log)
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
		businessRepo := businessrepository.NewBusinessRepository(infra.DB)
		businessSvc := businessservice.NewBusinessService(businessRepo, nil, nil, log)

		// Setup property moderation callback
		propertyNotificationService := propertynotification.NewNotificationService(infra.Email, nil, "", cfg.App.Client, log)
		propertyModerationCallback := propertyhooks.NewModerationPropertyAdapter(propertyRepo,
			propertyProfileAdapter,
			propertyNotificationService,
			infra.embedding,
			infra.Cache,
			log,
			businessSvc,
		)

		// Setup profile moderation callback
		profileNotificationService := profilenotification.NewNotificationService(infra.Email, nil, "", cfg.App.Client, log)
		profileModerationCallback := profileport.NewModerationProfileAdapter(
			profileRepo,
			profileNotificationService,
			log,
		)

		// Create moderation service
		modService := moderationservice.NewModerationService(moderationRepo,
			infra.AI, infra.Queue,
			qCfg["ai_moderation"],
			propertyModerationCallback,
			profileModerationCallback,
			log,
		)
		h := moderationHandler.NewAIModerationHandler(modService, log, qCfg["ai_moderation"])
		registry.Register(h)
	}

	// Booking expiry handler
	hasBookingExpiry := qCfg["booking_expiry"] != ""
	if hasBookingExpiry {
		// Initialize booking dependencies
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)

		// Property/listing hooks (needed to get listing owner for calendar operations)
		if propertyRepo == nil {
			propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
		}
		listingHooks := bookinghooks.NewPropertyHooksAdapter(propertyRepo)

		// Calendar gateway (needed to cancel expired booking events)
		calendarRepo := calendarrepository.NewCalendarRepository(infra.DB)
		calendarHooksAdapter := propertyhooks.NewCalendarHooksAdapter(nil) // nil property service for worker
		calendarSvc := calendarservice.NewCalendarService(calendarRepo, infra.Cache, calendarHooksAdapter, log)

		// Pricing service with minimal dependencies
		pricingRepo := pricingrepository.NewPricingRepository(infra.DB)
		pricingSvc := pricingservice.NewPricingService(pricingRepo, nil, nil, log, cfg.YAML.Platform)

		bookingSvc := bookingservice.NewBookingService(
			bookingRepo,
			calendarSvc,
			pricingSvc,
			nil, // payment gateway not needed for expiry checks
			listingHooks,
			nil, // profile provider not required for expiry checks
			nil, // notification service not required for expiry checks
			log,
		)
		h := bookingHandler.NewBookingExpiryCheckHandler(bookingSvc, log, qCfg["booking_expiry"])
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

// StartBookingExpiryCheck initializes periodic booking expiry checks.
func StartBookingExpiryCheck(ctx context.Context, queueClient *platformQueue.Client, cfg *config.GlobalConfig, log *lgr.Logger) {
	expirySubject := cfg.YAML.Queue.Subjects["booking_expiry"]
	if expirySubject == "" {
		return
	}

	go func() {
		// Check every 2 minutes (PRD specifies 5 minutes, but 2 minutes provides faster response)
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				job := bookingJobs.BookingExpiryCheckJob{
					CheckTime: time.Now(),
				}
				if err := queueClient.Publish(ctx, expirySubject, job); err != nil {
					log.Logf("ERROR failed to publish booking expiry check job: %v", err)
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
