package graphql

import (
	"context"
	"fmt"
	authmiddleware "hauslet/internal/modules/auth/middleware"
	"hauslet/internal/modules/leads/domain"
	"hauslet/internal/modules/leads/service"
	"hauslet/internal/transport/graph/viewer"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Resolver provides GraphQL resolvers for the leads module
type Resolver struct {
	leadService service.LeadService
	log         *slog.Logger
}

// NewResolver creates a new instance of Resolver
func NewResolver(leadService service.LeadService, log *slog.Logger) *Resolver {
	return &Resolver{
		leadService: leadService,
		log:         log,
	}
}

// Lead retrieves a single lead by ID (requires authentication)
func (r *Resolver) Lead(ctx context.Context, id string) (*domain.Lead, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		r.log.Warn("unauthenticated attempt to get lead")
		return nil, domain.ErrUnauthorized
	}

	leadID, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.ErrInvalidLeadID
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	lead, err := r.leadService.GetLead(ctx, leadID, requesterID)
	if err != nil {
		r.log.Error("failed to get lead", "lead_id", id, "error", err)
		return nil, err
	}

	return lead, nil
}

// CreateLead creates a new lead (PUBLIC endpoint - no auth required)
func (r *Resolver) CreateLead(ctx context.Context, input CreateLeadInput) (*domain.Lead, error) {
	// Extract metadata from request context
	ipAddress := extractIPFromContext(ctx)
	userAgent := extractUserAgentFromContext(ctx)
	referrerURL := extractReferrerFromContext(ctx)

	listingID, err := uuid.Parse(input.ListingID)
	if err != nil {
		return nil, domain.ErrInvalidListingID
	}

	serviceInput := service.CreateLeadInput{
		ListingID:   listingID,
		Name:        input.Name,
		Email:       input.Email,
		PhoneNumber: input.PhoneNumber,
		Message:     input.Message,
		Source:      mapLeadSource(string(input.Source)),
		UserAgent:   &userAgent,
		IPAddress:   &ipAddress,
		ReferrerURL: &referrerURL,
		UTMParams:   normalizeUTMParams(input.UtmParams),
	}

	// HYBRID: Check if user is authenticated (optional)
	v := viewer.FromContext(ctx)
	if v != nil && v.UserID != "" {
		userID, err := uuid.Parse(v.UserID)
		if err == nil {
			serviceInput.UserID = &userID // Pass UserID for auto-fill and verification
			r.log.Info("authenticated user creating lead", "user_id", userID, "listing_id", listingID)
		}
	} else {
		r.log.Info("anonymous user creating lead", "listing_id", listingID, "email", input.Email)
	}

	lead, err := r.leadService.CreateLead(ctx, serviceInput)
	if err != nil {
		r.log.Error("failed to create lead", "error", err, "email", input.Email)
		return nil, err
	}

	r.log.Info("lead created via graphql",
		"lead_id", lead.ID,
		"listing_id", listingID,
		"email", input.Email,
		"is_verified", lead.IsVerified,
		"is_authenticated", lead.IsAuthenticatedUser(),
	)

	return lead, nil
}

// UpdateLeadStatus updates the status of a lead
func (r *Resolver) UpdateLeadStatus(ctx context.Context, leadID string, status domain.LeadStatus, notes *string) (*domain.Lead, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, domain.ErrUnauthorized
	}

	leadUUID, err := uuid.Parse(leadID)
	if err != nil {
		return nil, domain.ErrInvalidLeadID
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	leadStatus := mapLeadStatus(string(status))

	lead, err := r.leadService.UpdateLeadStatus(ctx, leadUUID, leadStatus, requesterID, notes)
	if err != nil {
		r.log.Error("failed to update lead status", "lead_id", leadID, "error", err)
		return nil, err
	}

	return lead, nil
}

// AssignLead assigns a lead to a user
func (r *Resolver) AssignLead(ctx context.Context, leadID, assigneeID string, reason domain.AssignmentReason) (*domain.Lead, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, domain.ErrUnauthorized
	}

	leadUUID, err := uuid.Parse(leadID)
	if err != nil {
		return nil, domain.ErrInvalidLeadID
	}

	assigneeUUID, err := uuid.Parse(assigneeID)
	if err != nil {
		return nil, domain.ErrInvalidAssignee
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	assignmentReason := mapAssignmentReason(string(reason))

	lead, err := r.leadService.AssignLead(ctx, leadUUID, assigneeUUID, requesterID, assignmentReason)
	if err != nil {
		r.log.Error("failed to assign lead", "lead_id", leadID, "error", err)
		return nil, err
	}

	return lead, nil
}

