package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/pricing/domain"
	"sort"
	"time"

	"github.com/google/uuid"
)

// --- Core Price Calculation ---

func (s *PricingServiceImpl) CalculatePrice(ctx context.Context, listingID uuid.UUID, checkIn, checkOut time.Time, guestCount int) (*domain.PriceBreakdown, error) {
	if s.log != nil {
		s.log.Logf("INFO calculating price listing=%s checkin=%s checkout=%s guests=%d", listingID, checkIn.Format("2006-01-02"), checkOut.Format("2006-01-02"), guestCount)
	}

	if s.listingHooks == nil {
		return nil, fmt.Errorf("pricing listing hooks not configured")
	}

	// Validate date range
	if !checkOut.After(checkIn) {
		return nil, domain.ErrInvalidDateRange
	}

	// Get listing pricing info (base rate, fees, currency)
	listingPricing, err := s.listingHooks.GetListingPricing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get listing pricing: %w", err)
	}

	// Calculate number of nights
	nights := int(checkOut.Sub(checkIn).Hours() / 24)

	// Get all active pricing rules for the listing
	schemaRules, err := s.repo.GetRulesForListing(ctx, listingID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing rules: %w", err)
	}

	rules := domain.MapRulesFromSchema(schemaRules)

	// Calculate daily rates
	dailyRates := s.calculateDailyRates(listingPricing.BaseRate, checkIn, checkOut, rules)

	// Sum up base total
	baseTotal := 0.0
	for _, rate := range dailyRates {
		baseTotal += rate.FinalRate
	}

	// Apply length-of-stay discounts
	discounts := s.calculateDiscounts(baseTotal, nights, rules)

	// Calculate subtotal after discounts
	subtotal := baseTotal
	for _, discount := range discounts {
		subtotal -= discount.Amount
	}

	// Calculate extra guest fee if applicable
	extraGuestFee := 0.0
	if listingPricing.BaseGuestCount != nil && guestCount > *listingPricing.BaseGuestCount {
		extraGuests := guestCount - *listingPricing.BaseGuestCount
		if listingPricing.ExtraGuestFee != nil {
			extraGuestFee = float64(extraGuests) * *listingPricing.ExtraGuestFee * float64(nights)
		}
	}

	// Calculate total
	grossBeforePlatform := subtotal + extraGuestFee
	if listingPricing.CleaningFee != nil {
		grossBeforePlatform += *listingPricing.CleaningFee
	}
	if listingPricing.ServiceFee != nil {
		grossBeforePlatform += *listingPricing.ServiceFee
	}

	var platformFees *domain.PlatformFeeBreakdown
	guestServiceFee := 0.0
	if s.isFeesEnabled() {
		guestFee, minApplied := s.calculateGuestServiceFee(grossBeforePlatform)
		hostCommission := s.calculateHostCommission(grossBeforePlatform)
		hostRevenueBeforeProcessing := grossBeforePlatform - hostCommission
		payoutProcessing := s.calculatePayoutProcessing(hostRevenueBeforeProcessing)
		hostNet := hostRevenueBeforeProcessing - payoutProcessing
		if hostNet < 0 {
			hostNet = 0
		}

		platformFees = &domain.PlatformFeeBreakdown{
			GuestFeePercent:         s.platformConfig.Fees.GuestServicePercent,
			GuestFeeAmount:          guestFee,
			HostCommissionPercent:   s.platformConfig.Fees.HostCommissionPercent,
			HostCommissionAmount:    hostCommission,
			PayoutProcessingPercent: s.platformConfig.Fees.PayoutProcessingPercent,
			PayoutProcessingAmount:  payoutProcessing,
			MinimumGuestFeeApplied:  minApplied,
			HostNetAmount:           hostNet,
		}
		guestServiceFee = guestFee
	}

	total := grossBeforePlatform + guestServiceFee

	breakdown := &domain.PriceBreakdown{
		ListingID:     listingID,
		CheckIn:       checkIn,
		CheckOut:      checkOut,
		Nights:        nights,
		GuestCount:    guestCount,
		BaseTotal:     baseTotal,
		CleaningFee:   listingPricing.CleaningFee,
		ServiceFee:    listingPricing.ServiceFee,
		CautionFee:    listingPricing.CautionFee,
		Discounts:     discounts,
		DailyRates:    dailyRates,
		Subtotal:      subtotal,
		ExtraGuestFee: extraGuestFee,
		Total:         total,
		Currency:      listingPricing.Currency,
		PlatformFees:  platformFees,
		CalculatedAt:  time.Now(),
		ValidUntil:    time.Now().Add(24 * time.Hour), // 24h validity
	}

	// Cache the breakdown
	s.cachePriceBreakdown(ctx, breakdown)

	if s.log != nil {
		s.log.Logf("INFO calculated price total=%.2f %s nights=%d", total, listingPricing.Currency, nights)
	}

	return breakdown, nil
}

