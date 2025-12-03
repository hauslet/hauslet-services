package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	authgraphql "hauslet/internal/auth/port/graphql"
	authservice "hauslet/internal/auth/service"
	profilegraphql "hauslet/internal/profile/port/graphql"
	profileservice "hauslet/internal/profile/service"
)

// Resolver wires domain-specific resolvers into gqlgen.
type Resolver struct {
	AuthResolver    *authgraphql.Resolver
	ProfileResolver *profilegraphql.Resolver
}

func NewResolver(authSvc authservice.AuthService, profileSvc profileservice.ProfileService) *Resolver {
	return &Resolver{
		AuthResolver:    authgraphql.NewResolver(authSvc),
		ProfileResolver: profilegraphql.NewResolver(profileSvc),
	}
}
