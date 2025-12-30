package server

import (
	"context"
	"fmt"
	"hauslet/config"
	authrepository "hauslet/internal/modules/auth/repository"
	authservice "hauslet/internal/modules/auth/service"
	authsession "hauslet/internal/modules/auth/session"
	bookingnotification "hauslet/internal/modules/booking/notification"
	bookingadapters "hauslet/internal/modules/booking/port/adapters"
	bookinghooks "hauslet/internal/modules/booking/port/hooks"
	bookingrepository "hauslet/internal/modules/booking/repository"
	bookingservice "hauslet/internal/modules/booking/service"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	businessnotification "hauslet/internal/modules/business/notification"
	businesshooks "hauslet/internal/modules/business/port/hooks"
	businessrepository "hauslet/internal/modules/business/repository"
	businessservice "hauslet/internal/modules/business/service"
	calendarhttp "hauslet/internal/modules/calendar/port/http"
	calendarrepository "hauslet/internal/modules/calendar/repository"
	calendarservice "hauslet/internal/modules/calendar/service"
	financenotification "hauslet/internal/modules/finance/notification"
	financehooks "hauslet/internal/modules/finance/port/hooks"
	financerepository "hauslet/internal/modules/finance/repository"
	financeservice "hauslet/internal/modules/finance/service"
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
	reviewnotification "hauslet/internal/modules/review/notification"
	reviewhooks "hauslet/internal/modules/review/port/hooks"
	reviewrepository "hauslet/internal/modules/review/repository"
	reviewservice "hauslet/internal/modules/review/service"
	wishlistrepository "hauslet/internal/modules/wishlist/repository"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	aiembeddings "hauslet/internal/platform/ai/embeddings"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/payment"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/storage"
	"hauslet/internal/platform/xchange"

	authhttp "hauslet/internal/modules/auth/port/http"

	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

// InfrastructureDependencies holds the foundational dependencies needed to bootstrap the application
type InfrastructureDependencies struct {
	DB          *gorm.DB
	Redis       *redis.RedisClient
	Queue       *queue.Client
	R2          *storage.R2Storage
	Logger      *lgr.Logger
	Config      *config.GlobalConfig
	EmailClient *email.Client
}

// Container holds all initialized services, repositories, and handlers for the application
type Container struct {
	// Infrastructure
	DB     *gorm.DB
	Redis  *redis.RedisClient
	Queue  *queue.Client
	R2     *storage.R2Storage
	Logger *lgr.Logger
	Config *config.GlobalConfig

	// Platform Services
	EmailClient   *email.Client
	PaymentClient *payment.Client
	FXClient      *xchange.Client
	EmbeddingAI   *aiembeddings.Client

	// Module Services
	AuthSvc       authservice.AuthService
	ProfileSvc    profileservice.ProfileService
	BusinessSvc   businessservice.BusinessService
	PropertySvc   propertyservice.PropertyService
	BookingSvc    bookingservice.BookingService
	PaymentsSvc   paymentsservice.PaymentService
	FinanceSvc    financeservice.FinanceService
	PayoutSvc     financeservice.PayoutService
	CalendarSvc   calendarservice.CalendarService
	PricingSvc    pricingservice.PricingService
	WishlistSvc   wishlistservice.WishlistService
	ReviewSvc     reviewservice.ReviewService
	ModerationSvc moderationservice.ModerationService

	// HTTP Handlers
	AuthHTTP           *authhttp.HTTPHandler
	PropertyHTTP       *propertyhttp.HTTPHandler
	CalendarHTTP       *calendarhttp.HTTPHandler
	PaymentWebhookHTTP *paymentshttp.WebhookHandler

	// Middleware
	BusinessMW *businessmiddleware.Middleware
}

// NewContainer initializes all application dependencies in the correct topological order
func NewContainer(ctx context.Context, deps InfrastructureDependencies) (*Container, error) {
	c := &Container{
		DB:          deps.DB,
		Redis:       deps.Redis,
		Queue:       deps.Queue,
		R2:          deps.R2,
		Logger:      deps.Logger,
		Config:      deps.Config,
		EmailClient: deps.EmailClient,
	}

	// Initialize platform services
	if err := c.initPlatformServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize platform services: %w", err)
	}

	// Initialize module services in dependency order
	if err := c.initModeration(); err != nil {
		return nil, fmt.Errorf("failed to initialize moderation: %w", err)
	}

	if err := c.initProfile(); err != nil {
		return nil, fmt.Errorf("failed to initialize profile: %w", err)
	}

	if err := c.initBusiness(); err != nil {
		return nil, fmt.Errorf("failed to initialize business: %w", err)
	}

	if err := c.initPayments(); err != nil {
		return nil, fmt.Errorf("failed to initialize payments: %w", err)
	}

	if err := c.initProperty(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize property: %w", err)
	}

	if err := c.initCalendar(); err != nil {
		return nil, fmt.Errorf("failed to initialize calendar: %w", err)
	}

	if err := c.initPricing(); err != nil {
		return nil, fmt.Errorf("failed to initialize pricing: %w", err)
	}

	if err := c.initFinance(); err != nil {
		return nil, fmt.Errorf("failed to initialize finance: %w", err)
	}

	if err := c.initBooking(); err != nil {
		return nil, fmt.Errorf("failed to initialize booking: %w", err)
	}

	if err := c.initPayout(); err != nil {
		return nil, fmt.Errorf("failed to initialize payout: %w", err)
	}

	if err := c.initWishlist(); err != nil {
		return nil, fmt.Errorf("failed to initialize wishlist: %w", err)
	}

	if err := c.initReview(); err != nil {
		return nil, fmt.Errorf("failed to initialize review: %w", err)
	}

	if err := c.initAuth(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	if err := c.initHTTPHandlers(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize HTTP handlers: %w", err)
	}

	return c, nil
}

// initPlatformServices initializes payment, FX, and AI clients
func (c *Container) initPlatformServices(ctx context.Context) error {
	// Initialize payment client
	paymentFactory := payment.NewProviderFactory(c.Config.Services.Payment)
	c.PaymentClient = payment.New(paymentFactory)

	// Initialize FX client
	fxProvider := xchange.NewExchangeRateAdapter(
		c.Config.Services.FX.APIKey,
		c.Config.Services.FX.BaseURL,
		*c.Redis,
	)
	c.FXClient = xchange.New(fxProvider)

	// Initialize AI embeddings client (optional - log warning if fails)
	if provider, err := aiembeddings.NewGeminiProvider(ctx, c.Config.Services.Gemini); err != nil {
		c.Logger.Logf("WARN failed to initialize embedding client: %v", err)
		c.EmbeddingAI = nil
	} else {
		c.EmbeddingAI = aiembeddings.New(provider)
	}

	return nil
}

// initModeration initializes the moderation service
func (c *Container) initModeration() error {
	aiModerationSubject := c.Config.YAML.Queue.Subjects["ai_moderation"]
	moderationRepo := moderationrepository.NewModerationRepository(c.DB)
	c.ModerationSvc = moderationservice.NewModerationService(
		moderationRepo,
		nil, // AI client not needed on API path; only enqueue/persist
		c.Queue,
		aiModerationSubject,
		nil, nil, nil,
		c.Logger,
	)
	return nil
}

// initProfile initializes the profile service and its adapters
func (c *Container) initProfile() error {
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	profileRepo := profilerepository.NewProfileRepository(c.DB)
	profileNotificationService := profilenotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	moderationAdapter := moderationhooks.NewModerationAdapter(c.ModerationSvc)

	c.ProfileSvc = profileservice.NewProfileService(
		profileRepo,
		c.R2,
		moderationAdapter,
		profileNotificationService,
		c.Logger,
	)

	return nil
}

// initBusiness initializes the business service and middleware
func (c *Container) initBusiness() error {
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	businessRepo := businessrepository.NewBusinessRepository(c.DB)
	businessNotificationService := businessnotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	businessProfileAdapter := profileport.NewBusinessProfileAdapter(c.ProfileSvc)

	c.BusinessSvc = businessservice.NewBusinessService(
		businessRepo,
		businessNotificationService,
		businessProfileAdapter,
		c.Logger,
	)

	c.BusinessMW = businessmiddleware.NewMiddleware(c.BusinessSvc, c.Logger)

	return nil
}

// initPayments initializes the payments service
func (c *Container) initPayments() error {
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	paymentsRepo := paymentsrepository.NewRepository(c.DB)

	paymentsProfileAdapter := profileport.NewPaymentsProfileAdapter(c.ProfileSvc)
	paymentsBusinessAdapter := businesshooks.NewPaymentsBusinessAdapter(c.BusinessSvc)

	paymentsNotificationSvc := paymentsnotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		paymentsProfileAdapter,
		paymentsBusinessAdapter,
		c.Logger,
	)

	c.PaymentsSvc = paymentsservice.NewPaymentService(
		paymentsRepo,
		c.PaymentClient,
		paymentsNotificationSvc,
		c.Logger,
	)

	return nil
}

