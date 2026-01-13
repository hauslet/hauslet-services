package service

import (
	"context"
	"time"

	"hauslet/internal/modules/review/domain"
	"hauslet/internal/modules/review/repository"

	"github.com/google/uuid"
)

// ReviewService defines all review-related operations:
// reviews, responses, moderation, and statistics.
type ReviewService interface {
	// --- Reviews ---

	CreateReview(ctx context.Context, input CreateReviewInput) (*domain.Review, error)
	GetReview(ctx context.Context, reviewID, requestorID uuid.UUID) (*domain.Review, error)
	UpdateReview(ctx context.Context, reviewID, actorID uuid.UUID, input UpdateReviewInput) (*domain.Review, error)
	DeleteReview(ctx context.Context, reviewID, actorID uuid.UUID) error

	ListReviewsForTarget(ctx context.Context, targetType domain.ReviewTargetType, targetID uuid.UUID, filter ReviewFilter,
	) ([]*domain.Review, int64, error)

	ListUserReviews(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Review, error)
	GetReviewForBooking(ctx context.Context, bookingID, reviewerID uuid.UUID) (*domain.Review, error)

	// --- Publishing ---

	PublishReview(ctx context.Context, reviewID, adminID uuid.UUID) error
	PublishExpiredStandoffs(ctx context.Context, olderThan time.Time) (int64, error)

	// --- Moderation ---

	ReportReview(ctx context.Context, reviewID, reporterID uuid.UUID, reason string) error
	HideReview(ctx context.Context, reviewID, adminID uuid.UUID, reason domain.ModerationReason) error
	UnhideReview(ctx context.Context, reviewID, adminID uuid.UUID) error

	// --- Internal hooks ---

	OnReviewCreated(ctx context.Context, reviewID uuid.UUID) error

	// --- Responses ---

	CreateResponse(ctx context.Context, reviewID, authorID uuid.UUID, body string) (*domain.ReviewResponse, error)
	GetResponse(ctx context.Context, reviewID uuid.UUID) (*domain.ReviewResponse, error)
	UpdateResponse(ctx context.Context, responseID, authorID uuid.UUID, body string) (*domain.ReviewResponse, error)
	DeleteResponse(ctx context.Context, responseID, actorID uuid.UUID) error

	// --- Statistics ---

	GetListingStats(ctx context.Context, listingID uuid.UUID) (*domain.ListingStats, error)
	GetHostStats(ctx context.Context, hostID uuid.UUID) (*domain.HostStats, error)

	RecalculateListingStats(ctx context.Context, listingID uuid.UUID) error
	RecalculateHostStats(ctx context.Context, hostID uuid.UUID) error
	RefreshAllStats(ctx context.Context) error
}

// --- Input DTOs ---

type CreateReviewInput struct {
	BookingID   uuid.UUID
	ReviewerID  uuid.UUID
	TargetType  domain.ReviewTargetType
	TargetID    uuid.UUID
	Rating      int
	Title       string
	Body        string
	SubRatings  *domain.SubRatings
	Language    string
	CountryCode string
}

type UpdateReviewInput struct {
	Rating     *int
	Title      *string
	Body       *string
	SubRatings *domain.SubRatings
}

// ReviewFilter defines pagination, sorting, and filtering options.
type ReviewFilter struct {
	Limit  int
	Offset int

	SortBy string // newest, highest_rated, lowest_rated, most_helpful

	Rating          *int
	Language        *string
	OnlyVisible     bool
	PreloadResponse bool // Eagerly fetch associated host responses when true
}

// --- External dependencies (cross-module) ---

// BookingQuerier exposes booking data needed for review authorization.
// This is a cross-module dependency (booking module).
type BookingQuerier interface {
	GetBookingParties(ctx context.Context, bookingID uuid.UUID) (guestID, hostID uuid.UUID, err error)
	GetBookingListing(ctx context.Context, bookingID uuid.UUID) (uuid.UUID, error)
	IsBookingCompleted(ctx context.Context, bookingID uuid.UUID) (bool, error)
}

// UserQuerier exposes minimal user data needed for notifications.
// This is a cross-module dependency (profile/user module).
// Implemented by an adapter like ReviewUserAdapter wrapping ProfileService.
type UserQuerier interface {
	GetUserContact(ctx context.Context, userID uuid.UUID) (name, email string, err error)
}

// BookingHooks defines callbacks for updating booking state when reviews are published.
// This is a cross-module dependency (booking module).
// Implemented by an adapter like ReviewHooksAdapter wrapping BookingRepository.
// Used to maintain guest_reviewed_at and host_reviewed_at timestamps in booking table.
type BookingHooks interface {
	// OnReviewPublished updates booking timestamps when review is published
	// reviewerType should be "guest" or "host" based on who is reviewing
	OnReviewPublished(ctx context.Context, bookingID, reviewerID uuid.UUID, reviewerType string) error
}

// RepositoryReviewFilter is re-exported for convenience.
type RepositoryReviewFilter = repository.ReviewFilter
