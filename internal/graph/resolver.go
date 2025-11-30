package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	authservice "hauslet/internal/auth/service"
	profileservice "hauslet/internal/profile/service"
)

// Resolver wires services into GraphQL resolvers.
type Resolver struct {
	AuthService    authservice.AuthService
	ProfileService profileservice.ProfileService
}
