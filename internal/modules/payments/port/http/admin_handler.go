package http

import (
	"context"
	"encoding/json"
	"net/http"

	"hauslet/internal/modules/payments/service"
	"log/slog"
)

// AdminHandler handles HTTP requests for payments admin operations
type AdminHandler struct {
	paymentService service.PaymentService
	ctx            context.Context
	log            *slog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(paymentService service.PaymentService, ctx context.Context, log *slog.Logger) *AdminHandler {
	return &AdminHandler{
		paymentService: paymentService,
		ctx:            ctx,
		log:            log,
	}
}

// sendError sends an error response
func (h *AdminHandler) sendError(w http.ResponseWriter, message string, statusCode int, field string) {
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
func (h *AdminHandler) sendSuccess(w http.ResponseWriter, data any, statusCode int) {
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