// initProperty initializes the property service
func (c *Container) initProperty(ctx context.Context) error {
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	thumbnailSubject := c.Config.YAML.Queue.Subjects["media_thumbnail"]

	propertyRepo := propertyrepository.NewPropertyRepository(c.DB)
	propertyNotificationService := propertynotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	propertyProfileAdapter := profileport.NewPropertyProfileAdapter(c.ProfileSvc)
	moderationAdapter := moderationhooks.NewModerationAdapter(c.ModerationSvc)

	c.PropertySvc = propertyservice.NewPropertyService(
		propertyRepo,
		propertyNotificationService,
		propertyProfileAdapter,
		c.R2,
		c.Queue,
		thumbnailSubject,
		moderationAdapter,
		*c.Redis,
		c.EmbeddingAI,
		c.Logger,
		businessmiddleware.NewPropertyAuthHelper(c.BusinessSvc),
		c.BusinessSvc,
	)

	return nil
}

// initCalendar initializes the calendar service
func (c *Container) initCalendar() error {
	calendarRepo := calendarrepository.NewCalendarRepository(c.DB)
	calendarHooksAdapter := propertyhooks.NewCalendarHooksAdapter(c.PropertySvc)

	c.CalendarSvc = calendarservice.NewCalendarService(
		calendarRepo,
		*c.Redis,
		calendarHooksAdapter,
		c.Logger,
	)

	return nil
}

