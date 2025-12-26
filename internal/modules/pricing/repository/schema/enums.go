package schema

// RuleType represents the type of pricing rule
type RuleType string

const (
	RuleTypeBase         RuleType = "base"          // Base price (default)
	RuleTypeSeasonal     RuleType = "seasonal"      // Seasonal pricing
	RuleTypeWeekend      RuleType = "weekend"       // Weekend pricing
	RuleTypeWeekday      RuleType = "weekday"       // Weekday pricing
	RuleTypeHoliday      RuleType = "holiday"       // Holiday pricing
	RuleTypeLengthOfStay RuleType = "length_of_stay" // Discounts based on length
	RuleTypeLastMinute   RuleType = "last_minute"   // Last-minute pricing
	RuleTypeEarlyBird    RuleType = "early_bird"    // Early booking discount
	RuleTypeGapFiller    RuleType = "gap_filler"    // Fill gaps between bookings
	RuleTypeCustom       RuleType = "custom"        // Custom date-specific pricing
)

// ModifierType represents how a rule modifies the price
type ModifierType string

const (
	ModifierPercentage ModifierType = "percentage" // Percentage increase/decrease
	ModifierFixed      ModifierType = "fixed"      // Fixed amount increase/decrease
	ModifierAbsolute   ModifierType = "absolute"   // Absolute price override
)

// DayOfWeek represents days for day-specific rules
type DayOfWeek int

const (
	Sunday DayOfWeek = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)
