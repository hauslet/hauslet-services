package service

import (
	"context"
	"hauslet/internal/modules/leads/domain"
	"time"

	"github.com/google/uuid"
)

// LeadService defines the interface for lead business operations
type LeadService interface {
	// CreateLead creates a new lead with validation, spam detection, and rate limiting
	// This is a PUBLIC endpoint - no authentication required but rate limited
	CreateLead(ctx context.Context, input CreateLeadInput) (*domain.Lead, error)

	// GetLead retrieves a lead by ID (requires authorization)
	GetLead(ctx context.Context, leadID uuid.UUID, requesterID uuid.UUID) (*domain.Lead, error)

	// UpdateLeadStatus updates the status of a lead (requires authorization)
	UpdateLeadStatus(ctx context.Context, leadID uuid.UUID, status domain.LeadStatus, requesterID uuid.UUID, notes *string) (*domain.Lead, error)

	// AssignLead assigns a lead to a user (requires authorization)
	AssignLead(ctx context.Context, leadID uuid.UUID, assigneeID uuid.UUID, requesterID uuid.UUID, reason domain.AssignmentReason) (*domain.Lead, error)

	// MarkAsSpam marks a lead as spam (requires authorization)
	MarkAsSpam(ctx context.Context, leadID uuid.UUID, requesterID uuid.UUID) error

	// DeleteLead soft deletes a lead (requires authorization)
	DeleteLead(ctx context.Context, leadID uuid.UUID, requesterID uuid.UUID) error

	// ListLeadsByListing retrieves leads for a specific listing (requires authorization)
	ListLeadsByListing(ctx context.Context, listingID uuid.UUID, requesterID uuid.UUID, filter LeadFilter, page Pagination) ([]*domain.Lead, int64, error)

	// ListLeadsByBusiness retrieves leads for a business (requires business membership)
	ListLeadsByBusiness(ctx context.Context, businessID uuid.UUID, requesterID uuid.UUID, filter LeadFilter, page Pagination) ([]*domain.Lead, int64, error)

	// ListMyLeads retrieves leads assigned to the requester
	ListMyLeads(ctx context.Context, requesterID uuid.UUID, filter LeadFilter, page Pagination) ([]*domain.Lead, int64, error)

	// GetLeadHistory retrieves the event history for a lead (requires authorization)
	GetLeadHistory(ctx context.Context, leadID uuid.UUID, requesterID uuid.UUID) ([]*domain.LeadEvent, error)

	// RegisterMessagingHooks allows the container to wire an optional messaging adapter.
	RegisterMessagingHooks(h MessagingHooks)
}

// CreateLeadInput represents the input for creating a new lead
type CreateLeadInput struct {
	ListingID uuid.UUID

	// Hybrid Authentication (Optional)
	UserID *uuid.UUID // If provided, will auto-fill from profile and mark as verified

	// Contact Information (required if UserID is nil, optional if UserID provided)
	Name        string
	Email       string
	PhoneNumber *string
	Message     string
	Source      domain.LeadSource

	// Tracking metadata (automatically captured from request)
	UserAgent   *string
	IPAddress   *string
	ReferrerURL *string
	UTMParams   map[string]string
}

// LeadFilter represents filter criteria for lead queries
type LeadFilter struct {
	Status     []domain.LeadStatus
	Source     []domain.LeadSource
	IsSpam     *bool
	Assigned   *bool
	DateFrom   *time.Time
	DateTo     *time.Time
	SearchTerm *string
}

// Pagination represents pagination parameters
type Pagination struct {
	Limit  int
	Offset int
}

// DefaultPagination returns default pagination values
func DefaultPagination() Pagination {
	return Pagination{
		Limit:  20,
		Offset: 0,
	}
}

// ValidatePagination ensures pagination values are within acceptable limits
func ValidatePagination(page Pagination) Pagination {
	if page.Limit <= 0 || page.Limit > 100 {
		page.Limit = 20
	}
	if page.Offset < 0 {
		page.Offset = 0
	}
	return page
}
