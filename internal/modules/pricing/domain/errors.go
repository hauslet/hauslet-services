package domain

import "errors"

var (
	// Rule errors
	ErrRuleNotFound       = errors.New("pricing rule not found")
	ErrInvalidRuleType    = errors.New("invalid rule type")
	ErrInvalidModifier    = errors.New("invalid modifier configuration")
	ErrRuleConflict       = errors.New("rule conflicts with existing rule")

	// Calculation errors
	ErrNoPricingRules     = errors.New("no pricing rules configured for listing")
	ErrInvalidDateRange   = errors.New("invalid date range")
	ErrCalculationFailed  = errors.New("price calculation failed")

	// Permission errors
	ErrUnauthorized       = errors.New("unauthorized to access pricing")
	ErrAccessDenied       = errors.New("access denied")

	// Discount errors
	ErrDiscountNotFound   = errors.New("discount not found")
	ErrInvalidDiscount    = errors.New("invalid discount configuration")
)
