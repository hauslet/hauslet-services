package server

import (
	"hauslet/config"
	"hauslet/internal/transport/graph"

	"github.com/go-chi/chi/v5"
)

// setupRoutes configures all HTTP routes for the application
func setupRoutes(r chi.Router, container *Container, cfg *config.GlobalConfig) {
	// Setup auth routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		container.AuthHTTP.SetupRoutesWithRateLimiting(r, *container.Redis)
	} else {
		container.AuthHTTP.SetupRoutes(r)
	}

	// Setup property routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.PropertyHTTP.SetupRoutesWithRateLimiting(r, container.AuthSvc, *container.Redis, container.BusinessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.PropertyHTTP.SetupRoutes(r, container.AuthSvc, container.BusinessMW)
		})
	}

	// Setup calendar routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		r.Group(func(r chi.Router) {
			container.CalendarHTTP.SetupRoutesWithRateLimiting(r, container.AuthSvc, *container.Redis, container.BusinessMW)
		})
	} else {
		r.Group(func(r chi.Router) {
			container.CalendarHTTP.SetupRoutes(r, container.AuthSvc, container.BusinessMW)
		})
	}

	// Setup payment webhook routes with optional rate limiting in production
	if cfg.App.Env == "production" {
		container.PaymentWebhookHTTP.SetupRoutesWithRateLimiting(r, *container.Redis)
	} else {
		container.PaymentWebhookHTTP.SetupRoutes(r)
	}

	// Setup GraphQL routes
	graph.SetupGraphQL(r,
		container.AuthSvc,
		container.ProfileSvc,
		container.PropertySvc,
		container.BusinessSvc,
		container.PaymentsSvc,
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
		container.BusinessMW.Auth.WithTenantSlug,
		container.FXClient,
		container.Redis,
		cfg,
		container.Logger,
	)
}
