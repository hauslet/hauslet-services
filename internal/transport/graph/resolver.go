package graph

import (
	cfg "hauslet/config"
	authgraphql "hauslet/internal/modules/auth/port/graphql"
	authservice "hauslet/internal/modules/auth/service"
	bookinggraphql "hauslet/internal/modules/booking/port/graphql"
	bookingservice "hauslet/internal/modules/booking/service"
	businessgraphql "hauslet/internal/modules/business/port/graphql"
	businessservice "hauslet/internal/modules/business/service"
	calendargraphql "hauslet/internal/modules/calendar/port/graphql"
	calendarservice "hauslet/internal/modules/calendar/service"
	financegraphql "hauslet/internal/modules/finance/port/graphql"
	financeservice "hauslet/internal/modules/finance/service"
	interactionsgraphql "hauslet/internal/modules/interactions/port/graphql"
	interactionsservice "hauslet/internal/modules/interactions/service"
	discoverygraphql "hauslet/internal/modules/discovery/port/graphql"
	discoveryservice "hauslet/internal/modules/discovery/service"
	leadsgraphql "hauslet/internal/modules/leads/port/graphql"
	leadsservice "hauslet/internal/modules/leads/service"
	paymentsgraphql "hauslet/internal/modules/payments/port/graphql"
	paymentsservice "hauslet/internal/modules/payments/service"
	profilegraphql "hauslet/internal/modules/profile/port/graphql"
	profileservice "hauslet/internal/modules/profile/service"
	promotiongraphql "hauslet/internal/modules/promotions/port/graphql"
	promotionservice "hauslet/internal/modules/promotions/service"
	propertygraphql "hauslet/internal/modules/property/port/graphql"
	propertyservice "hauslet/internal/modules/property/service"
	reviewgraphql "hauslet/internal/modules/review/port/graphql"
	reviewservice "hauslet/internal/modules/review/service"
	wishlistgraphql "hauslet/internal/modules/wishlist/port/graphql"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	"hauslet/internal/platform/xchange"
	"log/slog"
)

// Resolver wires domain-specific resolvers into gqlgen.
type Resolver struct {
	log                  *slog.Logger
	AuthResolver         *authgraphql.Resolver
	ProfileResolver      *profilegraphql.Resolver
	PropertyResolver     *propertygraphql.Resolver
	BusinessResolver     *businessgraphql.Resolver
	PaymentsResolver     *paymentsgraphql.Resolver
	FinanceResolver      *financegraphql.Resolver
	BookingResolver      *bookinggraphql.Resolver
	CalendarResolver     *calendargraphql.Resolver
	WishlistResolver     *wishlistgraphql.Resolver
	ReviewResolver       *reviewgraphql.Resolver
	PromotionResolver    *promotiongraphql.Resolver
	LeadResolver         *leadsgraphql.Resolver
	InteractionsResolver *interactionsgraphql.Resolver
	DiscoveryResolver    *discoverygraphql.Resolver
}

func NewResolver(
	authSvc authservice.AuthService,
	profileSvc profileservice.ProfileService,
	propertySvc propertyservice.PropertyService,
	businessSvc businessservice.BusinessService,
	paymentsSvc paymentsservice.PaymentService,
	financeSvc financeservice.FinanceService,
	payoutSvc financeservice.PayoutService,
	bookingSvc bookingservice.BookingService,
	calendarSvc calendarservice.CalendarService,
	wishlistSvc wishlistservice.WishlistService,
	reviewSvc reviewservice.ReviewService,
	promotionSvc promotionservice.PromotionService,
	subscriptionSvc promotionservice.SubscriptionService,
	usageSvc promotionservice.UsageService,
	leadSvc leadsservice.LeadService,
	interactionsTracker interactionsservice.TrackerService,
	interactionsReader interactionsservice.ReaderService,
	discoverySvc discoveryservice.DiscoveryService,
	fxClient xchange.XChange,
	appCfg *cfg.GlobalConfig,
	log *slog.Logger,
) *Resolver {
	return &Resolver{
		log:                  log,
		AuthResolver:         authgraphql.NewResolver(authSvc),
		ProfileResolver:      profilegraphql.NewResolver(profileSvc, &appCfg.Storage, log),
		PropertyResolver:     propertygraphql.NewResolver(propertySvc, &appCfg.Storage, fxClient, log),
		BusinessResolver:     businessgraphql.NewResolver(businessSvc, log),
		PaymentsResolver:     paymentsgraphql.NewResolver(paymentsSvc, log),
		FinanceResolver:      financegraphql.NewResolver(financeSvc, payoutSvc, businessSvc, log),
		BookingResolver:      bookinggraphql.NewResolver(bookingSvc, fxClient, log),
		CalendarResolver:     calendargraphql.NewResolver(calendarSvc, log),
		WishlistResolver:     wishlistgraphql.NewResolver(wishlistSvc, log),
		ReviewResolver:       reviewgraphql.NewResolver(reviewSvc, log),
		PromotionResolver:    promotiongraphql.NewResolver(promotionSvc, subscriptionSvc, usageSvc, &appCfg.YAML.Promotion, log),
		LeadResolver:         leadsgraphql.NewResolver(leadSvc, log),
		InteractionsResolver: interactionsgraphql.NewResolver(interactionsTracker, interactionsReader, log),
		DiscoveryResolver:    discoverygraphql.NewResolver(discoverySvc, log),
	}
}
