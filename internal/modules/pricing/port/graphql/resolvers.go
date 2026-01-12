package graphql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hauslet/internal/modules/pricing/domain"
	"hauslet/internal/modules/pricing/service"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles pricing-specific GraphQL fields.
type Resolver struct {
	pricingService service.PricingService
	log            *slog.Logger
}

func NewResolver(pricingService service.PricingService, log *slog.Logger) *Resolver {
	return &Resolver{
		pricingService: pricingService,
		log:            log,
	}
}

// ===========================
// QUERY RESOLVERS
// ===========================

// PricingRule retrieves a pricing rule by ID
func (r *Resolver) PricingRule(ctx context.Context, id uuid.UUID) (*domain.PricingRule, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	rule, err := r.pricingService.GetRule(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to get pricing rule", "rule_id", id, "error", err)
		return nil, err
	}

	return rule, nil
}

// PricingRulesForListing retrieves all pricing rules for a listing
func (r *Resolver) PricingRulesForListing(ctx context.Context, listingID uuid.UUID, activeOnly *bool) ([]*domain.PricingRule, error) {
	active := false
	if activeOnly != nil {
		active = *activeOnly
	}

	rules, err := r.pricingService.GetRulesForListing(ctx, listingID, active)
	if err != nil {
		r.log.Error("failed to get pricing rules for listing", "listing_id", listingID, "error", err)
		return nil, err
	}

	return rules, nil
}

// MultiPropertyDiscountsForOwner retrieves all multi-property discounts for the current user
func (r *Resolver) MultiPropertyDiscountsForOwner(ctx context.Context) ([]*domain.MultiPropertyDiscount, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	discounts, err := r.pricingService.GetDiscountsForOwner(ctx, userID)
	if err != nil {
		r.log.Error("failed to get multi-property discounts", "owner_id", userID, "error", err)
		return nil, err
	}

	return discounts, nil
}

// CalculatePrice calculates the total price for a booking
func (r *Resolver) CalculatePrice(ctx context.Context, listingID uuid.UUID, checkIn, checkOut time.Time, guestCount int) (*domain.PriceBreakdown, error) {
	breakdown, err := r.pricingService.CalculatePrice(ctx, listingID, checkIn, checkOut, guestCount)
	if err != nil {
		r.log.Error("failed to calculate price", "listing_id", listingID, "error", err)
		return nil, err
	}

	return breakdown, nil
}

// CalculateMultiPropertyPrice calculates price for multiple properties
func (r *Resolver) CalculateMultiPropertyPrice(ctx context.Context, bookings []MultiPropertyBookingInput) (*domain.PriceBreakdown, error) {
	// Convert GraphQL input to service input
	serviceBookings := make([]service.MultiPropertyBooking, len(bookings))
	for i, b := range bookings {
		serviceBookings[i] = service.MultiPropertyBooking{
			ListingID:  b.ListingID,
			CheckIn:    b.CheckIn,
			CheckOut:   b.CheckOut,
			GuestCount: b.GuestCount,
		}
	}

	breakdown, err := r.pricingService.CalculateMultiPropertyPrice(ctx, serviceBookings)
	if err != nil {
		r.log.Error("failed to calculate multi-property price", "error", err)
		return nil, err
	}

	return breakdown, nil
}

// BasePrice returns the base nightly rate for a listing
func (r *Resolver) BasePrice(ctx context.Context, listingID uuid.UUID) (*BasePriceResult, error) {
	baseRate, currency, err := r.pricingService.GetBasePrice(ctx, listingID)
	if err != nil {
		r.log.Error("failed to get base price", "listing_id", listingID, "error", err)
		return nil, err
	}

	return &BasePriceResult{
		BaseRate: baseRate,
		Currency: currency,
	}, nil
}

// PreviewPricing returns daily rates for a month
func (r *Resolver) PreviewPricing(ctx context.Context, listingID uuid.UUID, month time.Time) ([]domain.DailyRate, error) {
	rates, err := r.pricingService.PreviewPricing(ctx, listingID, month)
	if err != nil {
		r.log.Error("failed to preview pricing", "listing_id", listingID, "error", err)
		return nil, err
	}

	return rates, nil
}

// ===========================
// MUTATION RESOLVERS
// ===========================