// initPricing initializes the pricing service
func (c *Container) initPricing() error {
	pricingRepo := pricingrepository.NewPricingRepository(c.DB)
	pricingHooksAdapter := propertyhooks.NewPricingHooksAdapter(c.PropertySvc)

	c.PricingSvc = pricingservice.NewPricingService(
		pricingRepo,
		*c.Redis,
		pricingHooksAdapter,
		c.Logger,
		c.Config.YAML.Platform,
	)

	return nil
}

// initFinance initializes the finance service (partial - before booking)
func (c *Container) initFinance() error {
	bookingRepo := bookingrepository.NewBookingRepository(c.DB)
	propertyRepo := propertyrepository.NewPropertyRepository(c.DB)

	// Initialize property hooks adapter for booking querier
	propertyHooksAdapter := bookinghooks.NewPropertyHooksAdapter(propertyRepo)

	// Initialize booking party querier for finance authorization
	bookingPartyQuerier := financehooks.NewFinanceBookingAdapter(bookingRepo, propertyHooksAdapter)

	// Initialize finance repositories
	financeWalletRepo := financerepository.NewWalletRepository(c.DB)
	financeLedgerRepo := financerepository.NewLedgerRepository(c.DB)
	financeTransactionRepo := financerepository.NewTransactionRepository(c.DB)
	financeDisbursementRepo := financerepository.NewDisbursementRepository(c.DB)
	financeDisputeRepo := financerepository.NewDisputeRepository(c.DB)
	financeReconciliationRepo := financerepository.NewReconciliationRepository(c.DB)

	c.FinanceSvc = financeservice.NewFinanceService(
		financeWalletRepo,
		financeLedgerRepo,
		financeTransactionRepo,
		financeDisbursementRepo,
		financeDisputeRepo,
		financeReconciliationRepo,
		bookingPartyQuerier,
		c.DB,
		c.Logger,
	)

	return nil
}

// initBooking initializes the booking service
func (c *Container) initBooking() error {
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	bookingRepo := bookingrepository.NewBookingRepository(c.DB)
	propertyRepo := propertyrepository.NewPropertyRepository(c.DB)

	bookingNotificationService := bookingnotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	// Initialize adapters
	paymentAdapter := bookingadapters.NewPaymentServiceAdapter(c.PaymentsSvc, c.Logger)
	propertyHooksAdapter := bookinghooks.NewPropertyHooksAdapter(propertyRepo)
	bookingProfileAdapter := profileport.NewBookingProfileAdapter(c.ProfileSvc)
	financeHooksAdapter := financehooks.NewPaymentHooksAdapter(c.FinanceSvc)

	// Initialize review hooks adapter (needs to be created before booking service)
	reviewUserAdapter := reviewhooks.NewReviewUserAdapter(c.ProfileSvc)
	reviewNotificationService := reviewnotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	reviewBookingHooksAdapter := reviewhooks.NewReviewBookingHooksAdapter(
		bookingRepo,
		propertyRepo,
		c.BusinessSvc,
		reviewUserAdapter,
		reviewNotificationService,
		c.Config.YAML.Platform.Reviews.ReviewWindowDays,
		c.Logger,
	)

	c.BookingSvc = bookingservice.NewBookingService(
		bookingRepo,
		c.CalendarSvc,
		c.PricingSvc,
		paymentAdapter,
		propertyHooksAdapter,
		bookingProfileAdapter,
		bookingNotificationService,
		c.Queue,
		c.Config.YAML.Queue.Subjects["booking_refund"],
		financeHooksAdapter,
		reviewBookingHooksAdapter,
		c.Config.YAML.Platform,
		c.Logger,
	)

	return nil
}

