package domain

import (
	"time"

	"github.com/google/uuid"
)

// PricingRule represents a pricing rule in the domain model
type PricingRule struct {
	ID          uuid.UUID  `json:"id"`
	ListingID   uuid.UUID  `json:"listing_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	RuleType    RuleType   `json:"rule_type"`
	Active      bool       `json:"active"`
	Priority    int        `json:"priority"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	DaysOfWeek  *string    `json:"days_of_week,omitempty"`

	ModifierType  ModifierType `json:"modifier_type"`
	ModifierValue float64      `json:"modifier_value"`

	MinNights       *int `json:"min_nights,omitempty"`
	MaxNights       *int `json:"max_nights,omitempty"`
	MinLeadTimeDays *int `json:"min_lead_time_days,omitempty"`
	MaxLeadTimeDays *int `json:"max_lead_time_days,omitempty"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// MultiPropertyDiscount represents a multi-property discount
type MultiPropertyDiscount struct {
	ID              uuid.UUID   `json:"id"`
	OwnerID         uuid.UUID   `json:"owner_id"`
	Name            string      `json:"name"`
	Description     *string     `json:"description,omitempty"`
	ListingIDs      []uuid.UUID `json:"listing_ids"`
	MinProperties   int         `json:"min_properties"`
	DiscountPercent int         `json:"discount_percent"`
	ValidFrom       *time.Time  `json:"valid_from,omitempty"`
	ValidUntil      *time.Time  `json:"valid_until,omitempty"`
	Active          bool        `json:"active"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	DeletedAt       *time.Time  `json:"deleted_at,omitempty"`
}

// PriceBreakdown represents the detailed price calculation result
type PriceBreakdown struct {
	ListingID  uuid.UUID `json:"listing_id"`
	CheckIn    time.Time `json:"check_in"`
	CheckOut   time.Time `json:"check_out"`
	Nights     int       `json:"nights"`
	GuestCount int       `json:"guest_count"`

	BaseTotal     float64  `json:"base_total"`
	CleaningFee   *float64 `json:"cleaning_fee,omitempty"`
	ServiceFee    *float64 `json:"service_fee,omitempty"`
	CautionFee    *float64 `json:"caution_fee,omitempty"`
	ExtraGuestFee float64  `json:"extra_guest_fee"`

	Discounts  []Discount  `json:"discounts,omitempty"`
	DailyRates []DailyRate `json:"daily_rates"`

	Subtotal     float64               `json:"subtotal"`
	Total        float64               `json:"total"`
	Currency     string                `json:"currency"`
	PlatformFees *PlatformFeeBreakdown `json:"platform_fees,omitempty"`

	CalculatedAt time.Time `json:"calculated_at"`
	ValidUntil   time.Time `json:"valid_until"` // 24h validity
}

// PlatformFeeBreakdown captures Hauslet fee components.
type PlatformFeeBreakdown struct {
	GuestFeePercent         float64 `json:"guest_fee_percent"`
	GuestFeeAmount          float64 `json:"guest_fee_amount"`
	HostCommissionPercent   float64 `json:"host_commission_percent"`
	HostCommissionAmount    float64 `json:"host_commission_amount"`
	PayoutProcessingPercent float64 `json:"payout_processing_percent"`
	PayoutProcessingAmount  float64 `json:"payout_processing_amount"`
	MinimumGuestFeeApplied  bool    `json:"minimum_guest_fee_applied"`
	HostNetAmount           float64 `json:"host_net_amount"`
}

// Discount represents a discount applied to a booking
type Discount struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"` // "percentage" or "fixed"
}

// DailyRate represents the nightly rate for a specific date
type DailyRate struct {
	Date         time.Time `json:"date"`
	BaseRate     float64   `json:"base_rate"`
	AppliedRules []string  `json:"applied_rules,omitempty"` // Rule names that affected this date
	FinalRate    float64   `json:"final_rate"`
}

// --- DOMAIN METHODS ---

// IsActive checks if the rule is currently active
func (r *PricingRule) IsActive() bool {
	return r.Active && r.DeletedAt == nil
}

