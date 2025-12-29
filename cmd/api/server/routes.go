package server

import (
	"context"
	"hauslet/config"
	authhttp "hauslet/internal/modules/auth/port/http"
	authrepository "hauslet/internal/modules/auth/repository"
	authservice "hauslet/internal/modules/auth/service"
	authsession "hauslet/internal/modules/auth/session"
	bookingnotification "hauslet/internal/modules/booking/notification"
	bookingadapters "hauslet/internal/modules/booking/port/adapters"
	bookinghooks "hauslet/internal/modules/booking/port/hooks"
	bookingrepository "hauslet/internal/modules/booking/repository"
	bookingservice "hauslet/internal/modules/booking/service"
	financenotification "hauslet/internal/modules/finance/notification"
	financehooks "hauslet/internal/modules/finance/port/hooks"
	businesshooks "hauslet/internal/modules/business/port/hooks"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	businessnotification "hauslet/internal/modules/business/notification"
	businessrepository "hauslet/internal/modules/business/repository"
	businessservice "hauslet/internal/modules/business/service"
	calendarhttp "hauslet/internal/modules/calendar/port/http"
	financerepository "hauslet/internal/modules/finance/repository"
	financeservice "hauslet/internal/modules/finance/service"
	calendarrepository "hauslet/internal/modules/calendar/repository"
	calendarservice "hauslet/internal/modules/calendar/service"
	moderationhooks "hauslet/internal/modules/moderation/port/hooks"
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
	propertyhttp "hauslet/internal/modules/property/port/http"
	propertyrepository "hauslet/internal/modules/property/repository"
	propertyservice "hauslet/internal/modules/property/service"
	wishlistrepository "hauslet/internal/modules/wishlist/repository"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	aiembeddings "hauslet/internal/platform/ai/embeddings"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/payment"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/storage"
	"hauslet/internal/platform/xchange"
	"hauslet/internal/transport/graph"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

func setupRoutes(r chi.Router,
	ctx context.Context,
	db *gorm.DB, rds *redis.RedisClient,
	log *lgr.Logger,
	cfg *config.GlobalConfig,
	mC *email.Client,
	q *queue.Client,
	r2 *storage.R2Storage) {
	// Initialize auth service
	sessionStore := authsession.NewSessionStore(*rds)
	authRepo := authrepository.NewAuthRepository(db, sessionStore)
	emailSubject := cfg.YAML.Queue.Subjects["email"]

	// Initialize moderation service (AI client not needed on API path; only enqueue/persist).
	aiModerationSubject := cfg.YAML.Queue.Subjects["ai_moderation"]
	moderationRepo := moderationrepository.NewModerationRepository(db)
	moderationSvc := moderationservice.NewModerationService(moderationRepo, nil, q, aiModerationSubject, nil, nil, log)
	moderationAdapter := moderationhooks.NewModerationAdapter(moderationSvc)

	// Initialize profile service
	profileRepo := profilerepository.NewProfileRepository(db)
	profileNotificationService := profilenotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	profileService := profileservice.NewProfileService(profileRepo, r2, moderationAdapter, profileNotificationService, log)
	authProfileAdapter := profileport.NewAuthHooksAdapter(profileService, cfg.Storage.R2.CDNHost)
	businessProfileAdapter := profileport.NewBusinessProfileAdapter(profileService)
	bookingProfileAdapter := profileport.NewBookingProfileAdapter(profileService)
	financeProfileAdapter := financehooks.NewFinanceProfileAdapter(profileService)
	paymentsProfileAdapter := profileport.NewPaymentsProfileAdapter(profileService)

	// Initialize business service (before property service to enable business adapter)
	businessRepo := businessrepository.NewBusinessRepository(db)
	businessnotificationService := businessnotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	businessService := businessservice.NewBusinessService(
		businessRepo,
		businessnotificationService,
		businessProfileAdapter,
		log,
	)
	businessMW := businessmiddleware.NewMiddleware(businessService, log)
	paymentsBusinessAdapter := businesshooks.NewPaymentsBusinessAdapter(businessService)

	// Initialize payment platform client and services
	paymentFactory := payment.NewProviderFactory(cfg.Services.Payment)
	paymentClient := payment.New(paymentFactory)
	paymentsRepo := paymentsrepository.NewRepository(db)
	paymentsNotificationSvc := paymentsnotification.NewNotificationService(
		mC,
		q,
		emailSubject,
		cfg.App.Client,
		paymentsProfileAdapter,
		paymentsBusinessAdapter,
		log,
	)
	bookingNotificationService := bookingnotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	financeNotificationSvc := financenotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	paymentsService := paymentsservice.NewPaymentService(
		paymentsRepo,
		paymentClient,
		paymentsNotificationSvc,
		log,
	)

	// Initialize booking repository (service will be created after calendar/pricing are ready)
	bookingRepo := bookingrepository.NewBookingRepository(db)

	// Initialize property service with business adapter and moderation hooks
	propertyRepo := propertyrepository.NewPropertyRepository(db)
	thumbnailSubject := cfg.YAML.Queue.Subjects["media_thumbnail"]
	propertyNotificationService := propertynotification.NewNotificationService(mC, q, emailSubject, cfg.App.Client, log)
	propertyProfileAdapter := profileport.NewPropertyProfileAdapter(profileService)

	var embeddingClient *aiembeddings.Client
	if provider, err := aiembeddings.NewGeminiProvider(ctx, cfg.Services.Gemini); err != nil {
		log.Logf("WARN failed to initialize embedding client: %v", err)
	} else {
		embeddingClient = aiembeddings.New(provider)
	}

	propertyService := propertyservice.NewPropertyService(
		propertyRepo,
		propertyNotificationService,
		propertyProfileAdapter,
		r2, q,
		thumbnailSubject,
		moderationAdapter,
		*rds,
		embeddingClient,
		log,
		businessmiddleware.NewPropertyAuthHelper(businessService),
		businessService,
	)
	calendarRepo := calendarrepository.NewCalendarRepository(db)
	calendarHooksAdapter := propertyhooks.NewCalendarHooksAdapter(propertyService)
	calendarService := calendarservice.NewCalendarService(calendarRepo, *rds, calendarHooksAdapter, log)
	calendarHTTP := calendarhttp.NewHTTPHandler(ctx, calendarService, log)

	// Initialize pricing service for booking
	pricingRepo := pricingrepository.NewPricingRepository(db)
	pricingHooksAdapter := propertyhooks.NewPricingHooksAdapter(propertyService)
	pricingService := pricingservice.NewPricingService(
		pricingRepo,
		*rds,
		pricingHooksAdapter,
		log,
		cfg.YAML.Platform,
	)

	// Initialize payment adapter for booking
	paymentAdapter := bookingadapters.NewPaymentServiceAdapter(paymentsService, log)

	// Initialize property hooks adapter (needed for booking querier)
	propertyHooksAdapter := bookinghooks.NewPropertyHooksAdapter(propertyRepo)

	// Initialize booking party querier for finance authorization (uses repo, not service)
	bookingPartyQuerier := financehooks.NewFinanceBookingAdapter(bookingRepo, propertyHooksAdapter)

	// Initialize finance service (before booking service to enable finance hooks)
	financeWalletRepo := financerepository.NewWalletRepository(db)
	financeLedgerRepo := financerepository.NewLedgerRepository(db)
	financeTransactionRepo := financerepository.NewTransactionRepository(db)
	financeDisbursementRepo := financerepository.NewDisbursementRepository(db)
	financeDisputeRepo := financerepository.NewDisputeRepository(db)
	financeService := financeservice.NewFinanceService(
		financeWalletRepo,
		financeLedgerRepo,
		financeTransactionRepo,
		financeDisbursementRepo,
		financeDisputeRepo,
		bookingPartyQuerier,
		db,
		log,
	)

	// Initialize finance hooks adapter for booking lifecycle (before booking service)
	financeHooksAdapter := financehooks.NewPaymentHooksAdapter(financeService)

	// Initialize booking service with all dependencies
	bookingService := bookingservice.NewBookingService(
		bookingRepo,
		calendarService,
		pricingService,
		paymentAdapter,
		propertyHooksAdapter,
		bookingProfileAdapter,
		bookingNotificationService,
		q,
		cfg.YAML.Queue.Subjects["booking_refund"],
		financeHooksAdapter,
		cfg.YAML.Platform,
		log,
	)

	// Initialize booking querier adapter for payout processing
	bookingQuerierAdapter := financehooks.NewBookingQuerierAdapter(bookingRepo)

	// Initialize payout hooks adapter for marking bookings as settled
	bookingPayoutHooksAdapter := bookinghooks.NewPayoutHooksAdapter(bookingService)

	// Initialize payout service for automated host payouts
	payoutService := financeservice.NewPayoutService(
		financeWalletRepo,
		financeLedgerRepo,
		financeTransactionRepo,
		financeDisbursementRepo,
		paymentsRepo,               // PayoutDetailRepository for fetching host bank details
		bookingQuerierAdapter,      // Booking querier for finding bookings ready for payout
		financeNotificationSvc,     // Notification service for payout emails
		bookingPayoutHooksAdapter,  // Booking hooks for marking bookings as settled after payout
		paymentClient,
		financeProfileAdapter,      // Profile adapter for getting host email/name
		cfg.YAML.Platform,          // Platform config for commission rate and retry settings
		db,
		log,
	)

	// Initialize booking hooks adapter for payment lifecycle
	bookingHooksAdapter := bookinghooks.NewBookingHooksAdapter(bookingService)

	// Note: financeHooksAdapter already initialized earlier (line 189) for booking service

	// Initialize payout hooks adapter for transfer webhooks
	payoutHooksAdapter := financehooks.NewPayoutHooksAdapter(payoutService)

	// Initialize payment webhook handler with booking, finance, and payout hooks
	paymentsWebhookHandler := paymentshttp.NewWebhookHandler(
		paymentsService,
		paymentClient,
		bookingHooksAdapter,
		financeHooksAdapter,
		payoutHooksAdapter,
		q,
		cfg.YAML.Queue.Subjects["payment_webhook"],
		log,
	)

	// Initialize wishlist service
	wishlistRepo := wishlistrepository.NewWishlistRepository(db)
	wishListListAdapter := propertyhooks.NewWishlistHooksAdapter(propertyService)
	wishlistService := wishlistservice.NewWishlistService(wishlistRepo, *rds, wishListListAdapter, log)

	// Initialize auth service
	authService := authservice.NewAuthService(
		&cfg.Auth,
		authRepo,
		log, mC, *rds, q,
		emailSubject,
		authProfileAdapter,
	)

	// Initialize auth HTTP handler with context
	authHTTP := authhttp.NewHTTPHandler(ctx, authService, log)

	// Setup auth routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		authHTTP.SetupRoutesWithRateLimiting(r, *rds)
	} else {
		authHTTP.SetupRoutes(r)
	}

	// Initialize property HTTP handler with context
	propertyHTTP := propertyhttp.NewHTTPHandler(ctx, propertyService, log)

	// Setup property routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			propertyHTTP.SetupRoutesWithRateLimiting(r, authService, *rds, businessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			propertyHTTP.SetupRoutes(r, authService, businessMW)
		})
	}

	// Setup business routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			calendarHTTP.SetupRoutes(r, authService, businessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			calendarHTTP.SetupRoutesWithRateLimiting(r, authService, *rds, businessMW)
		})
	}

	// Setup payment webhook routes
	r.Post("/webhooks/paystack", paymentsWebhookHandler.HandlePaystackWebhook)

	// Initialize FX client
	fxProvider := xchange.NewExchangeRateAdapter(
		cfg.Services.FX.APIKey,
		cfg.Services.FX.BaseURL,
		*rds,
	)
	fxClient := xchange.New(fxProvider)

	// Setup GraphQL routes
	graph.SetupGraphQL(r,
		authService,
		profileService,
		propertyService,
		businessService,
		paymentsService,
		financeService,
		payoutService,
		bookingService,
		wishlistService,
		businessMW.Auth.WithTenantSlug,
		fxClient,
		cfg,
		log,
	)
}