func (s *PricingServiceImpl) GetBasePrice(ctx context.Context, listingID uuid.UUID) (float64, string, error) {
	// Try cache first
	cacheKey := basePriceCacheKey(listingID)
	var cached struct {
		Price    float64
		Currency string
	}
	if found, err := s.getCachedValue(ctx, cacheKey, &cached); err == nil && found {
		return cached.Price, cached.Currency, nil
	}

	// Get from listing
	if s.listingHooks == nil {
		return 0, "", fmt.Errorf("pricing listing hooks not configured")
	}
	listingPricing, err := s.listingHooks.GetListingPricing(ctx, listingID)
	if err != nil {
		return 0, "", err
	}

	// Cache for future requests
	s.cacheBasePrice(ctx, listingID, listingPricing.BaseRate, listingPricing.Currency)

	return listingPricing.BaseRate, listingPricing.Currency, nil
}

func (s *PricingServiceImpl) PreviewPricing(ctx context.Context, listingID uuid.UUID, month time.Time) ([]domain.DailyRate, error) {
	// Get start and end of month
	startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	// Get listing base rate
	if s.listingHooks == nil {
		return nil, fmt.Errorf("pricing listing hooks not configured")
	}
	listingPricing, err := s.listingHooks.GetListingPricing(ctx, listingID)
	if err != nil {
		return nil, err
	}

	// Get pricing rules
	schemaRules, err := s.repo.GetRulesForListing(ctx, listingID, true)
	if err != nil {
		return nil, err
	}

	rules := domain.MapRulesFromSchema(schemaRules)

	// Calculate daily rates for the month
	dailyRates := s.calculateDailyRates(listingPricing.BaseRate, startOfMonth, endOfMonth, rules)

	return dailyRates, nil
}

// --- Pricing Rule Management ---

func (s *PricingServiceImpl) CreateRule(ctx context.Context, rule *domain.PricingRule) (*domain.PricingRule, error) {
	if s.log != nil {
		s.log.Logf("INFO creating pricing rule listing=%s type=%s", rule.ListingID, rule.RuleType)
	}

	schemaRule := domain.MapRuleFromEntityToSchema(rule)
	if err := s.repo.CreateRule(ctx, schemaRule); err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	rule.ID = schemaRule.ID
	rule.CreatedAt = schemaRule.CreatedAt
	rule.UpdatedAt = schemaRule.UpdatedAt

	// Invalidate price caches for this listing
	s.invalidatePriceCaches(ctx, rule.ListingID)

	return rule, nil
}

func (s *PricingServiceImpl) GetRule(ctx context.Context, ruleID uuid.UUID, requestorID uuid.UUID) (*domain.PricingRule, error) {
	schemaRule, err := s.repo.GetRuleByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}

	// Verify access
	if s.listingHooks == nil {
		return nil, fmt.Errorf("pricing listing hooks not configured")
	}
	ownerID, err := s.listingHooks.GetListingOwner(ctx, schemaRule.ListingID)
	if err != nil {
		return nil, err
	}
	if ownerID != requestorID {
		return nil, domain.ErrUnauthorized
	}

	return domain.MapRuleFromSchemaToEntity(schemaRule), nil
}