// AppliesToDate checks if the rule applies to a specific date
func (r *PricingRule) AppliesToDate(date time.Time) bool {
	if !r.IsActive() {
		return false
	}

	// Check date range
	if r.StartDate != nil && date.Before(*r.StartDate) {
		return false
	}
	if r.EndDate != nil && date.After(*r.EndDate) {
		return false
	}

	if r.DaysOfWeek != nil {
		currentDay := int(date.Weekday())
		spec := *r.DaysOfWeek
		if spec == "" {
			return false
		}

		match := false
		for i := 0; i < len(spec); {
			ch := spec[i]
			if ch < '0' || ch > '9' {
				i++
				continue
			}

			val := 0
			for i < len(spec) && spec[i] >= '0' && spec[i] <= '9' {
				val = val*10 + int(spec[i]-'0')
				i++
			}

			if val == currentDay {
				match = true
				break
			}
		}

		if !match {
			return false
		}
	}

	return true
}

// AppliesToStayLength checks if the rule applies to a given stay length
func (r *PricingRule) AppliesToStayLength(nights int) bool {
	if r.MinNights != nil && nights < *r.MinNights {
		return false
	}
	if r.MaxNights != nil && nights > *r.MaxNights {
		return false
	}
	return true
}

// AppliesToLeadTime checks if the rule applies to a given lead time
func (r *PricingRule) AppliesToLeadTime(leadTimeDays int) bool {
	if r.MinLeadTimeDays != nil && leadTimeDays < *r.MinLeadTimeDays {
		return false
	}
	if r.MaxLeadTimeDays != nil && leadTimeDays > *r.MaxLeadTimeDays {
		return false
	}
	return true
}

// ApplyModifier applies this rule's modifier to a base price
func (r *PricingRule) ApplyModifier(basePrice float64) float64 {
	switch r.ModifierType {
	case ModifierPercentage:
		return basePrice * (1 + r.ModifierValue/100)
	case ModifierFixed:
		return basePrice + r.ModifierValue
	case ModifierAbsolute:
		return r.ModifierValue
	default:
		return basePrice
	}
}

// GetPriority returns the effective priority for rule resolution
func (r *PricingRule) GetPriority() int {
	// Automatic priority based on rule type specificity
	basePriority := r.Priority

	switch r.RuleType {
	case RuleTypeCustom:
		basePriority += 1000 // Highest priority - exact date match
	case RuleTypeHoliday:
		basePriority += 800
	case RuleTypeWeekend, RuleTypeWeekday:
		basePriority += 600
	case RuleTypeSeasonal:
		basePriority += 400
	case RuleTypeLengthOfStay:
		basePriority += 200
	case RuleTypeLastMinute, RuleTypeEarlyBird:
		basePriority += 100
	case RuleTypeBase:
		basePriority += 0 // Lowest priority
	}

	return basePriority
}

// IsValid checks if the discount is currently valid
func (d *MultiPropertyDiscount) IsValid() bool {
	if !d.Active || d.DeletedAt != nil {
		return false
	}

	now := time.Now()
	if d.ValidFrom != nil && now.Before(*d.ValidFrom) {
		return false
	}
	if d.ValidUntil != nil && now.After(*d.ValidUntil) {
		return false
	}

	return true
}

// QualifiesForDiscount checks if the given listing IDs qualify for this discount
func (d *MultiPropertyDiscount) QualifiesForDiscount(listingIDs []uuid.UUID) bool {
	if !d.IsValid() {
		return false
	}

	// Count how many of the provided listings are in the discount
	matchCount := 0
	for _, listingID := range listingIDs {
		for _, discountListingID := range d.ListingIDs {
			if listingID == discountListingID {
				matchCount++
				break
			}
		}
	}

	return matchCount >= d.MinProperties
}

// CalculateDiscountAmount calculates the discount amount for a given total
func (d *MultiPropertyDiscount) CalculateDiscountAmount(total float64) float64 {
	return total * (float64(d.DiscountPercent) / 100.0)
}

// IsExpired checks if the price quote has expired
func (p *PriceBreakdown) IsExpired() bool {
	return time.Now().After(p.ValidUntil)
}

// GetTotalNights returns the number of nights
func (p *PriceBreakdown) GetTotalNights() int {
	return p.Nights
}

// GetAverageNightlyRate returns the average rate per night
func (p *PriceBreakdown) GetAverageNightlyRate() float64 {
	if p.Nights == 0 {
		return 0
	}
	return p.BaseTotal / float64(p.Nights)
}

// GetTotalDiscounts returns the sum of all discounts
func (p *PriceBreakdown) GetTotalDiscounts() float64 {
	total := 0.0
	for _, discount := range p.Discounts {
		total += discount.Amount
	}
	return total
}
