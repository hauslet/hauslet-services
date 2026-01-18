package server

import (
	"hauslet/config"
	"hauslet/internal/transport/graph"

	_ "hauslet/docs"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// setupRoutes configures all HTTP routes for the application
func setupRoutes(r chi.Router, container *Container, cfg *config.GlobalConfig) {
	// Setup auth routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		container.AuthHTTP.SetupRoutesWithRateLimiting(r, container.RateLimiter)
	} else {
		container.AuthHTTP.SetupRoutes(r)
	}

	// Setup property routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.PropertyHTTP.SetupRoutesWithRateLimiting(r, container.AuthSvc, container.RateLimiter, container.BusinessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.PropertyHTTP.SetupRoutes(r, container.AuthSvc, container.BusinessMW)
		})
	}

	// Setup calendar routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.CalendarHTTP.SetupRoutesWithRateLimiting(r, container.AuthSvc, container.RateLimiter, container.BusinessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.CalendarHTTP.SetupRoutes(r, container.AuthSvc, container.BusinessMW)
		})
	}

	// Setup lead routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.LeadHTTP.SetupRoutesWithRateLimiting(r, container.AuthSvc, container.RateLimiter, container.BusinessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.LeadHTTP.SetupRoutes(r, container.AuthSvc, container.BusinessMW)
		})
	}

	// Setup messaging REST endpoints (attachments + helpers)
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.MessagingHTTP.SetupRoutesWithRateLimiting(r, container.AuthSvc, container.RateLimiter)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.MessagingHTTP.SetupRoutes(r, container.AuthSvc)
		})
	}

	// Setup finance admin routes with optional rate limiting in production
	authMiddleware := container.AuthSvc.OAuthService().Middleware()
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.FinanceHTTP.SetupRoutesWithRateLimiting(r, &authMiddleware, container.RateLimiter)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.FinanceHTTP.SetupRoutes(r, &authMiddleware)
		})
	}

	// Setup review admin routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.ReviewHTTP.SetupRoutesWithRateLimiting(r, &authMiddleware, container.RateLimiter)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.ReviewHTTP.SetupRoutes(r, &authMiddleware)
		})
	}

	// Setup payment webhook routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		container.PaymentWebhookHTTP.SetupRoutesWithRateLimiting(r, container.RateLimiter)
	} else {
		container.PaymentWebhookHTTP.SetupRoutes(r)
	}

	// Setup verification webhook routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		container.VerificationWebhookHTTP.SetupRoutesWithRateLimiting(r, container.RateLimiter)
	} else {
		container.VerificationWebhookHTTP.SetupRoutes(r)
	}

	// Setup GraphQL routes
	graph.SetupGraphQL(r,
		container.AuthSvc,
		container.ProfileSvc,
		container.PropertySvc,
		container.BusinessSvc,
		container.PaymentsSvc,
		container.PricingSvc,
		container.FinanceSvc,
		container.PayoutSvc,
		container.BookingSvc,
		container.CalendarSvc,
		container.WishlistSvc,
		container.ReviewSvc,
		container.PromotionSvc,
		container.SubscriptionSvc,
		container.UsageSvc,
		container.LeadSvc,
		container.InteractionTracker,
		container.InteractionReader,
		container.DiscoverySvc,
		container.VerificationSvc,
		container.MessagingSvc,
		container.BusinessMW.Auth.WithTenantSlug,
		container.FXClient,
		container.RateLimiter,
		container.EventSubscriber,
		cfg,
		container.Logger,
	)

	// Setup Swagger UI (Dev/Staging only)
	if cfg.App.Env != "production" {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}
}
