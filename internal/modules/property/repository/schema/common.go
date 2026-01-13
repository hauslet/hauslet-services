package schema

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// --- SHARED ENUMS ---

// FeeFrequency defines when the fee is applied.
type FeeFrequency string

const (
	FeeFreqOneTime       FeeFrequency = "one_time"
	FeeFreqPerStay       FeeFrequency = "per_stay"
	FeeFreqPerGuest      FeeFrequency = "per_guest"
	FeeFreqPerExtraGuest FeeFrequency = "per_extra_guest"
	FeeFreqPerNight      FeeFrequency = "per_night" // Common for Shortlets
	FeeFreqPerMonth      FeeFrequency = "per_month" // Common for Rentals/Service Charge
	FeeFreqPerYear       FeeFrequency = "per_year"  // Common for Rentals/Service Charge
)

// FeeCategory helps frontend group fees (e.g., hiding legal fees in standard filters)
type FeeCategory string

const (
	FeeCatLegal   FeeCategory = "legal"
	FeeCatAgency  FeeCategory = "agency"
	FeeCatService FeeCategory = "service"
	FeeCatCaution FeeCategory = "caution" // Refundable deposits
	FeeCatOther   FeeCategory = "other"
)

// CustomFee is the generic structure for any cost associated with a listing
type CustomFee struct {
	Name      string       `json:"name"`      // e.g., "Generator Fuel", "Legal Fee"
	Amount    float64      `json:"amount"`    // The cost value
	Frequency FeeFrequency `json:"frequency"` // How often it is paid
	Category  FeeCategory  `json:"category"`  // For grouping logic

	// Optional helpful flags
	IsRefundable bool `json:"is_refundable,omitempty"` // Useful for Caution Fees
	IsOptional   bool `json:"is_optional,omitempty"`   // Useful for things like "Extra Cleaning"
}

// --- SHARED TYPES ---

type AmenityHighlight struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Icon    string `json:"icon"`
}

type RuleItem struct {
	Name        RuleSubCategory `json:"name" validate:"required"`
	Description map[string]any  `json:"description" validate:"required,min=1"`
}

type RuleGroup struct {
	Category RuleCategory `json:"category" validate:"required"`
	Rules    []RuleItem   `json:"rules" validate:"required,min=1"`
}

type AmenitiesType struct {
	Group string   `json:"group"`
	Items []string `json:"items"`
}

// ShowingAvailability defines when viewings can be scheduled
type ShowingAvailability struct {
	DayOfWeek string `json:"day_of_week"` // "monday", "tuesday", etc.
	StartTime string `json:"start_time"`  // HH:MM format (24h)
	EndTime   string `json:"end_time"`    // HH:MM format (24h)
	Timezone  string `json:"timezone"`    // IANA timezone (e.g., "Africa/Lagos")
}

// DiscountType defines the behavior of the discount
type DiscountType string

const (
	DiscountTypeFlat         DiscountType = "flat"           // Standard % off (General Promotion)
	DiscountTypeLengthOfStay DiscountType = "length_of_stay" // Triggered by duration (Weekly/Monthly)
)

type Discount struct {
	Name       string       `json:"name"`       // e.g., "Flash Sale", "Weekly Discount"
	Type       DiscountType `json:"type"`       // "flat" or "length_of_stay"
	Percentage float64      `json:"percentage"` // e.g., 10.0 for 10%

	// Logic for application
	MinNights *int `json:"min_nights,omitempty"` // Required if Type is "length_of_stay" (e.g., 7 or 28)

	Active bool `json:"active"` // Corresponds to the toggle switch in your UI
}

// --- BOOKING SETTINGS ---

type ApprovalMethod string

const (
	ApprovalMethodInstant ApprovalMethod = "instant" // Matches "Instant Book"
	ApprovalMethodRequest ApprovalMethod = "request" // Matches "Request to Book"
)

type GuestRequirements struct {
	VerifiedID           bool `json:"verified_id"`            // Matches "Verified ID"
	PositiveReviewsOnly  bool `json:"positive_reviews_only"`  // Matches "Positive Reviews Only"
	ProfilePhotoRequired bool `json:"profile_photo_required"` // Matches "Profile Photo Required"
}

type BookingSettings struct {
	ApprovalMethod    ApprovalMethod    `json:"approval_method"`
	GuestRequirements GuestRequirements `json:"guest_requirements"`
	PreBookingMessage string            `json:"pre_booking_message,omitempty"` // The text area input
}

// --- STAY LIMITS & ADVANCE BOOKING ---
type StayLimits struct {
	MinNights int  `json:"min_nights"`           // Matches "Minimum Night Stay"
	MaxNights *int `json:"max_nights,omitempty"` // Matches "Maximum Night Stay"
}

type AdvanceBooking struct {
	// Matches "How far in advance can guests book?"
	// e.g., 3 months, 6 months, 12 months. -1 could mean "Always available"
	MonthsAhead int `json:"months_ahead"`

	// Matches "Advance Notice (Preparation Time)"
	// The minimum hours required before check-in.
	// 0 = Same Day, 24 = 1 Day, 48 = 2 Days, etc.
	MinNoticeHours int `json:"min_notice_hours"`
}

// --- HELPER FUNCTIONS ---

// generateRandomString creates a random string of specified total length
func generateRandomString(prefix string, totalLength int) (string, error) {
	randomLen := totalLength - len(prefix)
	if randomLen < 1 {
		return "", fmt.Errorf("prefix is longer than total length")
	}

	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, randomLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	var result strings.Builder
	result.WriteString(prefix)

	for i := range randomLen {
		result.WriteByte(chars[int(b[i])%len(chars)])
	}

	return result.String(), nil
}
