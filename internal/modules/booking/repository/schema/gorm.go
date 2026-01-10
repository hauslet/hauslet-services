package schema

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Booking represents the GORM model for bookings.
type Booking struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BookingReference string     `gorm:"type:varchar(20);uniqueIndex;not null"` // Format: H123456-ABC
	ListingID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	CalendarEventID  uuid.UUID  `gorm:"type:uuid;not null;index"`
	CleaningEventID  *uuid.UUID `gorm:"type:uuid"`

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

	HoldExpiresAt      *time.Time `gorm:"index"`
	PaymentDueAt       *time.Time `gorm:"index"`
	ActiveAt           *time.Time
	CompletedAt        *time.Time
	ArchivedAt         *time.Time
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

// BeforeCreate generates a unique booking reference before creating the record.
func (b *Booking) BeforeCreate(tx *gorm.DB) error {
	// Only generate if not already set (allows manual override in tests)
	if b.BookingReference == "" {
		ref, err := generateBookingReference(tx)
		if err != nil {
			return fmt.Errorf("failed to generate booking reference: %w", err)
		}
		b.BookingReference = ref
	}
	return nil
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

// generateBookingReference creates a unique booking reference in format H123456-XYZ.
func generateBookingReference(tx *gorm.DB) (string, error) {
	const maxRetries = 5
	const prefix = "H"

	for i := 0; i < maxRetries; i++ {
		// 1. Generate Number Part
		numPart, err := cryptoRandInt(100000, 999999)
		if err != nil {
			return "", fmt.Errorf("failed to generate number part: %w", err)
		}

		// 2. Generate Letter Part (Using Safe Charset)
		letterPart, err := cryptoRandSafeChars(3)
		if err != nil {
			return "", fmt.Errorf("failed to generate letter part: %w", err)
		}

		reference := fmt.Sprintf("%s%d-%s", prefix, numPart, letterPart)

		// 3. Check Uniqueness
		var count int64
		if err := tx.Model(&Booking{}).Where("booking_reference = ?", reference).Count(&count).Error; err != nil {
			return "", fmt.Errorf("failed to check reference uniqueness: %w", err)
		}

		if count == 0 {
			return reference, nil
		}
	}

	return "", errors.New("failed to generate unique booking reference after max retries")
}

// cryptoRandSafeChars generates n random characters from a safe list.
// Removed: A, E, I, O, U (Vowels to prevent bad words)
// Removed: 0, 1, I, L (To prevent visual confusion)
func cryptoRandSafeChars(n int) (string, error) {
	// "Crockford's Base32" inspired, but without numbers since you handle them separately
	const letters = "BCDFGHJKMNPQRSTVWXYZ"

	result := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		result[i] = letters[num.Int64()]
	}

	return string(result), nil
}

// cryptoRandInt remains the same as your original code
func cryptoRandInt(min, max int) (int, error) {
	rangeSize := max - min + 1
	n, err := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + min, nil
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
	VATPercent    float64               `json:"vat_percent,omitempty"`
	VATAmount     float64               `json:"vat_amount,omitempty"`
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
