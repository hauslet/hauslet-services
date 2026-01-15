package server

import (
	"context"
	"fmt"
	"hauslet/config"
	"hauslet/internal/modules/auth/authorization"
	authrepository "hauslet/internal/modules/auth/repository"
	authservice "hauslet/internal/modules/auth/service"
	authsession "hauslet/internal/modules/auth/session"
	bookingnotification "hauslet/internal/modules/booking/notification"
	bookingadapters "hauslet/internal/modules/booking/port/adapters"
	bookinghooks "hauslet/internal/modules/booking/port/hooks"
	bookingrepository "hauslet/internal/modules/booking/repository"
	bookingservice "hauslet/internal/modules/booking/service"
	businessadapter "hauslet/internal/modules/business/adapter"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	businessnotification "hauslet/internal/modules/business/notification"
	businesshooks "hauslet/internal/modules/business/port/hooks"
	businessrepository "hauslet/internal/modules/business/repository"
	businessservice "hauslet/internal/modules/business/service"
	calendarnotification "hauslet/internal/modules/calendar/notification"
	calendarhttp "hauslet/internal/modules/calendar/port/http"
	calendarrepository "hauslet/internal/modules/calendar/repository"
	calendarservice "hauslet/internal/modules/calendar/service"
	discoveryhooks "hauslet/internal/modules/discovery/port/hooks"
	discoveryrepository "hauslet/internal/modules/discovery/repository"
	discoveryservice "hauslet/internal/modules/discovery/service"
	financenotification "hauslet/internal/modules/finance/notification"
	financehooks "hauslet/internal/modules/finance/port/hooks"
	financehttp "hauslet/internal/modules/finance/port/http"
	financerepository "hauslet/internal/modules/finance/repository"
	financeservice "hauslet/internal/modules/finance/service"
	interactionsrepository "hauslet/internal/modules/interactions/repository"
	interactionsservice "hauslet/internal/modules/interactions/service"
	leadhttp "hauslet/internal/modules/leads/port/http"
	leadsrepository "hauslet/internal/modules/leads/repository"
	leadsservice "hauslet/internal/modules/leads/service"
	messaginghooks "hauslet/internal/modules/messaging/port/hooks"
	messaginghttp "hauslet/internal/modules/messaging/port/http"
	vertexai "hauslet/internal/modules/messaging/port/vertexai"
	messagingrepository "hauslet/internal/modules/messaging/repository"
	messagingservice "hauslet/internal/modules/messaging/service"
	moderationhooks "hauslet/internal/modules/moderation/port/hooks"
	moderationrepository "hauslet/internal/modules/moderation/repository"
	moderationservice "hauslet/internal/modules/moderation/service"
	paymentsnotification "hauslet/internal/modules/payments/notification"
	paymentshttp "hauslet/internal/modules/payments/port/http"
	paymentsrepository "hauslet/internal/modules/payments/repository"
	paymentsservice "hauslet/internal/modules/payments/service"
	pricingrepository "hauslet/internal/modules/pricing/repository"
	pricingservice "hauslet/internal/modules/pricing/service"
	profileadapter "hauslet/internal/modules/profile/adapter"
	profilenotification "hauslet/internal/modules/profile/notification"
	profileport "hauslet/internal/modules/profile/port/hooks"
	profilerepository "hauslet/internal/modules/profile/repository"
	profileservice "hauslet/internal/modules/profile/service"
	promotionhooks "hauslet/internal/modules/promotions/port/hooks"
	promotionrepository "hauslet/internal/modules/promotions/repository"
	promotionservice "hauslet/internal/modules/promotions/service"
	propertynotification "hauslet/internal/modules/property/notification"
	propertyhooks "hauslet/internal/modules/property/port/hooks"
	propertyhttp "hauslet/internal/modules/property/port/http"
	propertyrepository "hauslet/internal/modules/property/repository"
	propertyservice "hauslet/internal/modules/property/service"
	reviewnotification "hauslet/internal/modules/review/notification"
	reviewhooks "hauslet/internal/modules/review/port/hooks"
	reviewhttp "hauslet/internal/modules/review/port/http"
	reviewrepository "hauslet/internal/modules/review/repository"
	reviewservice "hauslet/internal/modules/review/service"
	verificationhttp "hauslet/internal/modules/verification/port/http"
	verificationrepository "hauslet/internal/modules/verification/repository"
	verificationservice "hauslet/internal/modules/verification/service"
	wishlistrepository "hauslet/internal/modules/wishlist/repository"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	aiembeddings "hauslet/internal/platform/ai/embeddings"
	"hauslet/internal/platform/breaker"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/events"
	"hauslet/internal/platform/evidence"
	"hauslet/internal/platform/kyc"
	"hauslet/internal/platform/payment"
	"hauslet/internal/platform/queue"
	"hauslet/internal/platform/ratelimit"
	"hauslet/internal/platform/redis"
	"hauslet/internal/platform/sms"
	"hauslet/internal/platform/storage"
	"hauslet/internal/platform/xchange"
	"log/slog"
	"time"

	authhttp "hauslet/internal/modules/auth/port/http"

	redisclient "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// InfrastructureDependencies holds the foundational dependencies needed to bootstrap the application
type InfrastructureDependencies struct {
	DB             *gorm.DB
	Redis          *redis.RedisClient
	Queue          *queue.Client
	R2             *storage.R2Storage
	Logger         *slog.Logger
	Config         *config.GlobalConfig
	EmailClient    *email.Client
	KYC            *kyc.Client
	SMS            *sms.Client
	Evidence       evidence.Store
	RateLimiter    ratelimit.Limiter
	CircuitBreaker breaker.CircuitBreaker
}

// Container holds all initialized services, repositories, and handlers for the application
type Container struct {
	// Infrastructure
	DB     *gorm.DB
	Redis  *redis.RedisClient
	Queue  *queue.Client
	R2     *storage.R2Storage
	Logger *slog.Logger
	Config *config.GlobalConfig
	redis  redisclient.Client

	// Platform Services
	EmailClient     *email.Client
	PaymentClient   *payment.Client
	FXClient        *xchange.Client
	EmbeddingAI     *aiembeddings.Client
	KYCClient       *kyc.Client
	SMSClient       *sms.Client
	EvidenceStore   evidence.Store
	RateLimiter     ratelimit.Limiter
	CircuitBreaker  breaker.CircuitBreaker
	EventBroker     events.Broker
	EventPublisher  *events.Publisher
	EventSubscriber *events.Subscriber

	// Module Services
	AuthSvc            authservice.AuthService
	ProfileSvc         profileservice.ProfileService
	BusinessSvc        businessservice.BusinessService
	PropertySvc        propertyservice.PropertyService
	BookingSvc         bookingservice.BookingService
	PaymentsSvc        paymentsservice.PaymentService
	FinanceSvc         financeservice.FinanceService
	PayoutSvc          financeservice.PayoutService
	CalendarSvc        calendarservice.CalendarService
	PricingSvc         pricingservice.PricingService
	WishlistSvc        wishlistservice.WishlistService
	ReviewSvc          reviewservice.ReviewService
	ModerationSvc      moderationservice.ModerationService
	PromotionSvc       promotionservice.PromotionService
	SubscriptionSvc    promotionservice.SubscriptionService
	UsageSvc           promotionservice.UsageService
	LeadSvc            leadsservice.LeadService
	InteractionTracker interactionsservice.TrackerService
	InteractionReader  interactionsservice.ReaderService
	DiscoverySvc       discoveryservice.DiscoveryService
	VerificationSvc    verificationservice.VerificationService
	SupplyGate         authorization.SupplyGate
	MessagingSvc       messagingservice.MessagingService
	AISupportSvc       messagingservice.AISupportService

	// HTTP Handlers
	AuthHTTP                *authhttp.HTTPHandler
	PropertyHTTP            *propertyhttp.HTTPHandler
	CalendarHTTP            *calendarhttp.HTTPHandler
	LeadHTTP                *leadhttp.HTTPHandler
	MessagingHTTP           *messaginghttp.HTTPHandler
	FinanceHTTP             *financehttp.HTTPHandler
	ReviewHTTP              *reviewhttp.AdminHandler
	PaymentWebhookHTTP      *paymentshttp.WebhookHandler
	VerificationWebhookHTTP *verificationhttp.WebhookHandler

	// Middleware
	BusinessMW *businessmiddleware.Middleware
}

// NewContainer initializes all application dependencies in the correct topological order
func NewContainer(ctx context.Context, deps InfrastructureDependencies) (*Container, error) {
	c := &Container{
		DB:             deps.DB,
		Redis:          deps.Redis,
		Queue:          deps.Queue,
		R2:             deps.R2,
		Logger:         deps.Logger,
		Config:         deps.Config,
		EmailClient:    deps.EmailClient,
		KYCClient:      deps.KYC,
		SMSClient:      deps.SMS,
		EvidenceStore:  deps.Evidence,
		RateLimiter:    deps.RateLimiter,
		CircuitBreaker: deps.CircuitBreaker,
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

	if err := c.initVerification(); err != nil {
		return nil, fmt.Errorf("failed to initialize verification: %w", err)
	}

	if err := c.initPayments(); err != nil {
		return nil, fmt.Errorf("failed to initialize payments: %w", err)
	}

	if err := c.initPromotions(); err != nil {
		return nil, fmt.Errorf("failed to initialize promotions: %w", err)
	}

	if err := c.initSupplyGate(); err != nil {
		return nil, fmt.Errorf("failed to initialize supply gate: %w", err)
	}

	if err := c.initProperty(); err != nil {
		return nil, fmt.Errorf("failed to initialize property: %w", err)
	}

	if err := c.initDiscovery(); err != nil {
		return nil, fmt.Errorf("failed to initialize discovery: %w", err)
	}

	if err := c.initLeads(); err != nil {
		return nil, fmt.Errorf("failed to initialize leads: %w", err)
	}

	if err := c.initCalendar(); err != nil {
		return nil, fmt.Errorf("failed to initialize calendar: %w", err)
	}

	if err := c.initPricing(); err != nil {
		return nil, fmt.Errorf("failed to initialize pricing: %w", err)
	}

	if err := c.initAuth(); err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	if err := c.initFinance(); err != nil {
		return nil, fmt.Errorf("failed to initialize finance: %w", err)
	}

	if err := c.initBooking(); err != nil {
		return nil, fmt.Errorf("failed to initialize booking: %w", err)
	}

	if err := c.initMessaging(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize messaging: %w", err)
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

	if err := c.initInteractions(); err != nil {
		return nil, fmt.Errorf("failed to initialize interactions: %w", err)
	}

	if err := c.initHTTPHandlers(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize HTTP handlers: %w", err)
	}

	return c, nil
}

// initPlatformServices initializes payment, FX, AI clients, and event infrastructure
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
		c.Logger.Warn("failed to initialize embedding client", "error", err)
		c.EmbeddingAI = nil
	} else {
		c.EmbeddingAI = aiembeddings.New(provider)
	}

	// Initialize event infrastructure (Redis pub/sub for GraphQL subscriptions)
	if err := c.initEventInfrastructure(); err != nil {
		c.Logger.Warn("failed to initialize event infrastructure", "error", err)
		// Non-critical: subscriptions won't work but API can still function
	}

	return nil
}

// initEventInfrastructure initializes the event broker, publisher, and subscriber
func (c *Container) initEventInfrastructure() error {
	// Get the raw Redis client from the interface
	// The redis.RedisClient interface wraps *redis.Client
	rawClient, ok := (*c.Redis).(*redisclient.Client)
	if !ok {
		return fmt.Errorf("redis client is not *redis.Client")
	}

	// Initialize event broker
	c.EventBroker = events.NewRedisBroker(rawClient, c.Logger)

	// Initialize publisher (for services to emit events)
	c.EventPublisher = events.NewPublisher(c.EventBroker, c.Logger)

	// Initialize subscriber (for GraphQL resolvers)
	c.EventSubscriber = events.NewSubscriber(c.EventBroker, c.Logger)

	c.Logger.Info("✅ Event infrastructure initialized (Redis pub/sub)")
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

	// Create subscription adapter (optional, will be nil before initPromotions)
	var subscriptionAdapter profileservice.SubscriptionAdapter
	if c.SubscriptionSvc != nil {
		subscriptionAdapter = profileport.NewProfileSubscriptionAdapter(c.SubscriptionSvc)
	}

	c.ProfileSvc = profileservice.NewProfileService(
		profileRepo,
		c.R2,
		moderationAdapter,
		subscriptionAdapter, // Can be nil initially
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
		c.SupplyGate,
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
		*c.Redis,
		c.Logger,
	)

	return nil
}

// initPromotions initializes the promotion, subscription, and usage services
func (c *Container) initPromotions() error {
	// Initialize repositories
	promoRepo := promotionrepository.NewListingPromotionRepository(c.DB)
	subscriptionRepo := promotionrepository.NewAgentSubscriptionRepository(c.DB)
	usageRepo := promotionrepository.NewUsageTrackingRepository(c.DB)

	// Initialize usage service (no dependencies on other promotion services)
	c.UsageSvc = promotionservice.NewUsageService(
		usageRepo,
		&c.Config.YAML.Promotion,
		c.DB,
		c.Logger,
	)

	// Initialize profile adapter for promotions module
	profileAdapter := promotionhooks.NewPromotionProfileAdapter(c.ProfileSvc)

	// Initialize property adapter for promotions module
	propertyAdapter := promotionhooks.NewPromotionPropertyAdapter(c.PropertySvc)

	// Initialize subscription service
	c.SubscriptionSvc = promotionservice.NewSubscriptionService(
		subscriptionRepo,
		c.UsageSvc,
		c.PaymentsSvc,
		profileAdapter,
		propertyAdapter,
		&c.Config.YAML.Promotion,
		c.DB,
		c.Logger,
	)

	// Initialize promotion service
	c.PromotionSvc = promotionservice.NewPromotionService(
		promoRepo,
		subscriptionRepo,
		c.UsageSvc,
		c.PaymentsSvc,
		&c.Config.YAML.Promotion,
		c.DB,
		c.Logger,
	)

	// Reinitialize ProfileSvc with subscription adapter now that SubscriptionSvc is available
	// This allows profile service to auto-create free subscriptions when users select supply roles
	profileRepo := profilerepository.NewProfileRepository(c.DB)
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	profileNotificationService := profilenotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)
	moderationAdapter := moderationhooks.NewModerationAdapter(c.ModerationSvc)
	subscriptionAdapter := profileport.NewProfileSubscriptionAdapter(c.SubscriptionSvc)

	c.ProfileSvc = profileservice.NewProfileService(
		profileRepo,
		c.R2,
		moderationAdapter,
		subscriptionAdapter,
		profileNotificationService,
		c.Logger,
	)

	return nil
}