// CreatePricingRule creates a new pricing rule
func (r *Resolver) CreatePricingRule(ctx context.Context, input CreatePricingRuleInput) (*domain.PricingRule, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	rule := &domain.PricingRule{
		ID:              uuid.New(),
		ListingID:       input.ListingID,
		Name:            input.Name,
		Description:     input.Description,
		RuleType:        input.RuleType,
		Active:          input.Active,
		Priority:        input.Priority,
		StartDate:       input.StartDate,
		EndDate:         input.EndDate,
		DaysOfWeek:      input.DaysOfWeek,
		ModifierType:    input.ModifierType,
		ModifierValue:   input.ModifierValue,
		MinNights:       input.MinNights,
		MaxNights:       input.MaxNights,
		MinLeadTimeDays: input.MinLeadTimeDays,
		MaxLeadTimeDays: input.MaxLeadTimeDays,
		CreatedBy:       &userID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	createdRule, err := r.pricingService.CreateRule(ctx, rule)
	if err != nil {
		r.log.Error("failed to create pricing rule", "error", err)
		return nil, err
	}

	return createdRule, nil
}

// UpdatePricingRule updates an existing pricing rule
func (r *Resolver) UpdatePricingRule(ctx context.Context, input UpdatePricingRuleInput) (*domain.PricingRule, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get existing rule
	existingRule, err := r.pricingService.GetRule(ctx, input.ID, userID)
	if err != nil {
		r.log.Error("failed to get existing pricing rule", "rule_id", input.ID, "error", err)
		return nil, err
	}

	// Update fields that are provided
	if input.Name != nil {
		existingRule.Name = *input.Name
	}
	if input.Description != nil {
		existingRule.Description = input.Description
	}
	if input.RuleType != nil {
		existingRule.RuleType = *input.RuleType
	}
	if input.Active != nil {
		existingRule.Active = *input.Active
	}
	if input.Priority != nil {
		existingRule.Priority = *input.Priority
	}
	if input.StartDate != nil {
		existingRule.StartDate = input.StartDate
	}
	if input.EndDate != nil {
		existingRule.EndDate = input.EndDate
	}
	if input.DaysOfWeek != nil {
		existingRule.DaysOfWeek = input.DaysOfWeek
	}
	if input.ModifierType != nil {
		existingRule.ModifierType = *input.ModifierType
	}
	if input.ModifierValue != nil {
		existingRule.ModifierValue = *input.ModifierValue
	}
	if input.MinNights != nil {
		existingRule.MinNights = input.MinNights
	}
	if input.MaxNights != nil {
		existingRule.MaxNights = input.MaxNights
	}
	if input.MinLeadTimeDays != nil {
		existingRule.MinLeadTimeDays = input.MinLeadTimeDays
	}
	if input.MaxLeadTimeDays != nil {
		existingRule.MaxLeadTimeDays = input.MaxLeadTimeDays
	}

	existingRule.UpdatedBy = &userID
	existingRule.UpdatedAt = time.Now()

	updatedRule, err := r.pricingService.UpdateRule(ctx, existingRule, userID)
	if err != nil {
		r.log.Error("failed to update pricing rule", "rule_id", input.ID, "error", err)
		return nil, err
	}

	return updatedRule, nil
}

// DeletePricingRule deletes a pricing rule
func (r *Resolver) DeletePricingRule(ctx context.Context, id uuid.UUID) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	err = r.pricingService.DeleteRule(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to delete pricing rule", "rule_id", id, "error", err)
		return false, err
	}

	return true, nil
}

