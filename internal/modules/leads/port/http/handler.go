package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/leads/domain"
	"hauslet/internal/modules/leads/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
)

// createLeadRequest represents the payload for creating a lead (PUBLIC endpoint)
type createLeadRequest struct {
	ListingID   string            `json:"listing_id"`
	Name        string            `json:"name"`
	Email       string            `json:"email"`
	PhoneNumber *string           `json:"phone_number,omitempty"`
	Message     string            `json:"message"`
	Source      string            `json:"source"`               // website, mobile_app, api, etc.
	UTMParams   map[string]string `json:"utm_params,omitempty"` // UTM tracking parameters
}

// createLeadResponse represents the response after creating a lead
type createLeadResponse struct {
	LeadID      uuid.UUID `json:"lead_id"`
	ListingID   uuid.UUID `json:"listing_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	PhoneNumber *string   `json:"phone_number,omitempty"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	IsVerified  bool      `json:"is_verified"`
	CreatedAt   time.Time `json:"created_at"`
}

// getLeadResponse represents the response for retrieving a lead
type getLeadResponse struct {
	LeadID       uuid.UUID          `json:"lead_id"`
	ListingID    uuid.UUID          `json:"listing_id"`
	BusinessID   *uuid.UUID         `json:"business_id,omitempty"`
	Name         string             `json:"name"`
	Email        string             `json:"email"`
	PhoneNumber  *string            `json:"phone_number,omitempty"`
	Message      string             `json:"message"`
	Source       string             `json:"source"`
	Status       string             `json:"status"`
	IsSpam       bool               `json:"is_spam"`
	SpamScore    float64            `json:"spam_score"`
	AssignedTo   *uuid.UUID         `json:"assigned_to,omitempty"`
	AssignedAt   *time.Time         `json:"assigned_at,omitempty"`
	AutoAssigned bool               `json:"auto_assigned"`
	UTMParams    map[string]string  `json:"utm_params,omitempty"`
	IsVerified   bool               `json:"is_verified"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	ResponseTime *int64             `json:"response_time_seconds,omitempty"` // seconds to first response
}

// createLead handles POST /api/v1/leads - PUBLIC endpoint for ad integrations
func (h *HTTPHandler) createLead(w http.ResponseWriter, r *http.Request) {
	var req createLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// Validate required fields
	if req.ListingID == "" {
		writeError(w, http.StatusBadRequest, "listing_id is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	listingID, err := uuid.Parse(req.ListingID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing_id format")
		return
	}

	// Extract metadata from request context (IP, User-Agent, Referrer)
	ipAddress := authmiddleware.GetIPFromContext(r.Context())
	userAgent := authmiddleware.GetUserAgentFromContext(r.Context())
	referrerURL := authmiddleware.GetReferrerFromContext(r.Context())

	// Map source string to domain enum
	leadSource := mapLeadSource(req.Source)

	serviceInput := service.CreateLeadInput{
		ListingID:   listingID,
		Name:        strings.TrimSpace(req.Name),
		Email:       strings.TrimSpace(strings.ToLower(req.Email)),
		PhoneNumber: req.PhoneNumber,
		Message:     strings.TrimSpace(req.Message),
		Source:      leadSource,
		UserAgent:   &userAgent,
		IPAddress:   &ipAddress,
		ReferrerURL: &referrerURL,
		UTMParams:   req.UTMParams,
	}

	// HYBRID: Check if user is authenticated (optional for public endpoint)
	userID, err := extractUserID(r)
	if err == nil {
		// User is authenticated - will auto-fill from profile and mark as verified
		serviceInput.UserID = &userID
		h.log.Info("authenticated user creating lead", "user_id", userID, "listing_id", listingID)
	} else {
		// Anonymous user - use provided contact info
		h.log.Info("anonymous user creating lead", "listing_id", listingID, "email", req.Email, "ip", ipAddress)
	}

	lead, err := h.leadService.CreateLead(h.ctx, serviceInput)
	if err != nil {
		h.log.Error("failed to create lead", "error", err, "email", req.Email, "ip", ipAddress)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.log.Info("lead created via rest api",
		"lead_id", lead.ID,
		"listing_id", listingID,
		"email", lead.Email,
		"is_verified", lead.IsVerified,
		"is_authenticated", lead.IsAuthenticatedUser(),
		"source", lead.Source,
	)

	resp := createLeadResponse{
		LeadID:      lead.ID,
		ListingID:   lead.ListingID,
		Name:        lead.Name,
		Email:       lead.Email,
		PhoneNumber: lead.PhoneNumber,
		Source:      lead.Source.String(),
		Status:      lead.Status.String(),
		IsVerified:  lead.IsVerified,
		CreatedAt:   lead.CreatedAt,
	}

	writeJSON(w, http.StatusCreated, resp)
}

// getLead handles GET /api/v1/leads/:id - Authenticated endpoint
func (h *HTTPHandler) getLead(w http.ResponseWriter, r *http.Request) {
	leadID, err := uuid.Parse(chi.URLParam(r, "leadId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lead id")
		return
	}

	requesterID, err := extractUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	lead, err := h.leadService.GetLead(h.ctx, leadID, requesterID)
	if err != nil {
		h.log.Error("failed to get lead", "lead_id", leadID, "error", err)

		// Check if it's an authorization error
		if err == domain.ErrUnauthorized || err == domain.ErrForbidden {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}

		if err == domain.ErrLeadNotFound {
			writeError(w, http.StatusNotFound, "lead not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "failed to retrieve lead")
		return
	}

	resp := getLeadResponse{
		LeadID:       lead.ID,
		ListingID:    lead.ListingID,
		BusinessID:   lead.BusinessID,
		Name:         lead.Name,
		Email:        lead.Email,
		PhoneNumber:  lead.PhoneNumber,
		Message:      lead.Message,
		Source:       lead.Source.String(),
		Status:       lead.Status.String(),
		IsSpam:       lead.IsSpam,
		SpamScore:    lead.SpamScore,
		AssignedTo:   lead.AssignedTo,
		AssignedAt:   lead.AssignedAt,
		AutoAssigned: lead.AutoAssigned,
		UTMParams:    lead.UTMParams,
		IsVerified:   lead.IsVerified,
		CreatedAt:    lead.CreatedAt,
		UpdatedAt:    lead.UpdatedAt,
		ResponseTime: lead.ResponseTime,
	}

	writeJSON(w, http.StatusOK, resp)
}

// Helper Functions

// extractUserID extracts and parses the user ID from the authenticated request
func extractUserID(r *http.Request) (uuid.UUID, error) {
	userInfo, err := token.GetUserInfo(r)
	if err != nil {
		return uuid.Nil, err
	}

	userID := userInfo.StrAttr("uid")
	if userID == "" {
		userID = userInfo.ID
		// Handle prefixed IDs (e.g., "oauth2_<uuid>")
		if parts := strings.SplitN(userID, "_", 2); len(parts) == 2 {
			userID = parts[1]
		}
	}

	return uuid.Parse(userID)
}

// mapLeadSource converts string to domain LeadSource enum with validation
func mapLeadSource(s string) domain.LeadSource {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "website":
		return domain.SourceWebsite
	case "mobile_app", "mobile":
		return domain.SourceMobileApp
	case "api":
		return domain.SourceAPI
	case "whatsapp":
		return domain.SourceWhatsApp
	case "phone_call", "phone":
		return domain.SourcePhoneCall
	case "email_campaign", "email":
		return domain.SourceEmailCampaign
	default:
		// Default to API for external integrations (ads, webhooks)
		return domain.SourceAPI
	}
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error":   http.StatusText(status),
		"message": message,
	})
}
