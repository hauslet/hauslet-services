package http

import (
	"time"

	"hauslet/cmd/api/server/middleware"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
	authmw "github.com/go-pkgz/auth/v2/middleware"
)

// SetupRoutes configures payment admin routes without rate limiting
func (h *AdminHandler) SetupRoutes(r chi.Router, authMiddleware *authmw.Authenticator) {
	// Admin-only routes (requires admin or root role)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth, authmiddleware.RBAC("admin"))

		// Payment routes
		r.Post("/admin/payments", h.CreatePayment)
		r.Post("/admin/payments/verify/{reference}", h.VerifyPayment)
		r.Post("/admin/payments/refund", h.RefundPayment)
		r.Get("/admin/payments/list", h.ListAllPayments)

		// Payout details routes
		r.Get("/admin/payments/payout-details/user/{userId}", h.GetUserPayoutDetails)

		// Transaction routes
		r.Get("/admin/payments/transactions/booking/{bookingID}", h.GetBookingTransactions)
		r.Get("/admin/payments/transactions/list", h.ListAllTransactions)
	})
}

// SetupRoutesWithRateLimiting configures payment admin routes with rate limiting
func (h *AdminHandler) SetupRoutesWithRateLimiting(r chi.Router, authMiddleware *authmw.Authenticator, limiter ratelimit.Limiter) {
	// Admin-only routes with rate limiting (requires admin or root role)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth, authmiddleware.RBAC("admin"))

		// Payment mutation routes - strict rate limiting
		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Post("/admin/payments", h.CreatePayment)

		r.With(middleware.RateLimitIP(limiter, 30, time.Minute)).
			Post("/admin/payments/verify/{reference}", h.VerifyPayment)

		r.With(middleware.RateLimitIP(limiter, 10, time.Minute)).
			Post("/admin/payments/refund", h.RefundPayment)

		// Payment query routes - moderate rate limiting
		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/payments/list", h.ListAllPayments)

		// Payout details routes - moderate rate limiting
		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/payments/payout-details/user/{userId}", h.GetUserPayoutDetails)

		// Transaction routes - moderate rate limiting
		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/payments/transactions/booking/{bookingID}", h.GetBookingTransactions)

		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/payments/transactions/list", h.ListAllTransactions)
	})
}