// CreateMultiPropertyDiscount creates a new multi-property discount
func (r *Resolver) CreateMultiPropertyDiscount(ctx context.Context, input CreateMultiPropertyDiscountInput) (*domain.MultiPropertyDiscount, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	discount := &domain.MultiPropertyDiscount{
		ID:              uuid.New(),
		OwnerID:         userID,
		Name:            input.Name,
		Description:     input.Description,
		ListingIDs:      input.ListingIds,
		MinProperties:   input.MinProperties,
		DiscountPercent: input.DiscountPercent,
		ValidFrom:       input.ValidFrom,
		ValidUntil:      input.ValidUntil,
		Active:          input.Active,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	createdDiscount, err := r.pricingService.CreateMultiPropertyDiscount(ctx, discount)
	if err != nil {
		r.log.Error("failed to create multi-property discount", "error", err)
		return nil, err
	}

	return createdDiscount, nil
}

// UpdateMultiPropertyDiscount updates an existing multi-property discount
func (r *Resolver) UpdateMultiPropertyDiscount(ctx context.Context, input UpdateMultiPropertyDiscountInput) (*domain.MultiPropertyDiscount, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get existing discount
	discounts, err := r.pricingService.GetDiscountsForOwner(ctx, userID)
	if err != nil {
		r.log.Error("failed to get discounts for owner", "owner_id", userID, "error", err)
		return nil, err
	}

	var existingDiscount *domain.MultiPropertyDiscount
	for _, d := range discounts {
		if d.ID == input.ID {
			existingDiscount = d
			break
		}
	}

	if existingDiscount == nil {
		return nil, fmt.Errorf("multi-property discount not found")
	}

	// Update fields that are provided
	if input.Name != nil {
		existingDiscount.Name = *input.Name
	}
	if input.Description != nil {
		existingDiscount.Description = input.Description
	}
	if input.ListingIds != nil {
		existingDiscount.ListingIDs = input.ListingIds
	}
	if input.MinProperties != nil {
		existingDiscount.MinProperties = *input.MinProperties
	}
	if input.DiscountPercent != nil {
		existingDiscount.DiscountPercent = *input.DiscountPercent
	}
	if input.ValidFrom != nil {
		existingDiscount.ValidFrom = input.ValidFrom
	}
	if input.ValidUntil != nil {
		existingDiscount.ValidUntil = input.ValidUntil
	}
	if input.Active != nil {
		existingDiscount.Active = *input.Active
	}

	existingDiscount.UpdatedAt = time.Now()

	updatedDiscount, err := r.pricingService.UpdateDiscount(ctx, existingDiscount)
	if err != nil {
		r.log.Error("failed to update multi-property discount", "discount_id", input.ID, "error", err)
		return nil, err
	}

	return updatedDiscount, nil
}

// DeleteMultiPropertyDiscount deletes a multi-property discount
func (r *Resolver) DeleteMultiPropertyDiscount(ctx context.Context, id uuid.UUID) (bool, error) {
	userID, err := viewer.GetUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	err = r.pricingService.DeleteDiscount(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to delete multi-property discount", "discount_id", id, "error", err)
		return false, err
	}

	return true, nil
}

// ===========================
// HELPER TYPES
// ===========================

// BasePriceResult represents the base price result
type BasePriceResult struct {
	BaseRate float64
	Currency string
}

// MultiPropertyBookingInput represents input for multi-property price calculation
type MultiPropertyBookingInput struct {
	ListingID  uuid.UUID
	CheckIn    time.Time
	CheckOut   time.Time
	GuestCount int
}

// CreatePricingRuleInput represents input for creating a pricing rule
type CreatePricingRuleInput struct {
	ListingID       uuid.UUID
	Name            string
	Description     *string
	RuleType        domain.RuleType
	Active          bool
	Priority        int
	StartDate       *time.Time
	EndDate         *time.Time
	DaysOfWeek      *string
	ModifierType    domain.ModifierType
	ModifierValue   float64
	MinNights       *int
	MaxNights       *int
	MinLeadTimeDays *int
	MaxLeadTimeDays *int
}

// UpdatePricingRuleInput represents input for updating a pricing rule
type UpdatePricingRuleInput struct {
	ID              uuid.UUID
	Name            *string
	Description     *string
	RuleType        *domain.RuleType
	Active          *bool
	Priority        *int
	StartDate       *time.Time
	EndDate         *time.Time
	DaysOfWeek      *string
	ModifierType    *domain.ModifierType
	ModifierValue   *float64
	MinNights       *int
	MaxNights       *int
	MinLeadTimeDays *int
	MaxLeadTimeDays *int
}

// CreateMultiPropertyDiscountInput represents input for creating a multi-property discount
type CreateMultiPropertyDiscountInput struct {
	Name            string
	Description     *string
	ListingIds      []uuid.UUID
	MinProperties   int
	DiscountPercent int
	ValidFrom       *time.Time
	ValidUntil      *time.Time
	Active          bool
}

// UpdateMultiPropertyDiscountInput represents input for updating a multi-property discount
type UpdateMultiPropertyDiscountInput struct {
	ID              uuid.UUID
	Name            *string
	Description     *string
	ListingIds      []uuid.UUID
	MinProperties   *int
	DiscountPercent *int
	ValidFrom       *time.Time
	ValidUntil      *time.Time
	Active          *bool
}