func (s *PricingServiceImpl) GetRulesForListing(ctx context.Context, listingID uuid.UUID, activeOnly bool) ([]*domain.PricingRule, error) {
	schemaRules, err := s.repo.GetRulesForListing(ctx, listingID, activeOnly)
	if err != nil {
		return nil, err
	}
	return domain.MapRulesFromSchema(schemaRules), nil
}

func (s *PricingServiceImpl) UpdateRule(ctx context.Context, rule *domain.PricingRule, requestorID uuid.UUID) (*domain.PricingRule, error) {
	// Verify access
	if s.listingHooks == nil {
		return nil, fmt.Errorf("pricing listing hooks not configured")
	}
	ownerID, err := s.listingHooks.GetListingOwner(ctx, rule.ListingID)
	if err != nil {
		return nil, err
	}
	if ownerID != requestorID {
		return nil, domain.ErrUnauthorized
	}

	schemaRule := domain.MapRuleFromEntityToSchema(rule)
	if err := s.repo.UpdateRule(ctx, schemaRule); err != nil {
		return nil, err
	}

	rule.UpdatedAt = schemaRule.UpdatedAt

	// Invalidate caches
	s.invalidatePriceCaches(ctx, rule.ListingID)

	return rule, nil
}

func (s *PricingServiceImpl) DeleteRule(ctx context.Context, ruleID uuid.UUID, requestorID uuid.UUID) error {
	// Get rule first to verify access
	rule, err := s.GetRule(ctx, ruleID, requestorID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteRule(ctx, ruleID); err != nil {
		return err
	}

	// Invalidate caches
	s.invalidatePriceCaches(ctx, rule.ListingID)

	return nil
}

// --- Multi-Property Discounts ---

func (s *PricingServiceImpl) CreateMultiPropertyDiscount(ctx context.Context, discount *domain.MultiPropertyDiscount) (*domain.MultiPropertyDiscount, error) {
	schemaDiscount := domain.MapDiscountFromEntityToSchema(discount)
	if err := s.repo.CreateDiscount(ctx, schemaDiscount); err != nil {
		return nil, err
	}

	discount.ID = schemaDiscount.ID
	discount.CreatedAt = schemaDiscount.CreatedAt
	discount.UpdatedAt = schemaDiscount.UpdatedAt

	return discount, nil
}

func (s *PricingServiceImpl) GetDiscountsForOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.MultiPropertyDiscount, error) {
	schemaDiscounts, err := s.repo.GetDiscountsForOwner(ctx, ownerID, true)
	if err != nil {
		return nil, err
	}

	discounts := make([]*domain.MultiPropertyDiscount, len(schemaDiscounts))
	for i, sd := range schemaDiscounts {
		discounts[i] = domain.MapDiscountFromSchemaToEntity(sd)
	}

	return discounts, nil
}

func (s *PricingServiceImpl) CalculateMultiPropertyPrice(ctx context.Context, bookings []MultiPropertyBooking) (*domain.PriceBreakdown, error) {
	// Calculate individual prices
	individualPrices := make([]*domain.PriceBreakdown, len(bookings))
	listingIDs := make([]uuid.UUID, len(bookings))

	for i, booking := range bookings {
		price, err := s.CalculatePrice(ctx, booking.ListingID, booking.CheckIn, booking.CheckOut, booking.GuestCount)
		if err != nil {
			return nil, err
		}
		individualPrices[i] = price
		listingIDs[i] = booking.ListingID
	}

	// Calculate subtotal
	subtotal := 0.0
	for _, price := range individualPrices {
		subtotal += price.Total
	}

	// Check for multi-property discount
	discount, err := s.repo.GetDiscountForListings(ctx, listingIDs)
	if err == nil && discount != nil {
		domainDiscount := domain.MapDiscountFromSchemaToEntity(discount)
		if domainDiscount.QualifiesForDiscount(listingIDs) {
			discountAmount := domainDiscount.CalculateDiscountAmount(subtotal)

			// Create combined breakdown
			return &domain.PriceBreakdown{
				Subtotal: subtotal,
				Discounts: []domain.Discount{
					{
						Name:   domainDiscount.Name,
						Amount: discountAmount,
						Type:   "percentage",
					},
				},
				Total:        subtotal - discountAmount,
				Currency:     individualPrices[0].Currency,
				CalculatedAt: time.Now(),
				ValidUntil:   time.Now().Add(24 * time.Hour),
			}, nil
		}
	}

	// No discount applicable
	return &domain.PriceBreakdown{
		Subtotal:     subtotal,
		Total:        subtotal,
		Currency:     individualPrices[0].Currency,
		CalculatedAt: time.Now(),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}, nil
}

