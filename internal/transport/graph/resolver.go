package graph

import (
	cfg "hauslet/config"
	authgraphql "hauslet/internal/modules/auth/port/graphql"
	authservice "hauslet/internal/modules/auth/service"
	bookinggraphql "hauslet/internal/modules/booking/port/graphql"
	bookingservice "hauslet/internal/modules/booking/service"
	businessgraphql "hauslet/internal/modules/business/port/graphql"
	businessservice "hauslet/internal/modules/business/service"
	financegraphql "hauslet/internal/modules/finance/port/graphql"
	financeservice "hauslet/internal/modules/finance/service"
	paymentsgraphql "hauslet/internal/modules/payments/port/graphql"
	paymentsservice "hauslet/internal/modules/payments/service"
	profilegraphql "hauslet/internal/modules/profile/port/graphql"
	profileservice "hauslet/internal/modules/profile/service"
	propertygraphql "hauslet/internal/modules/property/port/graphql"
	propertyservice "hauslet/internal/modules/property/service"
	wishlistgraphql "hauslet/internal/modules/wishlist/port/graphql"
	wishlistservice "hauslet/internal/modules/wishlist/service"
	"hauslet/internal/platform/xchange"

	"github.com/go-pkgz/lgr"
)

// Resolver wires domain-specific resolvers into gqlgen.
type Resolver struct {
	log              *lgr.Logger
	AuthResolver     *authgraphql.Resolver
	ProfileResolver  *profilegraphql.Resolver
	PropertyResolver *propertygraphql.Resolver
	BusinessResolver *businessgraphql.Resolver
	PaymentsResolver *paymentsgraphql.Resolver
	FinanceResolver  *financegraphql.Resolver
	BookingResolver  *bookinggraphql.Resolver
	WishlistResolver *wishlistgraphql.Resolver
}

func NewResolver(
	authSvc authservice.AuthService,
	profileSvc profileservice.ProfileService,
	propertySvc propertyservice.Service,
	businessSvc businessservice.BusinessService,
	paymentsSvc paymentsservice.PaymentService,
	financeSvc financeservice.FinanceService,
	payoutSvc financeservice.PayoutService,
	bookingSvc bookingservice.BookingService,
	wishlistSvc wishlistservice.WishlistService,
	fxClient xchange.XChange,
	appCfg *cfg.GlobalConfig,
	log *lgr.Logger,
) *Resolver {
	return &Resolver{
		log:              log,
		AuthResolver:     authgraphql.NewResolver(authSvc),
		ProfileResolver:  profilegraphql.NewResolver(profileSvc, &appCfg.Storage, log),
		PropertyResolver: propertygraphql.NewResolver(propertySvc, &appCfg.Storage, fxClient, log),
		BusinessResolver: businessgraphql.NewResolver(businessSvc, log),
		PaymentsResolver: paymentsgraphql.NewResolver(paymentsSvc, log),
		FinanceResolver:  financegraphql.NewResolver(financeSvc, payoutSvc, log),
		BookingResolver:  bookinggraphql.NewResolver(bookingSvc, fxClient, log),
		WishlistResolver: wishlistgraphql.NewResolver(wishlistSvc, log),
	}
}