// MarkLeadAsSpam marks a lead as spam
func (r *Resolver) MarkLeadAsSpam(ctx context.Context, leadID string) (bool, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return false, domain.ErrUnauthorized
	}

	leadUUID, err := uuid.Parse(leadID)
	if err != nil {
		return false, domain.ErrInvalidLeadID
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return false, domain.ErrUnauthorized
	}

	err = r.leadService.MarkAsSpam(ctx, leadUUID, requesterID)
	if err != nil {
		r.log.Error("failed to mark lead as spam", "lead_id", leadID, "error", err)
		return false, err
	}

	return true, nil
}

// DeleteLead soft deletes a lead
func (r *Resolver) DeleteLead(ctx context.Context, leadID string) (bool, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return false, domain.ErrUnauthorized
	}

	leadUUID, err := uuid.Parse(leadID)
	if err != nil {
		return false, domain.ErrInvalidLeadID
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return false, domain.ErrUnauthorized
	}

	err = r.leadService.DeleteLead(ctx, leadUUID, requesterID)
	if err != nil {
		r.log.Error("failed to delete lead", "lead_id", leadID, "error", err)
		return false, err
	}

	return true, nil
}

// LeadsByListing retrieves leads for a specific listing
func (r *Resolver) LeadsByListing(ctx context.Context, listingID string, filter *LeadFilterInput, page *PageInput) (*LeadConnection, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, domain.ErrUnauthorized
	}

	listingUUID, err := uuid.Parse(listingID)
	if err != nil {
		return nil, domain.ErrInvalidListingID
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	serviceFilter := convertFilter(filter)
	servicePage := convertPage(page)

	leads, total, err := r.leadService.ListLeadsByListing(ctx, listingUUID, requesterID, serviceFilter, servicePage)
	if err != nil {
		r.log.Error("failed to list leads by listing", "listing_id", listingID, "error", err)
		return nil, err
	}

	return buildLeadConnection(leads, total, servicePage), nil
}

// LeadsByBusiness retrieves leads for a business
func (r *Resolver) LeadsByBusiness(ctx context.Context, businessID string, filter *LeadFilterInput, page *PageInput) (*LeadConnection, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, domain.ErrUnauthorized
	}

	businessUUID, err := uuid.Parse(businessID)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	serviceFilter := convertFilter(filter)
	servicePage := convertPage(page)

	leads, total, err := r.leadService.ListLeadsByBusiness(ctx, businessUUID, requesterID, serviceFilter, servicePage)
	if err != nil {
		r.log.Error("failed to list leads by business", "business_id", businessID, "error", err)
		return nil, err
	}

	return buildLeadConnection(leads, total, servicePage), nil
}

// MyLeads retrieves leads assigned to the current user
func (r *Resolver) MyLeads(ctx context.Context, filter *LeadFilterInput, page *PageInput) (*LeadConnection, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, domain.ErrUnauthorized
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	serviceFilter := convertFilter(filter)
	servicePage := convertPage(page)

	leads, total, err := r.leadService.ListMyLeads(ctx, requesterID, serviceFilter, servicePage)
	if err != nil {
		r.log.Error("failed to list my leads", "error", err)
		return nil, err
	}

	return buildLeadConnection(leads, total, servicePage), nil
}

// LeadHistory retrieves the event history for a lead
func (r *Resolver) LeadHistory(ctx context.Context, leadID string) ([]*domain.LeadEvent, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return nil, domain.ErrUnauthorized
	}

	leadUUID, err := uuid.Parse(leadID)
	if err != nil {
		return nil, domain.ErrInvalidLeadID
	}

	requesterID, err := uuid.Parse(v.UserID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	events, err := r.leadService.GetLeadHistory(ctx, leadUUID, requesterID)
	if err != nil {
		r.log.Error("failed to get lead history", "lead_id", leadID, "error", err)
		return nil, err
	}

	return events, nil
}

// Helper functions

func extractIPFromContext(ctx context.Context) string {
	return authmiddleware.GetIPFromContext(ctx)
}

func extractUserAgentFromContext(ctx context.Context) string {
	return authmiddleware.GetUserAgentFromContext(ctx)
}

func extractReferrerFromContext(ctx context.Context) string {
	return authmiddleware.GetReferrerFromContext(ctx)
}

