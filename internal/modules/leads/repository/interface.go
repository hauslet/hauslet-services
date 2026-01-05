package repository

import (
	"context"
	"hauslet/internal/modules/leads/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LeadRepository defines the interface for lead data operations
type LeadRepository interface {
	// Lead CRUD
	CreateLead(ctx context.Context, lead *schema.Lead) error
	GetLeadByID(ctx context.Context, id uuid.UUID) (*schema.Lead, error)
	UpdateLead(ctx context.Context, lead *schema.Lead) error
	DeleteLead(ctx context.Context, id uuid.UUID) error

	// Queries with filters
	ListLeadsByListing(ctx context.Context, listingID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error)
	ListLeadsByBusiness(ctx context.Context, businessID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error)
	ListLeadsByOwner(ctx context.Context, ownerID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error)
	ListLeadsByAssignee(ctx context.Context, userID uuid.UUID, filter LeadFilter, page Pagination) ([]*schema.Lead, int64, error)

	// Rate Limiting Queries
	CountLeadsByEmail(ctx context.Context, email string, since time.Time) (int64, error)
	CountLeadsByIP(ctx context.Context, ipAddress string, since time.Time) (int64, error)
	CountLeadsByListing(ctx context.Context, listingID uuid.UUID, email string, since time.Time) (int64, error)

	// Duplicate Detection
	GetLeadByEmailAndListing(ctx context.Context, email string, listingID uuid.UUID, since time.Time) (*schema.Lead, error)

	// Transaction support
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// LeadEventRepository defines the interface for lead event data operations
type LeadEventRepository interface {
	CreateLeadEvent(ctx context.Context, event *schema.LeadEvent) error
	ListLeadEvents(ctx context.Context, leadID uuid.UUID, limit int) ([]*schema.LeadEvent, error)
}

// LeadAssignmentRepository defines the interface for lead assignment data operations
type LeadAssignmentRepository interface {
	CreateLeadAssignment(ctx context.Context, assignment *schema.LeadAssignment) error
	ListLeadAssignments(ctx context.Context, leadID uuid.UUID) ([]*schema.LeadAssignment, error)
	GetLatestAssignment(ctx context.Context, leadID uuid.UUID) (*schema.LeadAssignment, error)
}

// LeadFilter represents filter criteria for lead queries
type LeadFilter struct {
	Status     []string
	Source     []string
	IsSpam     *bool
	Assigned   *bool
	DateFrom   *time.Time
	DateTo     *time.Time
	SearchTerm *string // Search in name, email, message
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
