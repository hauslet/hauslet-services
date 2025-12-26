package domain

import (
	"time"

	"github.com/google/uuid"
)

// CancellationActor represents who initiated the cancellation
type CancellationActor string

const (
	CancelledByGuest CancellationActor = "guest"
	CancelledByHost  CancellationActor = "host"
	CancelledByAdmin CancellationActor = "admin"
)

// RefundCalculationInput contains all inputs needed to calculate a refund
type RefundCalculationInput struct {
	// Booking details
	BookingID        uuid.UUID `json:"booking_id"`
	TotalPaid        float64   `json:"total_paid"`          // Total amount paid by guest
	Currency         string    `json:"currency"`            // Currency code
	BookingCreatedAt time.Time `json:"booking_created_at"`  // When booking was created (for grace period)
	CheckInTime      time.Time `json:"check_in_time"`       // Original check-in time
	CancellationTime time.Time `json:"cancellation_time"`   // When cancellation is happening

	// Policy
	RefundPolicy string `json:"refund_policy"` // flexible/moderate/strict/long_term

	// Cancellation context
	CancelledBy CancellationActor `json:"cancelled_by"` // Who initiated cancellation
	Reason      *string           `json:"reason,omitempty"` // Optional reason
}

// RefundBreakdown provides detailed breakdown of refund calculation
type RefundBreakdown struct {
	// Original booking details
	BookingID      uuid.UUID `json:"booking_id"`
	OriginalAmount float64   `json:"original_amount"`
	Currency       string    `json:"currency"`

	// Refund calculation
	RefundPercentage   float64 `json:"refund_percentage"`    // e.g., 100.0, 50.0, 0.0
	BaseRefund         float64 `json:"base_refund"`          // Amount before fees (original * percentage)
	ProcessingFee      float64 `json:"processing_fee"`       // Non-refundable processing fee
	ProcessingFeePayer string  `json:"processing_fee_payer"` // guest/host/shared
	NetRefund          float64 `json:"net_refund"`           // Final amount to refund to guest

	// Policy context
	AppliedPolicy      string    `json:"applied_policy"`       // Which policy was used (e.g., "moderate")
	IsGracePeriod      bool      `json:"is_grace_period"`      // Was grace period applied?
	HoursUntilCheckIn  float64   `json:"hours_until_checkin"`  // Hours between cancellation and check-in
	HoursAfterBooking  float64   `json:"hours_after_booking"`  // Hours between booking creation and cancellation
	CancelledBy        string    `json:"cancelled_by"`         // Who cancelled

	// Timestamps
	BookedAt     time.Time `json:"booked_at"`
	CheckInAt    time.Time `json:"check_in_at"`
	CancelledAt  time.Time `json:"cancelled_at"`
	CalculatedAt time.Time `json:"calculated_at"`

	// Human-readable explanation
	Reason      string `json:"reason"`       // Detailed explanation of refund calculation
	Summary     string `json:"summary"`      // Short summary (e.g., "100% refund - grace period")
	PolicyRules string `json:"policy_rules"` // Policy rules that applied
}

// IsFullRefund checks if this is a 100% refund
func (r *RefundBreakdown) IsFullRefund() bool {
	return r.RefundPercentage >= 100.0
}

// IsPartialRefund checks if this is a partial refund
func (r *RefundBreakdown) IsPartialRefund() bool {
	return r.RefundPercentage > 0 && r.RefundPercentage < 100.0
}

// IsNoRefund checks if this is a no-refund scenario
func (r *RefundBreakdown) IsNoRefund() bool {
	return r.RefundPercentage <= 0
}

// GetHostImpact calculates the financial impact on the host
// Returns the amount host keeps from the original payment
func (r *RefundBreakdown) GetHostImpact() float64 {
	return r.OriginalAmount - r.NetRefund
}

// GetPlatformFeeImpact calculates platform fee refund impact
// If processing fee is paid by guest, platform keeps it
func (r *RefundBreakdown) GetPlatformFeeImpact() float64 {
	switch r.ProcessingFeePayer {
	case "guest":
		return r.ProcessingFee
	case "host":
		return 0
	case "shared":
		return r.ProcessingFee / 2
	default:
		return r.ProcessingFee
	}
}

// RefundError represents refund calculation errors
type RefundError struct {
	Code    string
	Message string
}

func (e *RefundError) Error() string {
	return e.Message
}

// Refund calculation error codes
var (
	ErrNoPaymentFound       = &RefundError{"NO_PAYMENT", "no payment found for booking"}
	ErrAlreadyRefunded      = &RefundError{"ALREADY_REFUNDED", "booking has already been refunded"}
	ErrRefundExceedsPayment = &RefundError{"EXCEEDS_PAYMENT", "refund amount exceeds paid amount"}
	ErrInvalidPolicy        = &RefundError{"INVALID_POLICY", "invalid refund policy"}
	ErrPolicyNotFound       = &RefundError{"POLICY_NOT_FOUND", "refund policy not found in configuration"}
	ErrCheckedIn            = &RefundError{"CHECKED_IN", "cannot refund after check-in"}
	ErrInvalidDates         = &RefundError{"INVALID_DATES", "invalid booking or cancellation dates"}
)