func (s *PricingServiceImpl) UpdateDiscount(ctx context.Context, discount *domain.MultiPropertyDiscount) (*domain.MultiPropertyDiscount, error) {
	schemaDiscount := domain.MapDiscountFromEntityToSchema(discount)
	if err := s.repo.UpdateDiscount(ctx, schemaDiscount); err != nil {
		return nil, err
	}

	discount.UpdatedAt = schemaDiscount.UpdatedAt
	return discount, nil
}

func (s *PricingServiceImpl) DeleteDiscount(ctx context.Context, discountID uuid.UUID, ownerID uuid.UUID) error {
	// Verify ownership
	discount, err := s.repo.GetDiscountByID(ctx, discountID)
	if err != nil {
		return err
	}

	if discount.OwnerID != ownerID {
		return domain.ErrUnauthorized
	}

	return s.repo.DeleteDiscount(ctx, discountID)
}

// --- Helper Methods ---

func (s *PricingServiceImpl) calculateDailyRates(baseRate float64, checkIn, checkOut time.Time, rules []*domain.PricingRule) []domain.DailyRate {
	dailyRates := make([]domain.DailyRate, 0)

	// Iterate through each night
	for date := checkIn; date.Before(checkOut); date = date.AddDate(0, 0, 1) {
		rate := s.calculateRateForDate(baseRate, date, rules)
		dailyRates = append(dailyRates, rate)
	}

	return dailyRates
}

func (s *PricingServiceImpl) calculateRateForDate(baseRate float64, date time.Time, rules []*domain.PricingRule) domain.DailyRate {
	applicableRules := make([]*domain.PricingRule, 0)

	// Find all rules that apply to this date
	for _, rule := range rules {
		if rule.AppliesToDate(date) {
			applicableRules = append(applicableRules, rule)
		}
	}

	// Sort by priority (highest first)
	sort.Slice(applicableRules, func(i, j int) bool {
		return applicableRules[i].GetPriority() > applicableRules[j].GetPriority()
	})

	// Apply highest priority rule
	finalRate := baseRate
	appliedRuleNames := make([]string, 0)

	if len(applicableRules) > 0 {
		topRule := applicableRules[0]
		finalRate = topRule.ApplyModifier(baseRate)
		appliedRuleNames = append(appliedRuleNames, topRule.Name)
	}

	return domain.DailyRate{
		Date:         date,
		BaseRate:     baseRate,
		AppliedRules: appliedRuleNames,
		FinalRate:    finalRate,
	}
}

func (s *PricingServiceImpl) calculateDiscounts(baseTotal float64, nights int, rules []*domain.PricingRule) []domain.Discount {
	discounts := make([]domain.Discount, 0)

	// Find length-of-stay discount rules
	for _, rule := range rules {
		if rule.RuleType == domain.RuleTypeLengthOfStay && rule.AppliesToStayLength(nights) {
			amount := 0.0
			switch rule.ModifierType {
			case domain.ModifierPercentage:
				// For discounts, percentage is negative
				amount = baseTotal * (rule.ModifierValue / 100)
			case domain.ModifierFixed:
				amount = rule.ModifierValue
			}

			if amount > 0 {
				discounts = append(discounts, domain.Discount{
					Name:   rule.Name,
					Amount: amount,
					Type:   string(rule.ModifierType),
				})
			}
		}
	}

	return discounts
}
