package graph

import (
	"hauslet/config"
	"hauslet/internal/auth/service"
	"hauslet/internal/graph/loaders"
	profileservice "hauslet/internal/profile/service"

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
	cfg *config.AppConfig,
	log *lgr.Logger) {

	srv := handler.New(
		NewExecutableSchema(Config{
			Resolvers: &Resolver{
				AuthService:    authService,
				ProfileService: profileService,
			},
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

	if cfg.Env != "production" {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	authMiddleware := authService.OAuthService().Middleware()
	r.Group(func(r chi.Router) {

		// Optional: Middleware to extract User from JWT and put in Context
		r.Use(authMiddleware.Trace)
		// Capture viewer info for resolvers (optional auth).
		r.Use(WithViewerContext)
		// DataLoaders to batch profile fetches.
		r.Use(loaders.Middleware(profileService))

		// The Query Endpoint
		r.Handle("/query", srv)
	})

	if cfg.Env != "production" {
		r.Handle("/playground", playground.Handler("Hauslet GraphQL", "/query"))
		log.Logf("[INFO] GraphQL Playground available at /playground")
	}
}
