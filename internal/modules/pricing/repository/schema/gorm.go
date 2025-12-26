package schema

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Validation Errors ---
var (
	ErrInvalidRuleType      = errors.New("invalid rule type")
	ErrInvalidModifierType  = errors.New("invalid modifier type")
	ErrInvalidModifierValue = errors.New("invalid modifier value")
	ErrListingIDRequired    = errors.New("listing ID is required")
	ErrInvalidPriority      = errors.New("priority must be non-negative")
	ErrInvalidDateRange     = errors.New("start date must be before end date")
	ErrInvalidMinStay       = errors.New("minimum stay must be positive")
)

// PricingRule represents a pricing rule for a listing
type PricingRule struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index"`

	Name        string   `gorm:"type:varchar(255);not null"`
	Description *string  `gorm:"type:text"`
	RuleType    RuleType `gorm:"type:varchar(50);not null;index"`
	Active      bool     `gorm:"default:true;index"`

	// Priority for conflict resolution (higher = takes precedence)
	Priority int `gorm:"default:0;index"`

	// Date range applicability
	StartDate *time.Time `gorm:"index"`
	EndDate   *time.Time `gorm:"index"`

	// Day of week applicability (comma-separated: "0,6" for Sun,Sat)
	DaysOfWeek *string `gorm:"type:varchar(50)"`

	// Modifier configuration
	ModifierType  ModifierType `gorm:"type:varchar(50);not null"`
	ModifierValue float64      `gorm:"not null"`

	// Constraints
	MinNights *int `gorm:"type:int"` // Minimum stay required for this rule
	MaxNights *int `gorm:"type:int"` // Maximum stay for this rule

	// Lead time (for early bird/last minute)
	MinLeadTimeDays *int `gorm:"type:int"` // Minimum days before check-in
	MaxLeadTimeDays *int `gorm:"type:int"` // Maximum days before check-in

	// Metadata
	CreatedBy *uuid.UUID `gorm:"type:uuid"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// MultiPropertyDiscount represents a discount for booking multiple listings
type MultiPropertyDiscount struct {
	ID      uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OwnerID uuid.UUID `gorm:"type:uuid;not null;index"`

	Name        string  `gorm:"type:varchar(255);not null"`
	Description *string `gorm:"type:text"`

	// Listings included in this discount
	ListingIDs []uuid.UUID `gorm:"type:uuid[];not null"`

	// Minimum properties to qualify
	MinProperties int `gorm:"default:2"`

	// Discount configuration
	DiscountPercent int `gorm:"not null"` // 10-30% typical

	// Validity period
	ValidFrom  *time.Time
	ValidUntil *time.Time

	Active bool `gorm:"default:true;index"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// --- GORM Hooks ---

// BeforeSave validates PricingRule before saving
func (r *PricingRule) BeforeSave(tx *gorm.DB) error {
	changed := func(field string) bool {
		return tx != nil && tx.Statement != nil && tx.Statement.Changed(field)
	}

	// 1. Validate RuleType
	if changed("rule_type") {
		switch r.RuleType {
		case RuleTypeBase, RuleTypeSeasonal, RuleTypeWeekend, RuleTypeWeekday,
			RuleTypeHoliday, RuleTypeLengthOfStay, RuleTypeLastMinute,
			RuleTypeEarlyBird, RuleTypeGapFiller, RuleTypeCustom:
		default:
			return ErrInvalidRuleType
		}
	}

	// 2. Validate ModifierType
	if changed("modifier_type") {
		switch r.ModifierType {
		case ModifierPercentage, ModifierFixed, ModifierAbsolute:
		default:
			return ErrInvalidModifierType
		}
	}

	// 3. Validate ModifierValue
	if changed("modifier_value") {
		// For percentage, value should be between -100 and 1000
		if r.ModifierType == ModifierPercentage {
			if r.ModifierValue < -100 || r.ModifierValue > 1000 {
				return fmt.Errorf("percentage modifier must be between -100 and 1000")
			}
		}
		// For absolute price, must be positive
		if r.ModifierType == ModifierAbsolute && r.ModifierValue <= 0 {
			return fmt.Errorf("absolute price must be positive")
		}
	}

	// 4. Validate ListingID
	if r.ListingID == uuid.Nil {
		return ErrListingIDRequired
	}

	// 5. Validate Priority
	if r.Priority < 0 {
		return ErrInvalidPriority
	}

	// 6. Validate Date Range
	if r.StartDate != nil && r.EndDate != nil {
		if !r.StartDate.Before(*r.EndDate) {
			return ErrInvalidDateRange
		}
	}

	// 7. Validate Min/Max Nights
	if r.MinNights != nil && *r.MinNights <= 0 {
		return ErrInvalidMinStay
	}

	if r.MinNights != nil && r.MaxNights != nil && *r.MinNights > *r.MaxNights {
		return errors.New("minimum nights cannot exceed maximum nights")
	}

	return nil
}

// BeforeSave validates MultiPropertyDiscount before saving
func (m *MultiPropertyDiscount) BeforeSave(tx *gorm.DB) error {
	// 1. Validate OwnerID
	if m.OwnerID == uuid.Nil {
		return errors.New("owner ID is required")
	}

	// 2. Validate ListingIDs
	if len(m.ListingIDs) < 2 {
		return errors.New("at least 2 listings required for multi-property discount")
	}

	// 3. Validate MinProperties
	if m.MinProperties < 2 {
		return errors.New("minimum listings must be at least 2")
	}

	if m.MinProperties > len(m.ListingIDs) {
		return errors.New("minimum listings cannot exceed total listings in discount")
	}

	// 4. Validate DiscountPercent
	if m.DiscountPercent <= 0 || m.DiscountPercent > 50 {
		return errors.New("discount percent must be between 1 and 50")
	}

	// 5. Validate Date Range
	if m.ValidFrom != nil && m.ValidUntil != nil {
		if !m.ValidFrom.Before(*m.ValidUntil) {
			return ErrInvalidDateRange
		}
	}

	return nil
}
