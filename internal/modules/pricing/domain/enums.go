package domain

// RuleType represents the type of pricing rule
type RuleType string

const (
	RuleTypeBase         RuleType = "base"
	RuleTypeSeasonal     RuleType = "seasonal"
	RuleTypeWeekend      RuleType = "weekend"
	RuleTypeWeekday      RuleType = "weekday"
	RuleTypeHoliday      RuleType = "holiday"
	RuleTypeLengthOfStay RuleType = "length_of_stay"
	RuleTypeLastMinute   RuleType = "last_minute"
	RuleTypeEarlyBird    RuleType = "early_bird"
	RuleTypeGapFiller    RuleType = "gap_filler"
	RuleTypeCustom       RuleType = "custom"
)

// ModifierType represents how a rule modifies the price
type ModifierType string

const (
	ModifierPercentage ModifierType = "percentage"
	ModifierFixed      ModifierType = "fixed"
	ModifierAbsolute   ModifierType = "absolute"
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
