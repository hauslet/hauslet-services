package domain

import (
	"time"

	"github.com/google/uuid"
)

// Booking represents the full booking aggregate in the domain layer.
type Booking struct {
	ID              uuid.UUID  `json:"id"`
	ListingID       uuid.UUID  `json:"listing_id"`
	CalendarEventID uuid.UUID  `json:"calendar_event_id"`
	CleaningEventID *uuid.UUID `json:"cleaning_event_id,omitempty"`

	GuestID    uuid.UUID `json:"guest_id"`
	GuestName  string    `json:"guest_name"`
	GuestEmail string    `json:"guest_email"`
	GuestPhone *string   `json:"guest_phone,omitempty"`

	GuestCount int `json:"guest_count"`

	Status      BookingStatus `json:"status"`
	BookingType BookingType   `json:"booking_type"`

	CheckIn  time.Time `json:"check_in"`
	CheckOut time.Time `json:"check_out"`

	CheckInTime  *string `json:"check_in_time,omitempty"`
	CheckOutTime *string `json:"check_out_time,omitempty"`

	HoldExpiresAt *time.Time `json:"hold_expires_at,omitempty"`
	PaymentDueAt  *time.Time `json:"payment_due_at,omitempty"`
	ActiveAt      *time.Time `json:"active_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ArchivedAt    *time.Time `json:"archived_at,omitempty"`

	// Payment tracking
	PaymentReference *string    `json:"payment_reference,omitempty"`
	LastPaymentID    *uuid.UUID `json:"last_payment_id,omitempty"`

	// Refund tracking
	RefundAmount      int64      `json:"refund_amount"`       // Amount refunded in minor units
	RefundInitiatedAt *time.Time `json:"refund_initiated_at,omitempty"`
	RefundProcessedAt *time.Time `json:"refund_processed_at,omitempty"`
	RefundReason      *string    `json:"refund_reason,omitempty"`
	RefundReference   *string    `json:"refund_reference,omitempty"`
	CancelledBy       *string    `json:"cancelled_by,omitempty"` // guest, host, admin

	SpecialRequests *string `json:"special_requests,omitempty"`

	PriceBreakdown *PriceBreakdownSnapshot `json:"price_breakdown,omitempty"`
	TotalPrice     float64                 `json:"total_price"`
	Currency       string                  `json:"currency"`

	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// BookingStatus represents the lifecycle state of a booking.
type BookingStatus string

const (
	BookingStatusDraft           BookingStatus = "draft"
	BookingStatusPendingApproval BookingStatus = "pending_host_approval"
	BookingStatusAwaitingPayment BookingStatus = "awaiting_payment"
	BookingStatusPaymentFailed   BookingStatus = "payment_failed"
	BookingStatusConfirmed       BookingStatus = "confirmed"
	BookingStatusActive          BookingStatus = "active"
	BookingStatusCompleted       BookingStatus = "completed"
	BookingStatusCancelled       BookingStatus = "cancelled"
	BookingStatusArchived        BookingStatus = "archived"
	BookingStatusDisputed        BookingStatus = "disputed"
	BookingStatusSettled         BookingStatus = "settled"
)

// BookingType describes whether a booking should auto-confirm or require approval.
type BookingType string

const (
	BookingInstant BookingType = "instant"
	BookingRequest BookingType = "request"
)

// PriceBreakdownSnapshot stores calculated pricing numbers at booking time.
type PriceBreakdownSnapshot struct {
	BaseTotal     float64               `json:"base_total"`
	CleaningFee   *float64              `json:"cleaning_fee,omitempty"`
	ServiceFee    *float64              `json:"service_fee,omitempty"`
	CautionFee    *float64              `json:"caution_fee,omitempty"`
	ExtraGuestFee float64               `json:"extra_guest_fee"`
	Discounts     []DiscountSnapshot    `json:"discounts,omitempty"`
	NightlyRates  []DailyRate           `json:"nightly_rates,omitempty"`
	Subtotal      float64               `json:"subtotal"`
	Total         float64               `json:"total"`
	Currency      string                `json:"currency"`
	PlatformFees  *PlatformFeeBreakdown `json:"platform_fees,omitempty"`
}

// DiscountSnapshot captures a discount that was applied during pricing.
type DiscountSnapshot struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"` // "percentage" or "fixed"
}

// DailyRate contains the nightly rate for a given date.
type DailyRate struct {
	Date      string  `json:"date"`
	BaseRate  float64 `json:"base_rate"`
	FinalRate float64 `json:"final_rate"`
}

// PlatformFeeBreakdown mirrors the pricing domain values for persistence.
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

// DurationNights returns the total nights for the booking.
func (b *Booking) DurationNights() int {
	return int(b.CheckOut.Sub(b.CheckIn).Hours() / 24)
}

// IsDraft returns true if the booking is still in draft/hold mode.
func (b *Booking) IsDraft() bool {
	return b.Status == BookingStatusDraft
}

// IsAwaitingPayment returns true if the booking is awaiting payment capture.
func (b *Booking) IsAwaitingPayment() bool {
	return b.Status == BookingStatusAwaitingPayment
}

// IsConfirmed returns true if the booking has been confirmed or progressed beyond.
func (b *Booking) IsConfirmed() bool {
	switch b.Status {
	case BookingStatusConfirmed, BookingStatusActive, BookingStatusCompleted, BookingStatusSettled:
		return true
	default:
		return false
	}
}

// IsCancelled returns true if the booking has been cancelled or archived.
func (b *Booking) IsCancelled() bool {
	return b.Status == BookingStatusCancelled || b.Status == BookingStatusArchived
}

// IsDisputed returns true if the booking is in a dispute workflow.
func (b *Booking) IsDisputed() bool {
	return b.Status == BookingStatusDisputed
}

// CanBeConfirmed determines if the booking can transition to confirmed.
func (b *Booking) CanBeConfirmed() bool {
	if b.IsCancelled() || b.IsDisputed() || time.Now().After(b.CheckOut) {
		return false
	}
	return b.Status == BookingStatusAwaitingPayment || b.Status == BookingStatusDraft
}

// CanBeCancelled determines if the booking can be cancelled.
func (b *Booking) CanBeCancelled() bool {
	switch b.Status {
	case BookingStatusCancelled, BookingStatusArchived, BookingStatusCompleted, BookingStatusSettled:
		return false
	}
	if time.Now().After(b.CheckOut) {
		return false
	}
	return true
}

// CanBePaid determines if the booking can have payment processed.
func (b *Booking) CanBePaid() bool {
	return b.Status == BookingStatusAwaitingPayment || b.Status == BookingStatusPaymentFailed
}

// MarkAwaitingPayment updates the booking to awaiting payment.
func (b *Booking) MarkAwaitingPayment(due time.Time) {
	b.Status = BookingStatusAwaitingPayment
	b.PaymentDueAt = &due
}

// MarkConfirmed updates the booking to confirmed status.
func (b *Booking) MarkConfirmed(at time.Time) {
	b.Status = BookingStatusConfirmed
	b.ConfirmedAt = &at
	b.HoldExpiresAt = nil
}

// MarkCancelled updates the booking to cancelled status.
func (b *Booking) MarkCancelled(at time.Time) {
	b.Status = BookingStatusCancelled
	b.CancelledAt = &at
	b.HoldExpiresAt = nil
}

// MarkArchived marks an expired draft/request as archived.
func (b *Booking) MarkArchived(at time.Time) {
	b.Status = BookingStatusArchived
	b.ArchivedAt = &at
}