func mapLeadSource(s string) domain.LeadSource {
	switch strings.ToUpper(s) {
	case "WEBSITE":
		return domain.SourceWebsite
	case "MOBILE_APP":
		return domain.SourceMobileApp
	case "API":
		return domain.SourceAPI
	case "WHATSAPP":
		return domain.SourceWhatsApp
	case "PHONE_CALL":
		return domain.SourcePhoneCall
	case "EMAIL_CAMPAIGN":
		return domain.SourceEmailCampaign
	default:
		return domain.SourceWebsite
	}
}

func mapLeadStatus(s string) domain.LeadStatus {
	switch strings.ToUpper(s) {
	case "NEW":
		return domain.StatusNew
	case "ASSIGNED":
		return domain.StatusAssigned
	case "CONTACTED":
		return domain.StatusContacted
	case "QUALIFIED":
		return domain.StatusQualified
	case "CONVERTED":
		return domain.StatusConverted
	case "LOST":
		return domain.StatusLost
	case "SPAM":
		return domain.StatusSpam
	case "ARCHIVED":
		return domain.StatusArchived
	default:
		return domain.StatusNew
	}
}

func mapAssignmentReason(r string) domain.AssignmentReason {
	switch strings.ToUpper(r) {
	case "AUTO":
		return domain.ReasonAuto
	case "MANUAL":
		return domain.ReasonManual
	case "REASSIGN":
		return domain.ReasonReassign
	case "ESCALATE":
		return domain.ReasonEscalate
	default:
		return domain.ReasonManual
	}
}

func convertFilter(filter *LeadFilterInput) service.LeadFilter {
	if filter == nil {
		return service.LeadFilter{}
	}

	return service.LeadFilter{
		Status:     normalizeLeadStatuses(filter.Status),
		Source:     normalizeLeadSources(filter.Source),
		IsSpam:     filter.IsSpam,
		Assigned:   filter.Assigned,
		DateFrom:   filter.DateFrom,
		DateTo:     filter.DateTo,
		SearchTerm: filter.SearchTerm,
	}
}

func convertPage(page *PageInput) service.Pagination {
	if page == nil {
		return service.DefaultPagination()
	}

	p := service.Pagination{
		Limit:  20,
		Offset: 0,
	}

	if page.Limit != nil {
		p.Limit = *page.Limit
	}
	if page.Offset != nil {
		p.Offset = *page.Offset
	}

	return service.ValidatePagination(p)
}

func buildLeadConnection(leads []*domain.Lead, total int64, page service.Pagination) *LeadConnection {
	hasNextPage := int64(page.Offset+page.Limit) < total

	return &LeadConnection{
		Items:       leads,
		TotalCount:  int(total),
		HasNextPage: hasNextPage,
	}
}

// Input/Output types (these would normally be generated by gqlgen)

type CreateLeadInput struct {
	ListingID   string
	Name        string
	Email       string
	PhoneNumber *string
	Message     string
	Source      domain.LeadSource
	UtmParams   map[string]any
}

type LeadFilterInput struct {
	Status     []domain.LeadStatus
	Source     []domain.LeadSource
	IsSpam     *bool
	Assigned   *bool
	DateFrom   *time.Time
	DateTo     *time.Time
	SearchTerm *string
}

type PageInput struct {
	Limit  *int
	Offset *int
}

type LeadConnection struct {
	Items       []*domain.Lead
	TotalCount  int
	HasNextPage bool
}

func normalizeUTMParams(params map[string]any) map[string]string {
	if params == nil {
		return nil
	}
	normalized := make(map[string]string, len(params))
	for key, value := range params {
		switch typed := value.(type) {
		case string:
			normalized[key] = typed
		default:
			normalized[key] = fmt.Sprint(typed)
		}
	}
	return normalized
}

func normalizeLeadStatuses(statuses []domain.LeadStatus) []domain.LeadStatus {
	if len(statuses) == 0 {
		return nil
	}
	normalized := make([]domain.LeadStatus, len(statuses))
	for i, status := range statuses {
		normalized[i] = mapLeadStatus(string(status))
	}
	return normalized
}

func normalizeLeadSources(sources []domain.LeadSource) []domain.LeadSource {
	if len(sources) == 0 {
		return nil
	}
	normalized := make([]domain.LeadSource, len(sources))
	for i, source := range sources {
		normalized[i] = mapLeadSource(string(source))
	}
	return normalized
}
