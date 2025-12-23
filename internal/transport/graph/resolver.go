package graph

import (
	cfg "hauslet/config"
	authgraphql "hauslet/internal/modules/auth/port/graphql"
	authservice "hauslet/internal/modules/auth/service"
	businessgraphql "hauslet/internal/modules/business/port/graphql"
	businessservice "hauslet/internal/modules/business/service"
	profilegraphql "hauslet/internal/modules/profile/port/graphql"
	profileservice "hauslet/internal/modules/profile/service"
	propertygraphql "hauslet/internal/modules/property/port/graphql"
	propertyservice "hauslet/internal/modules/property/service"
	wishlistgraphql "hauslet/internal/modules/wishlist/port/graphql"
	wishlistservice "hauslet/internal/modules/wishlist/service"

	"github.com/go-pkgz/lgr"
)

// Resolver wires domain-specific resolvers into gqlgen.
type Resolver struct {
	log              *lgr.Logger
	AuthResolver     *authgraphql.Resolver
	ProfileResolver  *profilegraphql.Resolver
	PropertyResolver *propertygraphql.Resolver
	BusinessResolver *businessgraphql.Resolver
	WishlistResolver *wishlistgraphql.Resolver
}

func NewResolver(
	authSvc authservice.AuthService,
	profileSvc profileservice.ProfileService,
	propertySvc propertyservice.Service,
	businessSvc businessservice.BusinessService,
	wishlistSvc wishlistservice.WishlistService,
	appCfg *cfg.GlobalConfig,
	log *lgr.Logger,
) *Resolver {
	return &Resolver{
		log:              log,
		AuthResolver:     authgraphql.NewResolver(authSvc),
		ProfileResolver:  profilegraphql.NewResolver(profileSvc, &appCfg.Storage, log),
		PropertyResolver: propertygraphql.NewResolver(propertySvc, &appCfg.Storage, log),
		BusinessResolver: businessgraphql.NewResolver(businessSvc, log),
		WishlistResolver: wishlistgraphql.NewResolver(wishlistSvc, log),
	}
}
