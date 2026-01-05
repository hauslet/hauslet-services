package setup

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"hauslet/config"
	bookingnotification "hauslet/internal/modules/booking/notification"
	bookinghooks "hauslet/internal/modules/booking/port/hooks"
	bookingrepository "hauslet/internal/modules/booking/repository"
	bookingservice "hauslet/internal/modules/booking/service"
	businessrepository "hauslet/internal/modules/business/repository"
	businessservice "hauslet/internal/modules/business/service"
	calendarnotification "hauslet/internal/modules/calendar/notification"
	calendarrepository "hauslet/internal/modules/calendar/repository"
	calendarschema "hauslet/internal/modules/calendar/repository/schema"
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
	promotionrepository "hauslet/internal/modules/promotions/repository"
	promotionservice "hauslet/internal/modules/promotions/service"
	propertynotification "hauslet/internal/modules/property/notification"
	propertyhooks "hauslet/internal/modules/property/port/hooks"
	propertyrepository "hauslet/internal/modules/property/repository"
	reviewnotification "hauslet/internal/modules/review/notification"
	reviewhooks "hauslet/internal/modules/review/port/hooks"
	reviewrepository "hauslet/internal/modules/review/repository"
	reviewservice "hauslet/internal/modules/review/service"
	verificationrepository "hauslet/internal/modules/verification/repository"
	verificationservice "hauslet/internal/modules/verification/service"
	"hauslet/internal/platform/payment"
	"hauslet/internal/queue"
	calendarjobs "hauslet/internal/queue/jobs/calendar"
	bookingHandler "hauslet/internal/transport/worker/handlers/booking"
	calendarHandler "hauslet/internal/transport/worker/handlers/calendar"
	emailHandler "hauslet/internal/transport/worker/handlers/emails"
	financeHandler "hauslet/internal/transport/worker/handlers/finance"
	listingHandler "hauslet/internal/transport/worker/handlers/listing"
	moderationHandler "hauslet/internal/transport/worker/handlers/moderation"
	paymentHandler "hauslet/internal/transport/worker/handlers/payments"
	promotionHandler "hauslet/internal/transport/worker/handlers/promotions"
	reviewHandler "hauslet/internal/transport/worker/handlers/review"
	verificationHandler "hauslet/internal/transport/worker/handlers/verification"
)