// initPayout initializes the payout service (after booking)
func (c *Container) initPayout() error {
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	bookingRepo := bookingrepository.NewBookingRepository(c.DB)
	paymentsRepo := paymentsrepository.NewRepository(c.DB)

	financeWalletRepo := financerepository.NewWalletRepository(c.DB)
	financeLedgerRepo := financerepository.NewLedgerRepository(c.DB)
	financeTransactionRepo := financerepository.NewTransactionRepository(c.DB)
	financeDisbursementRepo := financerepository.NewDisbursementRepository(c.DB)

	financeNotificationSvc := financenotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	bookingQuerierAdapter := financehooks.NewBookingQuerierAdapter(bookingRepo)
	bookingPayoutHooksAdapter := bookinghooks.NewPayoutHooksAdapter(c.BookingSvc)
	financeProfileAdapter := financehooks.NewFinanceProfileAdapter(c.ProfileSvc)

	c.PayoutSvc = financeservice.NewPayoutService(
		financeWalletRepo,
		financeLedgerRepo,
		financeTransactionRepo,
		financeDisbursementRepo,
		paymentsRepo,
		bookingQuerierAdapter,
		financeNotificationSvc,
		bookingPayoutHooksAdapter,
		c.PaymentClient,
		financeProfileAdapter,
		c.Config.YAML.Platform,
		c.DB,
		c.Logger,
	)

	return nil
}

// initWishlist initializes the wishlist service
func (c *Container) initWishlist() error {
	wishlistRepo := wishlistrepository.NewWishlistRepository(c.DB)
	wishListListAdapter := propertyhooks.NewWishlistHooksAdapter(c.PropertySvc)

	c.WishlistSvc = wishlistservice.NewWishlistService(
		wishlistRepo,
		*c.Redis,
		wishListListAdapter,
		c.Logger,
	)

	return nil
}

// initReview initializes the review service
func (c *Container) initReview() error {
	bookingRepo := bookingrepository.NewBookingRepository(c.DB)
	propertyRepo := propertyrepository.NewPropertyRepository(c.DB)
	emailSubject := c.Config.YAML.Queue.Subjects["email"]

	reviewRepo := reviewrepository.NewReviewRepository(c.DB)
	responseRepo := reviewrepository.NewResponseRepository(c.DB)
	statsRepo := reviewrepository.NewStatsRepository(c.DB)

	reviewNotificationService := reviewnotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	reviewBookingQuerier := reviewhooks.NewBookingQuerierAdapter(bookingRepo, propertyRepo)
	bookingReviewHooksAdapter := bookinghooks.NewReviewHooksAdapter(bookingRepo)
	reviewUserAdapter := reviewhooks.NewReviewUserAdapter(c.ProfileSvc)

	c.ReviewSvc = reviewservice.NewReviewService(
		reviewRepo,
		responseRepo,
		statsRepo,
		reviewNotificationService,
		reviewBookingQuerier,
		bookingReviewHooksAdapter,
		reviewUserAdapter,
		c.ModerationSvc,
		c.Logger,
	)

	return nil
}

// initAuth initializes the auth service
func (c *Container) initAuth(ctx context.Context) error {
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	sessionStore := authsession.NewSessionStore(*c.Redis)
	authRepo := authrepository.NewAuthRepository(c.DB, sessionStore)
	authProfileAdapter := profileport.NewAuthHooksAdapter(c.ProfileSvc, c.Config.Storage.R2.CDNHost)

	c.AuthSvc = authservice.NewAuthService(
		&c.Config.Auth,
		authRepo,
		c.Logger,
		c.EmailClient,
		*c.Redis,
		c.Queue,
		emailSubject,
		authProfileAdapter,
	)

	return nil
}

// initHTTPHandlers initializes HTTP handlers for auth, property, calendar, and payments
func (c *Container) initHTTPHandlers(ctx context.Context) error {
	// Initialize auth HTTP handler
	c.AuthHTTP = authhttp.NewHTTPHandler(ctx, c.AuthSvc, c.Logger)

	// Initialize property HTTP handler
	c.PropertyHTTP = propertyhttp.NewHTTPHandler(ctx, c.PropertySvc, c.Logger)

	// Initialize calendar HTTP handler
	c.CalendarHTTP = calendarhttp.NewHTTPHandler(ctx, c.CalendarSvc, c.Logger)

	// Initialize payment webhook handler
	bookingHooksAdapter := bookinghooks.NewBookingHooksAdapter(c.BookingSvc)
	financeHooksAdapter := financehooks.NewPaymentHooksAdapter(c.FinanceSvc)
	payoutHooksAdapter := financehooks.NewPayoutHooksAdapter(c.PayoutSvc)

	c.PaymentWebhookHTTP = paymentshttp.NewWebhookHandler(
		c.PaymentsSvc,
		c.PaymentClient,
		bookingHooksAdapter,
		financeHooksAdapter,
		payoutHooksAdapter,
		c.Queue,
		c.Config.YAML.Queue.Subjects["payment_webhook"],
		c.Logger,
	)

	return nil
}
