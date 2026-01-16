package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"hauslet/cmd/api/server/middleware"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/finance/service"
	"hauslet/internal/platform/ratelimit"

	"github.com/go-chi/chi/v5"
	authmw "github.com/go-pkgz/auth/v2/middleware"
)

// HTTPHandler handles HTTP requests for finance module
type HTTPHandler struct {
	financeService service.FinanceService
	payoutService  service.PayoutService
	ctx            context.Context
	log            *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler for finance
func NewHTTPHandler(
	ctx context.Context,
	financeService service.FinanceService,
	payoutService service.PayoutService,
	log *slog.Logger,
) *HTTPHandler {
	return &HTTPHandler{
		financeService: financeService,
		payoutService:  payoutService,
		ctx:            ctx,
		log:            log,
	}
}

// SetupRoutes configures all finance admin-related routes
func (h *HTTPHandler) SetupRoutes(r chi.Router, authMiddleware *authmw.Authenticator) {
	// Admin-only routes (requires admin or root role)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth, authmiddleware.RBAC("admin"))

		// Dispute routes
		r.Get("/admin/finance/disputes", h.ListDisputes)
		r.Post("/admin/finance/disputes/{id}/investigate", h.InvestigateDispute)
		r.Post("/admin/finance/disputes/{id}/resolve", h.ResolveDispute)

		// Reconciliation routes
		r.Get("/admin/finance/reconciliation/reports", h.ListReconciliationReports)
		r.Get("/admin/finance/reconciliation/reports/latest", h.GetLatestReconciliation)
		r.Get("/admin/finance/reconciliation/reports/{id}", h.GetReconciliationReport)
		r.Get("/admin/finance/reconciliation/reports/{id}/discrepancies", h.GetReconciliationDiscrepancies)

		// Wallet routes
		r.Get("/admin/finance/wallets/{id}", h.GetWallet)
		r.Get("/admin/finance/wallets/user/{userId}", h.GetUserWallets)
		r.Get("/admin/finance/wallets/{id}/ledger", h.GetWalletLedger)

		// Disbursement routes
		r.Get("/admin/finance/disbursements/{id}", h.GetDisbursement)
	})
}

// SetupRoutesWithRateLimiting configures finance admin routes with rate limiting
func (h *HTTPHandler) SetupRoutesWithRateLimiting(r chi.Router, authMiddleware *authmw.Authenticator, limiter ratelimit.Limiter) {
	// Admin-only routes with rate limiting (requires admin or root role)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Auth, authmiddleware.RBAC("admin"))

		// Dispute routes - moderate rate limiting
		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/finance/disputes", h.ListDisputes)

		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Post("/admin/finance/disputes/{id}/investigate", h.InvestigateDispute)

		r.With(middleware.RateLimitIP(limiter, 20, time.Minute)).
			Post("/admin/finance/disputes/{id}/resolve", h.ResolveDispute)

		// Reconciliation routes - relaxed rate limiting (read-only operations)
		r.With(middleware.RateLimitIP(limiter, 60, time.Minute)).
			Get("/admin/finance/reconciliation/reports", h.ListReconciliationReports)

		r.With(middleware.RateLimitIP(limiter, 60, time.Minute)).
			Get("/admin/finance/reconciliation/reports/latest", h.GetLatestReconciliation)

		r.With(middleware.RateLimitIP(limiter, 60, time.Minute)).
			Get("/admin/finance/reconciliation/reports/{id}", h.GetReconciliationReport)

		r.With(middleware.RateLimitIP(limiter, 60, time.Minute)).
			Get("/admin/finance/reconciliation/reports/{id}/discrepancies", h.GetReconciliationDiscrepancies)

		// Wallet routes - moderate rate limiting (sensitive financial data)
		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/finance/wallets/{id}", h.GetWallet)

		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/finance/wallets/user/{userId}", h.GetUserWallets)

		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/finance/wallets/{id}/ledger", h.GetWalletLedger)

		// Disbursement routes - moderate rate limiting
		r.With(middleware.RateLimitIP(limiter, 100, time.Minute)).
			Get("/admin/finance/disbursements/{id}", h.GetDisbursement)
	})
}

// sendError sends an error response
func (h *HTTPHandler) sendError(w http.ResponseWriter, message string, statusCode int, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResponse := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	if field != "" {
		errResponse.Field = field
	}

	json.NewEncoder(w).Encode(errResponse)
}

// sendSuccess sends a success response
func (h *HTTPHandler) sendSuccess(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}
