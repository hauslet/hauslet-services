package domain

import (
	"fmt"
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// Payment represents a payment transaction in the system
type Payment struct {
	ID uuid.UUID

	// Reference identifiers
	Reference    string     // Unique internal reference (e.g., "PAY-BKG-12345")
	ProviderRef  string     // Provider's transaction ID
	BookingID    *uuid.UUID // Related booking (if applicable)
	BusinessID   *uuid.UUID // Related business
	ResourceType ResourceType
	ResourceID   *uuid.UUID

	// Payer information
	PayerID    uuid.UUID // User making the payment
	PayerEmail string
	PayerName  string

	// Payment details
	Amount        int64             // Amount in minor units (kobo/cents)
	Currency      payment.Currency  // Payment currency
	Market        Market            // Market context
	Status        PaymentStatus     // Current status
	PaymentMethod PaymentMethodType // How payment was made

	// Provider details
	Provider          string  // "paystack" (only supported for now)
	AuthorizationCode *string // For saved card payments
	RedirectURL       *string // Checkout URL (if applicable)
	RequiresAction    bool    // Whether user action is needed

	// Metadata
	Description string
	Metadata    map[string]string

	// Refund tracking
	RefundedAmount int64      // Total amount refunded
	RefundedAt     *time.Time // When refund was processed

	// Audit fields
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// CanRefund checks if payment can be refunded
func (p *Payment) CanRefund() bool {
	if p.Status != PaymentStatusSucceeded {
		return false
	}
	if p.RefundedAmount >= p.Amount {
		return false
	}
	return true
}

// RemainingRefundableAmount returns how much can still be refunded
func (p *Payment) RemainingRefundableAmount() int64 {
	if !p.CanRefund() {
		return 0
	}
	return p.Amount - p.RefundedAmount
}

// IsFullyRefunded checks if payment has been completely refunded
func (p *Payment) IsFullyRefunded() bool {
	return p.RefundedAmount >= p.Amount
}

// IsPending checks if payment is still pending
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// IsSucceeded checks if payment succeeded
func (p *Payment) IsSucceeded() bool {
	return p.Status == PaymentStatusSucceeded
}

// IsFailed checks if payment failed
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}

// GenerateReference creates a unique payment reference
func GenerateReference(prefix string, id uuid.UUID) string {
	return fmt.Sprintf("PAY-%s-%s", prefix, id.String()[:8])
}

// PaymentSummary provides a summary view of payment
type PaymentSummary struct {
	ID        uuid.UUID
	Reference string
	Amount    int64
	Currency  payment.Currency
	Status    PaymentStatus
	CreatedAt time.Time
}

// PaymentStats represents aggregated payment statistics for a user
type PaymentStats struct {
	TotalSpent       int64
	UpcomingPayments int64
	TotalRefunds     int64
}
