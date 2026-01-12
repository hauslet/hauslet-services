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
	discoverygraphql "hauslet/internal/modules/discovery/port/graphql"
	discoveryservice "hauslet/internal/modules/discovery/service"
	financegraphql "hauslet/internal/modules/finance/port/graphql"
	financeservice "hauslet/internal/modules/finance/service"
	interactionsgraphql "hauslet/internal/modules/interactions/port/graphql"
	interactionsservice "hauslet/internal/modules/interactions/service"
	leadsgraphql "hauslet/internal/modules/leads/port/graphql"
	leadsservice "hauslet/internal/modules/leads/service"
	paymentsgraphql "hauslet/internal/modules/payments/port/graphql"
	paymentsservice "hauslet/internal/modules/payments/service"
	pricinggraphql "hauslet/internal/modules/pricing/port/graphql"
	pricingservice "hauslet/internal/modules/pricing/service"
	profilegraphql "hauslet/internal/modules/profile/port/graphql"
	profileservice "hauslet/internal/modules/profile/service"
	promotiongraphql "hauslet/internal/modules/promotions/port/graphql"
	promotionservice "hauslet/internal/modules/promotions/service"
	propertygraphql "hauslet/internal/modules/property/port/graphql"
	propertyservice "hauslet/internal/modules/property/service"
	reviewgraphql "hauslet/internal/modules/review/port/graphql"
	reviewservice "hauslet/internal/modules/review/service"
	verificationgraphql "hauslet/internal/modules/verification/port/graphql"
	verificationservice "hauslet/internal/modules/verification/service"
	wishlistgraphql "hauslet/internal/modules/wishlist/port/graphql"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	"hauslet/internal/platform/events"
	"hauslet/internal/platform/xchange"
	"log/slog"
)

// Resolver wires domain-specific resolvers into gqlgen.
type Resolver struct {
	log                  *slog.Logger
	eventSubscriber      *events.Subscriber
	AuthResolver         *authgraphql.Resolver
	ProfileResolver      *profilegraphql.Resolver
	PropertyResolver     *propertygraphql.Resolver
	BusinessResolver     *businessgraphql.Resolver
	PaymentsResolver     *paymentsgraphql.Resolver
	PricingResolver      *pricinggraphql.Resolver
	FinanceResolver      *financegraphql.Resolver
	BookingResolver      *bookinggraphql.Resolver
	CalendarResolver     *calendargraphql.Resolver
	WishlistResolver     *wishlistgraphql.Resolver
	ReviewResolver       *reviewgraphql.Resolver
	PromotionResolver    *promotiongraphql.Resolver
	LeadResolver         *leadsgraphql.Resolver
	InteractionsResolver *interactionsgraphql.Resolver
	DiscoveryResolver    *discoverygraphql.Resolver
	VerificationResolver *verificationgraphql.Resolver
}

func NewResolver(
	authSvc authservice.AuthService,
	profileSvc profileservice.ProfileService,
	propertySvc propertyservice.PropertyService,
	businessSvc businessservice.BusinessService,
	paymentsSvc paymentsservice.PaymentService,
	pricingSvc pricingservice.PricingService,
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
	verificationSvc verificationservice.VerificationService,
	fxClient xchange.XChange,
	eventSubscriber *events.Subscriber,
	appCfg *cfg.GlobalConfig,
	log *slog.Logger,
) *Resolver {
	return &Resolver{
		log:                  log,
		eventSubscriber:      eventSubscriber,
		AuthResolver:         authgraphql.NewResolver(authSvc),
		ProfileResolver:      profilegraphql.NewResolver(profileSvc, &appCfg.Storage, log),
		PropertyResolver:     propertygraphql.NewResolver(propertySvc, &appCfg.Storage, fxClient, log),
		BusinessResolver:     businessgraphql.NewResolver(businessSvc, log),
		PaymentsResolver:     paymentsgraphql.NewResolver(paymentsSvc, log),
		PricingResolver:      pricinggraphql.NewResolver(pricingSvc, log),
		FinanceResolver:      financegraphql.NewResolver(financeSvc, payoutSvc, businessSvc, log),
		BookingResolver:      bookinggraphql.NewResolver(bookingSvc, fxClient, log),
		CalendarResolver:     calendargraphql.NewResolver(calendarSvc, log),
		WishlistResolver:     wishlistgraphql.NewResolver(wishlistSvc, log),
		ReviewResolver:       reviewgraphql.NewResolver(reviewSvc, log),
		PromotionResolver:    promotiongraphql.NewResolver(promotionSvc, subscriptionSvc, usageSvc, &appCfg.YAML.Promotion, log),
		LeadResolver:         leadsgraphql.NewResolver(leadSvc, log),
		InteractionsResolver: interactionsgraphql.NewResolver(interactionsTracker, interactionsReader, log),
		DiscoveryResolver:    discoverygraphql.NewResolver(discoverySvc, log),
		VerificationResolver: verificationgraphql.NewResolver(verificationSvc, log),
	}
}
