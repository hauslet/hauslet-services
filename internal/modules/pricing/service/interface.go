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

// ListingPricing represents pricing info from a listing
type ListingPricing struct {
	ListingID      uuid.UUID
	Currency       string
	BaseRate       float64 // Nightly or rental rate
	CleaningFee    *float64
	ServiceFee     *float64
	CautionFee     *float64
	ExtraGuestFee  *float64
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
