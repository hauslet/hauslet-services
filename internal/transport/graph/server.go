package graph

import (
	"hauslet/config"
	"hauslet/internal/modules/auth/service"
	businessmiddleware "hauslet/internal/modules/business/middleware"
	businessservice "hauslet/internal/modules/business/service"
	profileservice "hauslet/internal/modules/profile/service"
	propertyservice "hauslet/internal/modules/property/service"
	"hauslet/internal/transport/graph/loaders"
	"hauslet/internal/transport/graph/viewer"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/lgr"
	"github.com/vektah/gqlparser/v2/ast"
)

func SetupGraphQL(r chi.Router,
	authService service.AuthService,
	profileService profileservice.ProfileService,
	propertyService propertyservice.Service,
	businessService businessservice.BusinessService,
	cfg *config.GlobalConfig,
	log *lgr.Logger) {

	srv := handler.New(
		NewExecutableSchema(Config{
			Resolvers:  NewResolver(authService, profileService, propertyService, businessService, cfg, log),
			Complexity: NewComplexityRoot(defaultMaxListLimit),
		}),
	)

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
	businessMW := businessmiddleware.NewMiddleware(businessService, log)
	r.Group(func(r chi.Router) {

		// Optional: Middleware to extract User from JWT and put in Context
		r.Use(authMiddleware.Trace)
		// Capture viewer info for resolvers (optional auth).
		r.Use(viewer.WithContext)
		// Resolve X-Tenant-Slug to business context and enforce membership.
		r.Use(businessMW.Auth.WithTenantSlug)
		// DataLoaders to batch profile and property fetches.
		r.Use(loaders.Middleware(profileService, propertyService))

		// The Query Endpoint
		r.Handle("/query", srv)
	})

	if cfg.App.Env != "production" {
		r.Handle("/playground", playground.Handler("Hauslet GraphQL", "/query"))
		log.Logf("[INFO] GraphQL Playground available at /playground")
	}
}
