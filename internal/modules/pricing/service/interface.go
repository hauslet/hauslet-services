package service

import (
	"context"
	"hauslet/config"
	"hauslet/internal/modules/pricing/domain"
	"hauslet/internal/modules/pricing/repository"
	"hauslet/internal/platform/redis"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type PricingService interface {
	// --- Core Price Calculation ---
	// CalculatePrice calculates the total price for a booking
	CalculatePrice(ctx context.Context, listingID uuid.UUID, checkIn, checkOut time.Time, guestCount int) (*domain.PriceBreakdown, error)

	// GetBasePrice returns the base nightly rate for display
	GetBasePrice(ctx context.Context, listingID uuid.UUID) (float64, string, error)

	// PreviewPricing returns daily rates for a month (for calendar display)
	PreviewPricing(ctx context.Context, listingID uuid.UUID, month time.Time) ([]domain.DailyRate, error)

	// --- Refund Calculation ---
	// CalculateRefund calculates the refund amount based on cancellation policy
	CalculateRefund(ctx context.Context, input domain.RefundCalculationInput) (*domain.RefundBreakdown, error)

	// --- Pricing Rule Management ---
	CreateRule(ctx context.Context, rule *domain.PricingRule) (*domain.PricingRule, error)
	GetRule(ctx context.Context, ruleID uuid.UUID, requestorID uuid.UUID) (*domain.PricingRule, error)
	GetRulesForListing(ctx context.Context, listingID uuid.UUID, activeOnly bool) ([]*domain.PricingRule, error)
	UpdateRule(ctx context.Context, rule *domain.PricingRule, requestorID uuid.UUID) (*domain.PricingRule, error)
	DeleteRule(ctx context.Context, ruleID uuid.UUID, requestorID uuid.UUID) error

	// --- Multi-Property Discounts ---
	CreateMultiPropertyDiscount(ctx context.Context, discount *domain.MultiPropertyDiscount) (*domain.MultiPropertyDiscount, error)
	GetDiscountsForOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.MultiPropertyDiscount, error)
	CalculateMultiPropertyPrice(ctx context.Context, bookings []MultiPropertyBooking) (*domain.PriceBreakdown, error)
	UpdateDiscount(ctx context.Context, discount *domain.MultiPropertyDiscount) (*domain.MultiPropertyDiscount, error)
	DeleteDiscount(ctx context.Context, discountID uuid.UUID, ownerID uuid.UUID) error
}

// MultiPropertyBooking represents a booking for multi-property calculation
type MultiPropertyBooking struct {
	ListingID  uuid.UUID
	CheckIn    time.Time
	CheckOut   time.Time
	GuestCount int
}

// ListingHooks interface for property module integration
type ListingHooks interface {
	// GetListingOwner returns the owner ID for a listing
	GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error)

	// GetListingPricing returns base pricing info from listing
	GetListingPricing(ctx context.Context, listingID uuid.UUID) (*ListingPricing, error)
}

// DiscountType represents the type of discount
type DiscountType string

const (
	DiscountTypeFlat         DiscountType = "flat"
	DiscountTypeLengthOfStay DiscountType = "length_of_stay"
)

type Discount struct {
	Name       string       `json:"name"`                 // e.g., "Flash Sale", "Weekly Discount"
	Type       DiscountType `json:"type"`                 // "flat" or "length_of_stay"
	Percentage float64      `json:"percentage"`           // e.g., 10.0 for 10%
	MinNights  *int         `json:"min_nights,omitempty"` // Required if Type is "length_of_stay"
	Active     bool         `json:"active"`               // Corresponds to the toggle switch
}

// CustomFee represents a fee associated with a listing
type CustomFee struct {
	Name         string
	Amount       float64
	Frequency    string // "one_time", "per_night", "per_month", "per_year"
	Category     string // "legal", "agency", "service", "caution", "other"
	IsRefundable bool
	IsOptional   bool
}

// ListingPricing represents pricing info from a listing
type ListingPricing struct {
	ListingID      uuid.UUID
	Currency       string
	BaseRate       float64 // Nightly or rental rate
	Fees           []CustomFee
	Discounts      []Discount
	BaseGuestCount *int
}

type PricingServiceImpl struct {
	repo           repository.PricingRepository
	cache          redis.RedisClient
	listingHooks   ListingHooks
	log            *slog.Logger
	platformConfig config.PlatformYAMLConfig
}

func NewPricingService(
	repo repository.PricingRepository,
	cache redis.RedisClient,
	listingHooks ListingHooks,
	log *slog.Logger,
	platformConfig config.PlatformYAMLConfig,
) PricingService {
	return &PricingServiceImpl{
		repo:           repo,
		cache:          cache,
		listingHooks:   listingHooks,
		log:            log,
		platformConfig: platformConfig,
	}
}
