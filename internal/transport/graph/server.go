package graph

import (
	"log/slog"
	"net/http"
	"time"

	"hauslet/cmd/api/server/middleware"
	"hauslet/config"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/auth/service"
	bookingservice "hauslet/internal/modules/booking/service"
	businessservice "hauslet/internal/modules/business/service"
	calendarservice "hauslet/internal/modules/calendar/service"
	discoveryservice "hauslet/internal/modules/discovery/service"
	financeservice "hauslet/internal/modules/finance/service"
	interactionsservice "hauslet/internal/modules/interactions/service"
	leadsservice "hauslet/internal/modules/leads/service"
	messagingservice "hauslet/internal/modules/messaging/service"
	paymentsservice "hauslet/internal/modules/payments/service"
	pricingservice "hauslet/internal/modules/pricing/service"
	profileservice "hauslet/internal/modules/profile/service"
	promotionservice "hauslet/internal/modules/promotions/service"
	propertyservice "hauslet/internal/modules/property/service"
	reviewservice "hauslet/internal/modules/review/service"
	verificationservice "hauslet/internal/modules/verification/service"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	"hauslet/internal/platform/events"
	"hauslet/internal/platform/ratelimit"
	"hauslet/internal/platform/xchange"
	"hauslet/internal/transport/graph/loaders"
	"hauslet/internal/transport/graph/viewer"
	localization "hauslet/internal/transport/middleware/localization"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/ast"
)

func SetupGraphQL(r chi.Router,
	authService service.AuthService,
	profileService profileservice.ProfileService,
	propertyService propertyservice.PropertyService,
	businessService businessservice.BusinessService,
	paymentsService paymentsservice.PaymentService,
	pricingService pricingservice.PricingService,
	financeService financeservice.FinanceService,
	payoutService financeservice.PayoutService,
	bookingService bookingservice.BookingService,
	calendarService calendarservice.CalendarService,
	wishlistService wishlistservice.WishlistService,
	reviewService reviewservice.ReviewService,
	promotionService promotionservice.PromotionService,
	subscriptionService promotionservice.SubscriptionService,
	usageService promotionservice.UsageService,
	leadService leadsservice.LeadService,
	interactionsTracker interactionsservice.TrackerService,
	interactionsReader interactionsservice.ReaderService,
	discoveryService discoveryservice.DiscoveryService,
	verificationService verificationservice.VerificationService,
	messagingService messagingservice.MessagingService,
	tenantSlugMiddleware func(http.Handler) http.Handler,
	fxClient xchange.XChange,
	rateLimiter ratelimit.Limiter,
	eventSubscriber *events.Subscriber,
	cfg *config.GlobalConfig,
	log *slog.Logger) {

	srv := handler.New(
		NewExecutableSchema(Config{
			Resolvers: NewResolver(
				authService,
				profileService,
				propertyService,
				businessService,
				paymentsService,
				pricingService,
				financeService,
				payoutService,
				bookingService,
				calendarService,
				wishlistService,
				reviewService,
				promotionService,
				subscriptionService,
				usageService,
				leadService,
				interactionsTracker,
				interactionsReader,
				discoveryService,
				verificationService,
				messagingService,
				fxClient,
				eventSubscriber,
				cfg,
				log,
			),
			Complexity: NewComplexityRoot(defaultMaxListLimit),
		}),
	)

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || // Allow non-browser clients
					origin == r.Header.Get("Host") || // Same origin
					origin == cfg.App.Client || // Configured client origin
					cfg.App.Env != "production" // Allow all in non-production
			},
		},
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     32 << 20, // 32MB
		MaxUploadSize: 50 << 20, // 50MB
	})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	if cfg.App.Env != "production" {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	authMiddleware := authService.OAuthService().Middleware()
	r.Group(func(r chi.Router) {
		r.Use(authmiddleware.RequestContext)

		// Optional: Middleware to extract User from JWT and put in Context
		r.Use(authMiddleware.Trace)
		// Capture viewer info for resolvers (optional auth).
		r.Use(viewer.WithContext)
		// Resolve X-Tenant-Slug to business context and enforce membership.
		if tenantSlugMiddleware != nil {
			r.Use(tenantSlugMiddleware)
		}
		// Preferred currency for price localization.
		r.Use(localization.WithPreferredCurrency)
		// DataLoaders to batch profile and property fetches.
		r.Use(loaders.Middleware(profileService, propertyService, bookingService, reviewService))

		// Apply rate limiting in production
		if cfg.App.Env == "production" && rateLimiter != nil {
			policy := middleware.RateLimitPolicy{
				Keys: func(r *http.Request) []ratelimit.LimitKey {
					ip := middleware.ClientIP(r)
					if ip == "" {
						return nil
					}
					return []ratelimit.LimitKey{
						{
							Type:   ratelimit.KeyTypeIP,
							Value:  ip,
							Limit:  60,
							Window: time.Minute,
						},
					}
				},
			}
			r.Use(middleware.RateLimitWithLimiter(rateLimiter, policy))
			log.Info("GraphQL rate limiting enabled: 60 req/min")
		}

		// The Query Endpoint
		r.Handle("/query", srv)
	})

	if cfg.App.Env != "production" {
		r.Handle("/playground", playground.Handler("Hauslet GraphQL", "/query"))
		log.Info("GraphQL Playground available at /playground")
	}
}