// initSupplyGate initializes the supply-side access gate.
func (c *Container) initSupplyGate() error {
	c.SupplyGate = authorization.NewSupplyGate(
		c.ProfileSvc,
		c.SubscriptionSvc,
		[]string{"admin", "root", "support"},
	)

	return nil
}

// initProperty initializes the property service
func (c *Container) initProperty() error {
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
		c.SubscriptionSvc,
		c.SupplyGate,
	)

	return nil
}

// initDiscovery initializes the discovery service
// initDiscovery initializes the discovery service
func (c *Container) initDiscovery() error {
	// Initialize repository
	discoveryRepo := discoveryrepository.NewDiscoveryRepository(c.DB)

	// Create hook adapters
	propertyHooks := discoveryhooks.NewPropertyDiscoveryAdapter(c.PropertySvc)
	promotionHooks := discoveryhooks.NewPromotionDiscoveryAdapter(c.PromotionSvc)

	// Initialize discovery service
	c.DiscoverySvc = discoveryservice.NewDiscoveryService(
		discoveryRepo,
		propertyHooks,
		promotionHooks,
		c.Logger,
	)

	return nil
}

// initLeads initializes the leads service
func (c *Container) initLeads() error {
	// Initialize repositories
	leadRepo := leadsrepository.NewLeadRepository(c.DB)
	leadEventRepo := leadsrepository.NewLeadEventRepository(c.DB)
	leadAssignmentRepo := leadsrepository.NewLeadAssignmentRepository(c.DB)

	// Initialize hooks adapters
	propertyHooks := leadsservice.NewPropertyHooksAdapter(c.PropertySvc)
	businessHooks := leadsservice.NewBusinessHooksAdapter(c.BusinessSvc)
	profileHooks := leadsservice.NewProfileHooksAdapter(c.ProfileSvc)

	rateLimitConfig := leadsservice.DefaultRateLimitConfig
	if c.Config != nil && c.Config.YAML != nil {
		leadsCfg := c.Config.YAML.RateLimit.Leads
		if leadsCfg.Window != "" {
			if window, err := time.ParseDuration(leadsCfg.Window); err == nil {
				rateLimitConfig.Window = window
			}
		}
		rateLimitConfig.Anonymous.PerEmail = leadsCfg.Anonymous.PerEmail
		rateLimitConfig.Anonymous.PerIP = leadsCfg.Anonymous.PerIP
		rateLimitConfig.Anonymous.PerListingEmail = leadsCfg.Anonymous.PerListingEmail
		rateLimitConfig.Anonymous.PerListingIP = leadsCfg.Anonymous.PerListingIP
		rateLimitConfig.Authenticated.PerUser = leadsCfg.Authenticated.PerUser
		rateLimitConfig.Authenticated.PerListingEmail = leadsCfg.Authenticated.PerListingEmail
		rateLimitConfig.Authenticated.PerListingIP = leadsCfg.Authenticated.PerListingIP
	}

	// Initialize lead service
	c.LeadSvc = leadsservice.NewLeadService(
		leadRepo,
		leadEventRepo,
		leadAssignmentRepo,
		propertyHooks,
		businessHooks,
		profileHooks,
		c.RateLimiter,
		rateLimitConfig,
		c.EventPublisher,
		c.Logger,
	)

	return nil
}

