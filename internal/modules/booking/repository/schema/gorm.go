package schema

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Booking represents the GORM model for bookings.
type Booking struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ListingID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	CalendarEventID uuid.UUID  `gorm:"type:uuid;not null;index"`
	CleaningEventID *uuid.UUID `gorm:"type:uuid"`

	GuestID    uuid.UUID `gorm:"type:uuid;not null;index"`
	GuestName  string    `gorm:"type:varchar(255);not null"`
	GuestEmail string    `gorm:"type:varchar(255);not null"`
	GuestPhone *string   `gorm:"type:varchar(50)"`

	GuestCount int `gorm:"not null"`

	Status      BookingStatus `gorm:"type:varchar(32);not null;default:'awaiting_payment';index"`
	BookingType BookingType   `gorm:"type:varchar(32);not null;default:'request'"`

	// Actual check-in/out timestamps (populated by host or system)
	CheckIn  *time.Time `gorm:"index"`
	CheckOut *time.Time `gorm:"index"`

	// Scheduled check-in/out timestamps (derived from listing rules)
	CheckInTime  *time.Time `gorm:"index"`
	CheckOutTime *time.Time `gorm:"index"`

	HoldExpiresAt *time.Time `gorm:"index"`
	PaymentDueAt  *time.Time `gorm:"index"`
	ActiveAt      *time.Time
	CompletedAt   *time.Time
	ArchivedAt    *time.Time
	ReviewInviteSentAt *time.Time `gorm:"index"`

	// Payment tracking
	PaymentReference *string    `gorm:"type:varchar(255);index"`
	LastPaymentID    *uuid.UUID `gorm:"type:uuid;index"`

	// Refund tracking
	RefundAmount      int64      `gorm:"default:0"`
	RefundInitiatedAt *time.Time `gorm:"index"`
	RefundProcessedAt *time.Time
	RefundReason      *string `gorm:"type:text"`
	RefundReference   *string `gorm:"type:varchar(255);index"`
	CancelledBy       *string `gorm:"type:varchar(20)"` // guest, host, admin

	SpecialRequests *string `gorm:"type:text"`

	PriceBreakdown *PriceBreakdownSnapshot `gorm:"type:jsonb;serializer:json"`
	TotalPrice     float64                 `gorm:"not null"`
	Currency       string                  `gorm:"type:varchar(10);not null"`

	ConfirmedAt *time.Time
	CancelledAt *time.Time

	// Review tracking (populated by review module via hooks)
	GuestReviewedAt *time.Time `gorm:"index"` // When guest reviewed the listing/host
	HostReviewedAt  *time.Time `gorm:"index"` // When host reviewed the guest

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// PriceBreakdownSnapshot is stored as JSON.
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

// DiscountSnapshot records applied discounts.
type DiscountSnapshot struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"`
}

// DailyRate contains nightly rate meta.
type DailyRate struct {
	Date      string  `json:"date"`
	BaseRate  float64 `json:"base_rate"`
	FinalRate float64 `json:"final_rate"`
}

// PlatformFeeBreakdown captures Hauslet fee metrics.
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

// BeforeSave validates the booking model.
func (b *Booking) BeforeSave(tx *gorm.DB) error {
	if b.ListingID == uuid.Nil {
		return errors.New("listing_id is required")
	}
	if b.CalendarEventID == uuid.Nil {
		return errors.New("calendar_event_id is required")
	}
	if b.GuestID == uuid.Nil {
		return errors.New("guest_id is required")
	}
	hasScheduled := b.CheckInTime != nil && b.CheckOutTime != nil
	hasActual := b.CheckIn != nil && b.CheckOut != nil

	switch {
	case hasScheduled:
		if !b.CheckOutTime.After(*b.CheckInTime) {
			return errors.New("scheduled checkout must be after scheduled checkin")
		}
	case hasActual:
		if !b.CheckOut.After(*b.CheckIn) {
			return errors.New("checkout must be after checkin")
		}
	default:
		return errors.New("check-in/check-out timestamps are required")
	}
	if b.GuestCount <= 0 {
		return errors.New("guest_count must be positive")
	}
	switch b.BookingType {
	case BookingInstant, BookingRequest:
	default:
		return errors.New("invalid booking_type value")
	}

	switch b.Status {
	case BookingStatusDraft,
		BookingStatusPendingApproval,
		BookingStatusAwaitingPayment,
		BookingStatusPaymentFailed,
		BookingStatusConfirmed,
		BookingStatusActive,
		BookingStatusCompleted,
		BookingStatusCancelled,
		BookingStatusArchived,
		BookingStatusDisputed,
		BookingStatusSettled:
		// valid
	default:
		return errors.New("invalid booking status")
	}

	return nil
}

// Value implements driver.Valuer for PriceBreakdownSnapshot.
func (p PriceBreakdownSnapshot) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner for PriceBreakdownSnapshot.
func (p *PriceBreakdownSnapshot) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal PriceBreakdownSnapshot value: %v", value)
	}
	return json.Unmarshal(bytes, p)
}
