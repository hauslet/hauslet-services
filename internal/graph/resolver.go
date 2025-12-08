package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	cfg "hauslet/config"
	authgraphql "hauslet/internal/auth/port/graphql"
	authservice "hauslet/internal/auth/service"
	profilegraphql "hauslet/internal/profile/port/graphql"
	profileservice "hauslet/internal/profile/service"
	propertygraphql "hauslet/internal/property/port/graphql"
	propertyservice "hauslet/internal/property/service"

	"github.com/go-pkgz/lgr"
)

// Resolver wires domain-specific resolvers into gqlgen.
type Resolver struct {
	log              *lgr.Logger
	AuthResolver     *authgraphql.Resolver
	ProfileResolver  *profilegraphql.Resolver
	PropertyResolver *propertygraphql.Resolver
}

func NewResolver(
	authSvc authservice.AuthService,
	profileSvc profileservice.ProfileService,
	propertySvc propertyservice.Service,
	appCfg *cfg.GlobalConfig,
	log *lgr.Logger,
) *Resolver {
	return &Resolver{
		log:              log,
		AuthResolver:     authgraphql.NewResolver(authSvc),
		ProfileResolver:  profilegraphql.NewResolver(profileSvc, log),
		PropertyResolver: propertygraphql.NewResolver(propertySvc, appCfg, log),
	}
}