// initCalendar initializes the calendar service
func (c *Container) initCalendar() error {
	calendarRepo := calendarrepository.NewCalendarRepository(c.DB)
	calendarHooksAdapter := propertyhooks.NewCalendarHooksAdapter(c.PropertySvc)
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	calendarProfileAdapter := profileport.NewCalendarProfileAdapter(c.ProfileSvc)
	calendarNotificationService := calendarnotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		calendarHooksAdapter,
		calendarProfileAdapter,
		c.Logger,
	)

	c.CalendarSvc = calendarservice.NewCalendarService(
		calendarRepo,
		*c.Redis,
		calendarHooksAdapter,
		calendarProfileAdapter,
		calendarNotificationService,
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

	// Create auth adapter for admin notifications
	adminRoles := c.Config.YAML.Platform.Reconciliation.AdminRoles
	if len(adminRoles) == 0 {
		adminRoles = []string{"admin", "root"} // fallback default
	}
	authAdminAdapter := financehooks.NewFinanceAuthAdapter(c.AuthSvc, adminRoles)

	// Initialize finance notification service
	emailSubject := c.Config.YAML.Queue.Subjects["email"]
	financeNotificationSvc := financenotification.NewNotificationService(
		c.EmailClient,
		c.Queue,
		emailSubject,
		c.Config.App.Client,
		c.Logger,
	)

	c.FinanceSvc = financeservice.NewFinanceService(
		financeWalletRepo,
		financeLedgerRepo,
		financeTransactionRepo,
		financeDisbursementRepo,
		financeDisputeRepo,
		financeReconciliationRepo,
		bookingPartyQuerier,
		authAdminAdapter,
		financeNotificationSvc,
		c.Config.YAML.Platform,
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

// initMessaging initializes the messaging service and its hooks.
func (c *Container) initMessaging(ctx context.Context) error {
	convRepo := messagingrepository.NewConversationRepository(c.DB)
	msgRepo := messagingrepository.NewMessageRepository(c.DB)
	partRepo := messagingrepository.NewParticipantRepository(c.DB)

	leadRepo := leadsrepository.NewLeadRepository(c.DB)
	propertyRepo := propertyrepository.NewPropertyRepository(c.DB)
	bookingRepo := bookingrepository.NewBookingRepository(c.DB)

	leadHooks := messaginghooks.NewLeadHooksAdapter(leadRepo, propertyRepo, c.LeadSvc)
	bookingHooks := messaginghooks.NewBookingHooksAdapter(bookingRepo, propertyRepo)
	profileHooks := messaginghooks.NewProfileHooksAdapter(c.ProfileSvc)

	var aiSupport messagingservice.AISupportService
	aiCfg := c.Config.Services.VertexAI
	if aiCfg.ProjectID != "" && aiCfg.AgentID != "" {
		vertexCfg := c.Config.Services.Messaging
		client, err := vertexai.NewVertexAIClient(ctx, aiCfg.ProjectID, aiCfg.Location, aiCfg.AgentID, aiCfg.CredentialsPath)
		if err != nil {
			return fmt.Errorf("failed to initialize vertex ai client: %w", err)
		}
		aiSupport = messagingservice.NewVertexAISupportService(client, msgRepo, vertexCfg, c.Logger)
		if aiSupport == nil {
			c.Logger.Warn("vertex ai support service unavailable, ai flow disabled")
		} else {
			c.AISupportSvc = aiSupport
		}
	} else {
		c.Logger.Info("vertex ai disabled (missing configuration)", "project_id", aiCfg.ProjectID, "agent_id", aiCfg.AgentID)
	}
	c.AISupportSvc = aiSupport

	c.MessagingSvc = messagingservice.NewMessagingService(
		c.DB,
		convRepo,
		msgRepo,
		partRepo,
		aiSupport,
		leadHooks,
		bookingHooks,
		profileHooks,
		c.EventPublisher,
		c.EventSubscriber,
		c.R2,
		c.Logger,
	)

	// Start subscribing to lead events for auto-conversation creation
	if err := c.MessagingSvc.SubscribeToLeadEvents(context.Background(), c.EventSubscriber); err != nil {
		c.Logger.Error("failed to subscribe to lead events", "error", err)
		// Don't fail - event subscription is optional and can be retried later
	}

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
func (c *Container) initAuth() error {
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

// initInteractions initializes the interactions tracking and analytics services
func (c *Container) initInteractions() error {
	interactionRepo := interactionsrepository.NewInteractionRepository(c.DB)
	analyticsRepo := interactionsrepository.NewAnalyticsRepository(c.DB)
	botDetector := interactionsservice.NewBotDetector()

	c.InteractionTracker = interactionsservice.NewTrackerService(
		*c.Redis,
		botDetector,
		c.Logger,
	)

	c.InteractionReader = interactionsservice.NewReaderService(
		interactionRepo,
		analyticsRepo,
		c.Logger,
	)

	return nil
}

// initVerification initializes the verification service with adapters
func (c *Container) initVerification() error {
	verificationRepo := verificationrepository.NewVerificationRepo(c.DB)

	// Create adapters to notify profile and business modules on verification success
	profileVerificationAdapter := profileadapter.NewVerificationAdapter(c.ProfileSvc, c.Logger)
	businessVerificationAdapter := businessadapter.NewVerificationAdapter(c.BusinessSvc, c.Logger)

	c.VerificationSvc = verificationservice.NewVerificationService(
		verificationRepo,
		c.KYCClient,
		c.SMSClient,
		c.EvidenceStore,
		c.RateLimiter,
		c.CircuitBreaker,
		*c.Redis,
		profileVerificationAdapter,
		businessVerificationAdapter,
		c.Queue,
		c.Config,
		c.Logger,
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

	// Initialize lead HTTP handler
	c.LeadHTTP = leadhttp.NewHTTPHandler(ctx, c.LeadSvc, c.Logger)

	// Initialize messaging HTTP handler
	c.MessagingHTTP = messaginghttp.NewHTTPHandler(ctx, c.MessagingSvc, c.Logger)

	// Initialize finance HTTP handler (admin routes)
	c.FinanceHTTP = financehttp.NewHTTPHandler(ctx, c.FinanceSvc, c.PayoutSvc, c.Logger)

	// Initialize review HTTP handler (admin routes)
	c.ReviewHTTP = reviewhttp.NewAdminHandler(c.ReviewSvc, ctx, c.Logger)

	// Initialize payment webhook handler
	bookingHooksAdapter := bookinghooks.NewBookingHooksAdapter(c.BookingSvc)
	financeHooksAdapter := financehooks.NewPaymentHooksAdapter(c.FinanceSvc)
	payoutHooksAdapter := financehooks.NewPayoutHooksAdapter(c.PayoutSvc)
	promotionHooksAdapter := promotionhooks.NewPaymentHooks(
		c.PromotionSvc,
		c.SubscriptionSvc,
		c.Logger,
	)

	c.PaymentWebhookHTTP = paymentshttp.NewWebhookHandler(
		c.PaymentsSvc,
		c.PaymentClient,
		bookingHooksAdapter,
		financeHooksAdapter,
		payoutHooksAdapter,
		promotionHooksAdapter,
		c.Queue,
		c.Config.YAML.Queue.Subjects["payment_webhook"],
		c.Logger,
	)

	// Initialize verification webhook handler
	c.VerificationWebhookHTTP = verificationhttp.NewWebhookHandler(
		c.VerificationSvc,
		c.Logger,
	)

	return nil
}