// RegisterHandlers builds and registers all queue handlers based on config.
func RegisterHandlers(infra *Infrastructure, cfg *config.GlobalConfig, log *slog.Logger) *queue.Registry {
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
	hasCalendarShowingReminders := qCfg["calendar_showing_reminders"] != ""
	hasCalendarOpenHouseReminders := qCfg["calendar_open_house_reminders"] != ""

	// Shared repos/services
	var propertyRepo propertyrepository.Repository
	var profileSvc profileservice.ProfileService
	var propertyProfileAdapter *profileport.PropertyProfileAdapter
	var profileRepo profilerepository.ProfileRepository

	if hasThumbnail || hasCleanup || hasModeration || hasPaymentWebhook || hasCalendarShowingReminders || hasCalendarOpenHouseReminders {
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
		businessSvc := businessservice.NewBusinessService(businessRepo, nil, nil, log, nil)

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
			nil, // TODO: replace nil with reviewHooks after review service is initialized
			log,
		)
		h := moderationHandler.NewAIModerationHandler(modService, log, qCfg["ai_moderation"])
		registry.Register(h)
	}

	// Booking expiry/completion handlers
	hasBookingExpiry := qCfg["booking_expiry"] != ""
	hasBookingCompletion := qCfg["booking_completion"] != ""
	hasBookingRefund := qCfg["booking_refund"] != ""
	hasBookingCheckInOut := qCfg["booking_checkin_out"] != ""
	if hasBookingExpiry || hasBookingCompletion || hasBookingRefund {
		// Initialize booking dependencies
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)

		if hasBookingExpiry || hasBookingCompletion {
			// Property/listing hooks (needed to get listing owner for calendar operations)
			if propertyRepo == nil {
				propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
			}
			listingHooks := bookinghooks.NewPropertyHooksAdapter(propertyRepo)

			var calendarSvc calendarservice.CalendarService
			var pricingSvc pricingservice.PricingService
			if hasBookingExpiry {
				// Calendar gateway (needed to cancel expired booking events)
				calendarRepo := calendarrepository.NewCalendarRepository(infra.DB)
				calendarHooksAdapter := propertyhooks.NewCalendarHooksRepoAdapter(propertyRepo)
				calendarProfileAdapter := profileport.NewCalendarProfileAdapter(profileSvc)
				calendarSvc = calendarservice.NewCalendarService(calendarRepo, infra.Cache, calendarHooksAdapter, calendarProfileAdapter, nil, log)

				// Pricing service with minimal dependencies
				pricingRepo := pricingrepository.NewPricingRepository(infra.DB)
				pricingSvc = pricingservice.NewPricingService(pricingRepo, nil, nil, log, cfg.YAML.Platform)
			}

			// Finance service and hooks (needed for booking completion)
			financeWalletRepo := financerepository.NewWalletRepository(infra.DB)
			financeLedgerRepo := financerepository.NewLedgerRepository(infra.DB)
			financeTransactionRepo := financerepository.NewTransactionRepository(infra.DB)
			financeDisbursementRepo := financerepository.NewDisbursementRepository(infra.DB)
			financeDisputeRepo := financerepository.NewDisputeRepository(infra.DB)
			financeReconciliationRepo := financerepository.NewReconciliationRepository(infra.DB)
			financeSvc := financeservice.NewFinanceService(
				financeWalletRepo,
				financeLedgerRepo,
				financeTransactionRepo,
				financeDisbursementRepo,
				financeDisputeRepo,
				financeReconciliationRepo,
				nil, // bookingPartyQuerier not needed for worker payment tasks
				infra.DB,
				log,
			)
			financeHooksAdapter := financehooks.NewPaymentHooksAdapter(financeSvc)

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
				financeHooksAdapter,
				nil, // review hooks not required for expiry checks
				cfg.YAML.Platform,
				log,
			)
			if hasBookingExpiry {
				h := bookingHandler.NewBookingExpiryCheckHandler(bookingSvc, log, qCfg["booking_expiry"])
				registry.Register(h)
			}
			if hasBookingCompletion {
				h := bookingHandler.NewBookingCompletionHandler(bookingSvc, log, qCfg["booking_completion"])
				registry.Register(h)
			}
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
				nil,
				nil,
				log,
			)
			paymentsSvc := paymentsservice.NewPaymentService(
				paymentsRepo,
				paymentClient,
				paymentsNotification,
				infra.Cache,
				log,
			)

			h := bookingHandler.NewBookingRefundHandler(bookingRepo, paymentsSvc, log, qCfg["booking_refund"])
			registry.Register(h)
		}
	}

	if hasBookingCheckInOut {
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)
		bookingSvc := bookingservice.NewBookingService(
			bookingRepo,
			nil, // calendar not required for check-in/out sync
			nil, // pricing not required for check-in/out sync
			nil, // payment gateway not required for check-in/out sync
			nil, // listing hooks not required for check-in/out sync
			nil, // profile provider not required for check-in/out sync
			nil, // notification service not required for check-in/out sync
			nil, // refund queue not required for check-in/out sync
			"",
			nil, // finance hooks not required for check-in/out sync
			nil, // review hooks not required for check-in/out sync
			cfg.YAML.Platform,
			log,
		)

		h := bookingHandler.NewBookingCheckInOutHandler(bookingSvc, log, qCfg["booking_checkin_out"])
		registry.Register(h)
	}

	if hasPaymentWebhook {
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)

		listingHooks := bookinghooks.NewPropertyHooksAdapter(propertyRepo)
		calendarRepo := calendarrepository.NewCalendarRepository(infra.DB)
		calendarHooksAdapter := propertyhooks.NewCalendarHooksRepoAdapter(propertyRepo)
		calendarProfileAdapter := profileport.NewCalendarProfileAdapter(profileSvc)
		calendarSvc := calendarservice.NewCalendarService(calendarRepo, infra.Cache, calendarHooksAdapter, calendarProfileAdapter, nil, log)

		bookingNotificationService := bookingnotification.NewNotificationService(
			infra.Email,
			infra.Queue,
			qCfg["email"],
			cfg.App.Client,
			log,
		)
		bookingProfileAdapter := profileport.NewBookingProfileAdapter(profileSvc)

		// Finance service and hooks (needed for booking lifecycle)
		financeWalletRepo := financerepository.NewWalletRepository(infra.DB)
		financeLedgerRepo := financerepository.NewLedgerRepository(infra.DB)
		financeTransactionRepo := financerepository.NewTransactionRepository(infra.DB)
		financeDisbursementRepo := financerepository.NewDisbursementRepository(infra.DB)
		financeDisputeRepo := financerepository.NewDisputeRepository(infra.DB)
		financeReconciliationRepo := financerepository.NewReconciliationRepository(infra.DB)
		financeSvc := financeservice.NewFinanceService(
			financeWalletRepo,
			financeLedgerRepo,
			financeTransactionRepo,
			financeDisbursementRepo,
			financeDisputeRepo,
			financeReconciliationRepo,
			nil, // bookingPartyQuerier not needed for worker expiry tasks
			infra.DB,
			log,
		)
		financeHooksAdapter := financehooks.NewPaymentHooksAdapter(financeSvc)

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
			financeHooksAdapter,
			nil, // review hooks not required for webhook processing
			cfg.YAML.Platform,
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
			nil,
			nil,
			log,
		)
		paymentsSvc := paymentsservice.NewPaymentService(
			paymentsRepo,
			paymentClient,
			paymentsNotification,
			infra.Cache,
			log,
		)

		// Note: financeWalletRepo, financeLedgerRepo, etc. already initialized above (line 228-241)
		// Reusing the existing finance service and hooks adapter

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
			nil, // promotion hooks not needed for worker webhook processing
			infra.Queue,
			qCfg["payment_webhook"],
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
			bookingQuerierAdapter, // Booking querier for finding bookings ready for payout
			financeNotificationSvc,
			bookingPayoutHooksAdapter, // Booking hooks for marking bookings as settled after payout
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

	// Reconciliation handler
	hasReconciliation := qCfg["finance_reconciliation"] != ""
	if hasReconciliation {
		// Initialize finance repositories
		financeWalletRepo := financerepository.NewWalletRepository(infra.DB)
		financeLedgerRepo := financerepository.NewLedgerRepository(infra.DB)
		financeTransactionRepo := financerepository.NewTransactionRepository(infra.DB)
		financeDisbursementRepo := financerepository.NewDisbursementRepository(infra.DB)
		financeDisputeRepo := financerepository.NewDisputeRepository(infra.DB)
		financeReconciliationRepo := financerepository.NewReconciliationRepository(infra.DB)

		// Initialize finance service
		financeSvc := financeservice.NewFinanceService(
			financeWalletRepo,
			financeLedgerRepo,
			financeTransactionRepo,
			financeDisbursementRepo,
			financeDisputeRepo,
			financeReconciliationRepo,
			nil, // bookingPartyQuerier not needed for reconciliation
			infra.DB,
			log,
		)

		h := financeHandler.NewReconciliationHandler(financeSvc, log, qCfg["finance_reconciliation"])
		registry.Register(h)
	}

	// Review handlers
	hasReviewStats := qCfg["review_stats"] != ""
	hasReviewStandoff := qCfg["review_standoff"] != ""
	hasReviewReminders := qCfg["review_reminders"] != ""
	if hasReviewStats || hasReviewStandoff || hasReviewReminders {
		// Initialize review dependencies
		reviewRepo := reviewrepository.NewReviewRepository(infra.DB)
		responseRepo := reviewrepository.NewResponseRepository(infra.DB)
		statsRepo := reviewrepository.NewStatsRepository(infra.DB)
		bookingRepo := bookingrepository.NewBookingRepository(infra.DB)

		// Initialize property repo if not already initialized
		if propertyRepo == nil {
			propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
		}

		// Initialize booking querier adapter for review service
		bookingQuerierAdapter := reviewhooks.NewBookingQuerierAdapter(bookingRepo, propertyRepo)

		// Initialize user querier adapter (wraps profile service)
		if profileSvc == nil {
			profileRepo = profilerepository.NewProfileRepository(infra.DB)
			profileSvc = profileservice.NewProfileService(profileRepo, infra.Storage, nil, nil, log)
		}
		userQuerierAdapter := reviewhooks.NewReviewUserAdapter(profileSvc)

		// Initialize review notification service
		reviewNotificationSvc := reviewnotification.NewNotificationService(
			infra.Email,
			infra.Queue,
			qCfg["email"],
			cfg.App.Client,
			log,
		)

		// Initialize review service (needed for stats and standoff handlers)
		reviewSvc := reviewservice.NewReviewService(
			reviewRepo,
			responseRepo,
			statsRepo,
			reviewNotificationSvc,
			bookingQuerierAdapter,
			nil, // booking hooks not needed for worker tasks
			userQuerierAdapter,
			nil, // moderation service not needed for worker tasks
			log,
		)

		if hasReviewStats {
			h := reviewHandler.NewStatsRecalculationHandler(reviewSvc, log, qCfg["review_stats"])
			registry.Register(h)
		}

		if hasReviewStandoff {
			h := reviewHandler.NewStandoffPublishHandler(reviewSvc, log, qCfg["review_standoff"])
			registry.Register(h)
		}

		if hasReviewReminders {
			h := reviewHandler.NewReminderHandler(
				bookingRepo,
				reviewRepo,
				bookingQuerierAdapter,
				userQuerierAdapter,
				reviewNotificationSvc,
				log,
				qCfg["review_reminders"],
			)
			registry.Register(h)
		}
	}

	// Calendar reminder handlers
	if hasCalendarShowingReminders || hasCalendarOpenHouseReminders {
		if propertyRepo == nil {
			propertyRepo = propertyrepository.NewPropertyRepository(infra.DB)
		}
		if profileSvc == nil {
			profileRepo = profilerepository.NewProfileRepository(infra.DB)
			profileSvc = profileservice.NewProfileService(profileRepo, infra.Storage, nil, nil, log)
		}

		calendarRepo := calendarrepository.NewCalendarRepository(infra.DB)
		listingInfoProvider := propertyhooks.NewCalendarHooksRepoAdapter(propertyRepo)
		calendarProfileAdapter := profileport.NewCalendarProfileAdapter(profileSvc)

		calendarNotificationSvc := calendarnotification.NewNotificationService(
			infra.Email,
			infra.Queue,
			qCfg["email"],
			cfg.App.Client,
			listingInfoProvider,
			calendarProfileAdapter,
			log,
		)

		if hasCalendarShowingReminders {
			h := calendarHandler.NewReminderHandler(
				calendarRepo,
				calendarNotificationSvc,
				log,
				qCfg["calendar_showing_reminders"],
				calendarschema.EventTypeShowing,
				calendarjobs.ShowingReminderJobType,
			)
			registry.Register(h)
		}

		if hasCalendarOpenHouseReminders {
			h := calendarHandler.NewReminderHandler(
				calendarRepo,
				calendarNotificationSvc,
				log,
				qCfg["calendar_open_house_reminders"],
				calendarschema.EventTypeOpenHouse,
				calendarjobs.OpenHouseReminderJobType,
			)
			registry.Register(h)
		}
	}

	// Promotion handlers
	hasPromotionExpiry := qCfg["promotion_expiry"] != ""
	hasSubscriptionBilling := qCfg["subscription_billing"] != ""
	if hasPromotionExpiry || hasSubscriptionBilling {
		// Initialize promotion repositories
		promoRepo := promotionrepository.NewListingPromotionRepository(infra.DB)
		subscriptionRepo := promotionrepository.NewAgentSubscriptionRepository(infra.DB)
		usageRepo := promotionrepository.NewUsageTrackingRepository(infra.DB)

		// Initialize usage service
		usageSvc := promotionservice.NewUsageService(
			usageRepo,
			&cfg.YAML.Promotion,
			infra.DB,
			log,
		)

		// Initialize payment service (needed for subscription billing)
		var paymentsSvc paymentsservice.PaymentService
		if hasSubscriptionBilling {
			paymentFactory := payment.NewProviderFactory(cfg.Services.Payment)
			paymentClient := payment.New(paymentFactory)
			paymentsRepo := paymentsrepository.NewRepository(infra.DB)
			paymentsNotification := paymentsnotification.NewNotificationService(
				infra.Email,
				infra.Queue,
				qCfg["email"],
				cfg.App.Client,
				nil,
				nil,
				log,
			)
			paymentsSvc = paymentsservice.NewPaymentService(
				paymentsRepo,
				paymentClient,
				paymentsNotification,
				infra.Cache,
				log,
			)
		}

		// Initialize subscription service
		subscriptionSvc := promotionservice.NewSubscriptionService(
			subscriptionRepo,
			usageSvc,
			paymentsSvc,
			&cfg.YAML.Promotion,
			infra.DB,
			log,
		)

		// Initialize promotion service
		promotionSvc := promotionservice.NewPromotionService(
			promoRepo,
			subscriptionRepo,
			usageSvc,
			paymentsSvc,
			&cfg.YAML.Promotion,
			infra.DB,
			log,
		)

		if hasPromotionExpiry {
			h := promotionHandler.NewPromotionExpiryHandler(promotionSvc, log, qCfg["promotion_expiry"])
			registry.Register(h)
		}

		if hasSubscriptionBilling {
			h := promotionHandler.NewSubscriptionBillingHandler(subscriptionSvc, log, qCfg["subscription_billing"])
			registry.Register(h)
		}
	}

	// Verification handlers
	hasVerificationSubmission := qCfg["verification_submission"] != ""
	hasVerificationSMS := qCfg["verification_sms"] != ""
	hasVerificationReconciliation := qCfg["verification_reconciliation"] != ""

	if hasVerificationSubmission || hasVerificationSMS || hasVerificationReconciliation {
		// Import verification dependencies
		verificationRepo := verificationrepository.NewVerificationRepo(infra.DB)

		if hasVerificationSubmission {
			// Submission handler needs KYC client and evidence store
			verificationSvc := verificationservice.NewVerificationService(
				verificationRepo,
				infra.KYC,
				infra.SMS,
				infra.EvidenceStore,
				infra.RateLimiter,
				infra.CircuitBreaker,
				infra.Redis,
				nil, // profile adapter not needed for worker
				nil, // business adapter not needed for worker
				nil, // queue client not needed in worker (handlers use queue directly)
				cfg,
				log,
			)

			h := verificationHandler.NewSubmissionHandler(
				verificationSvc,
				verificationRepo,
				infra.KYC,
				infra.EvidenceStore,
				log,
				qCfg["verification_submission"],
			)
			registry.Register(h)
		}

		if hasVerificationSMS {
			// SMS handler only needs SMS client
			h := verificationHandler.NewSMSHandler(
				infra.SMS,
				log,
				qCfg["verification_sms"],
			)
			registry.Register(h)
		}

		if hasVerificationReconciliation {
			// Reconciliation handler needs profile service
			if profileRepo == nil {
				profileRepo = profilerepository.NewProfileRepository(infra.DB)
			}
			if profileSvc == nil {
				profileSvc = profileservice.NewProfileService(profileRepo, infra.Storage, nil, nil, log)
			}

			h := verificationHandler.NewReconciliationHandler(
				verificationRepo,
				profileSvc,
				log,
				qCfg["verification_reconciliation"],
			)
			registry.Register(h)
		}
	}

	return registry
}

// Ticker functions (StartPeriodicCleanup, StartBookingExpiryCheck,
// StartPayoutProcessing, StartDisbursementRetry) - replaced by Cloud Scheduler.
// See deploy/terraform/cloudscheduler.tf for the new scheduler configuration.

// HandleShutdown manages graceful shutdown on signals.
func HandleShutdown(log *slog.Logger, srv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Warn("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if srv != nil {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("worker server shutdown error", "error", err)
		}
	}
}
