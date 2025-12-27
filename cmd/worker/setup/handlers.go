package setup

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"hauslet/config"
	bookinghooks "hauslet/internal/modules/booking/port/hooks"
	bookingrepository "hauslet/internal/modules/booking/repository"
	bookingservice "hauslet/internal/modules/booking/service"
	bookingnotification "hauslet/internal/modules/booking/notification"
	businessrepository "hauslet/internal/modules/business/repository"
	businessservice "hauslet/internal/modules/business/service"
	calendarrepository "hauslet/internal/modules/calendar/repository"
	calendarservice "hauslet/internal/modules/calendar/service"
	financenotification "hauslet/internal/modules/finance/notification"
	financehooks "hauslet/internal/modules/finance/port/hooks"
	financerepository "hauslet/internal/modules/finance/repository"
	financeservice "hauslet/internal/modules/finance/service"
	moderationrepository "hauslet/internal/modules/moderation/repository"
	moderationservice "hauslet/internal/modules/moderation/service"
	paymentsnotification "hauslet/internal/modules/payments/notification"
	paymentshttp "hauslet/internal/modules/payments/port/http"
	paymentsrepository "hauslet/internal/modules/payments/repository"
	paymentsservice "hauslet/internal/modules/payments/service"
	pricingrepository "hauslet/internal/modules/pricing/repository"
	pricingservice "hauslet/internal/modules/pricing/service"
	profilenotification "hauslet/internal/modules/profile/notification"
	profileport "hauslet/internal/modules/profile/port/hooks"
	profilerepository "hauslet/internal/modules/profile/repository"
	profileservice "hauslet/internal/modules/profile/service"
	propertynotification "hauslet/internal/modules/property/notification"
	propertyhooks "hauslet/internal/modules/property/port/hooks"
	propertyrepository "hauslet/internal/modules/property/repository"
	"hauslet/internal/platform/payment"
	"hauslet/internal/queue"
	bookingHandler "hauslet/internal/transport/worker/handlers/booking"
	emailHandler "hauslet/internal/transport/worker/handlers/emails"
	financeHandler "hauslet/internal/transport/worker/handlers/finance"
	listingHandler "hauslet/internal/transport/worker/handlers/listing"
	moderationHandler "hauslet/internal/transport/worker/handlers/moderation"
	paymentHandler "hauslet/internal/transport/worker/handlers/payments"

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
	hasPaymentWebhook := qCfg["payment_webhook"] != ""

	// Shared repos/services
	var propertyRepo propertyrepository.Repository
	var profileSvc profileservice.ProfileService
	var propertyProfileAdapter *profileport.PropertyProfileAdapter
	var profileRepo profilerepository.ProfileRepository

	if hasThumbnail || hasCleanup || hasModeration || hasPaymentWebhook {
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
	hasBookingRefund := qCfg["booking_refund"] != ""
	if hasBookingExpiry || hasBookingRefund {
		// Initialize booking dependencies
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)

		if hasBookingExpiry {
			// Property/listing hooks (needed to get listing owner for calendar operations)
			if propertyRepo == nil {
				propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
			}
			listingHooks := bookinghooks.NewPropertyHooksAdapter(propertyRepo)

			// Calendar gateway (needed to cancel expired booking events)
			calendarRepo := calendarrepository.NewCalendarRepository(infra.DB)
			calendarHooksAdapter := propertyhooks.NewCalendarHooksRepoAdapter(propertyRepo)
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
				nil, // refund queue not required for expiry checks
				"",
				log,
			)
			h := bookingHandler.NewBookingExpiryCheckHandler(bookingSvc, log, qCfg["booking_expiry"])
			registry.Register(h)
		}

		if hasBookingRefund {
			paymentFactory := payment.NewProviderFactory(cfg.Services.Payment)
			paymentClient := payment.New(paymentFactory)
			paymentsRepo := paymentsrepository.NewRepository(infra.DB)
			paymentsNotification := paymentsnotification.NewNotificationService(
				infra.Email,
				infra.Queue,
				qCfg["email"],
				cfg.App.Client,
				log,
			)
			paymentsSvc := paymentsservice.NewPaymentService(
				paymentsRepo,
				paymentClient,
				paymentsNotification,
				log,
			)

			h := bookingHandler.NewBookingRefundHandler(bookingRepo, paymentsSvc, log, qCfg["booking_refund"])
			registry.Register(h)
		}
	}

	if hasPaymentWebhook {
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)

		listingHooks := bookinghooks.NewPropertyHooksAdapter(propertyRepo)
		calendarRepo := calendarrepository.NewCalendarRepository(infra.DB)
		calendarHooksAdapter := propertyhooks.NewCalendarHooksRepoAdapter(propertyRepo)
		calendarSvc := calendarservice.NewCalendarService(calendarRepo, infra.Cache, calendarHooksAdapter, log)

		bookingNotificationService := bookingnotification.NewNotificationService(
			infra.Email,
			infra.Queue,
			qCfg["email"],
			cfg.App.Client,
			log,
		)
		bookingProfileAdapter := profileport.NewBookingProfileAdapter(profileSvc)

		bookingSvc := bookingservice.NewBookingService(
			bookingRepo,
			calendarSvc,
			nil, // pricing not needed for webhook processing
			nil, // payment gateway not needed for webhook processing
			listingHooks,
			bookingProfileAdapter,
			bookingNotificationService,
			nil, // refund queue not needed for webhook processing
			"",
			log,
		)
		bookingHooksAdapter := bookinghooks.NewBookingHooksAdapter(bookingSvc)

		paymentFactory := payment.NewProviderFactory(cfg.Services.Payment)
		paymentClient := payment.New(paymentFactory)
		paymentsRepo := paymentsrepository.NewRepository(infra.DB)
		paymentsNotification := paymentsnotification.NewNotificationService(
			infra.Email,
			infra.Queue,
			qCfg["email"],
			cfg.App.Client,
			log,
		)
		paymentsSvc := paymentsservice.NewPaymentService(
			paymentsRepo,
			paymentClient,
			paymentsNotification,
			log,
		)

		financeWalletRepo := financerepository.NewWalletRepository(infra.DB)
		financeLedgerRepo := financerepository.NewLedgerRepository(infra.DB)
		financeTransactionRepo := financerepository.NewTransactionRepository(infra.DB)
		financeDisbursementRepo := financerepository.NewDisbursementRepository(infra.DB)
		financeSvc := financeservice.NewFinanceService(
			financeWalletRepo,
			financeLedgerRepo,
			financeTransactionRepo,
			financeDisbursementRepo,
			infra.DB,
			log,
		)
		financeHooksAdapter := financehooks.NewPaymentHooksAdapter(financeSvc)

		payoutSvc := financeservice.NewPayoutService(
			financeWalletRepo,
			financeLedgerRepo,
			financeTransactionRepo,
			financeDisbursementRepo,
			paymentsRepo,
			nil, // booking querier not needed for webhook updates
			nil, // notification service not required for webhook updates
			nil, // booking hooks not required for webhook updates
			paymentClient,
			nil, // profile adapter not required for webhook updates
			cfg.YAML.Platform,
			infra.DB,
			log,
		)
		payoutHooksAdapter := financehooks.NewPayoutHooksAdapter(payoutSvc)

		webhookHandler := paymentshttp.NewWebhookHandler(
			paymentsSvc,
			paymentClient,
			bookingHooksAdapter,
			financeHooksAdapter,
			payoutHooksAdapter,
			nil,
			"",
			log,
		)

		h := paymentHandler.NewPaymentWebhookHandler(paymentClient, webhookHandler, log, qCfg["payment_webhook"])
		registry.Register(h)
	}

	// Payout processing handlers
	hasPayoutProcess := qCfg["payout_process"] != ""
	hasPayoutRetry := qCfg["payout_retry"] != ""
	if hasPayoutProcess || hasPayoutRetry {
		// Initialize finance repositories
		financeWalletRepo := financerepository.NewWalletRepository(infra.DB)
		financeLedgerRepo := financerepository.NewLedgerRepository(infra.DB)
		financeTransactionRepo := financerepository.NewTransactionRepository(infra.DB)
		financeDisbursementRepo := financerepository.NewDisbursementRepository(infra.DB)

		// Initialize payments repository for payout details
		paymentsRepo := paymentsrepository.NewRepository(infra.DB)

		// Initialize payment client
		paymentFactory := payment.NewProviderFactory(cfg.Services.Payment)
		paymentClient := payment.New(paymentFactory)

		// Initialize booking repository for payout processing
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)
		bookingQuerierAdapter := financehooks.NewBookingQuerierAdapter(bookingRepo)

		// Initialize payout hooks adapter for marking bookings as settled
		bookingPayoutHooksAdapter := bookinghooks.NewSimplePayoutHooksAdapter(bookingRepo, log)

		// Initialize finance notification service
		financeNotificationSvc := financenotification.NewNotificationService(
			infra.Email,
			infra.Queue,
			qCfg["email"],
			cfg.App.Client,
			log,
		)

		// Initialize payout service
		payoutSvc := financeservice.NewPayoutService(
			financeWalletRepo,
			financeLedgerRepo,
			financeTransactionRepo,
			financeDisbursementRepo,
			paymentsRepo,
			bookingQuerierAdapter,        // Booking querier for finding bookings ready for payout
			financeNotificationSvc,
			bookingPayoutHooksAdapter,    // Booking hooks for marking bookings as settled after payout
			paymentClient,
			nil, // profile adapter not needed for worker (notifications handled by API)
			cfg.YAML.Platform,
			infra.DB,
			log,
		)

		if hasPayoutProcess {
			h := financeHandler.NewPayoutProcessHandler(payoutSvc, log, qCfg["payout_process"])
			registry.Register(h)
		}

		if hasPayoutRetry {
			h := financeHandler.NewDisbursementRetryHandler(payoutSvc, log, qCfg["payout_retry"])
			registry.Register(h)
		}
	}

	return registry
}

// REMOVED: Ticker functions (StartPeriodicCleanup, StartBookingExpiryCheck,
// StartPayoutProcessing, StartDisbursementRetry) - replaced by Cloud Scheduler.
// See deploy/terraform/cloudscheduler.tf for the new scheduler configuration.

// HandleShutdown manages graceful shutdown on signals.
func HandleShutdown(log *lgr.Logger, srv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Logf("WARN Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if srv != nil {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Logf("ERROR worker server shutdown error: %v", err)
		}
	}
}
