package domain

import (
	"hauslet/internal/platform/payment"

	"github.com/google/uuid"
)

// CreatePaymentInput represents input for creating a payment
type CreatePaymentInput struct {
	// Required fields
	Amount   int64            // Amount in minor units
	Currency payment.Currency // Payment currency
	Market   Market           // Market context

	// Payer information
	PayerID    uuid.UUID
	PayerEmail string
	PayerName  string

	// Related entities
	BookingID    *uuid.UUID
	BusinessID   *uuid.UUID
	ResourceType ResourceType
	ResourceID   *uuid.UUID

	// Payment options
	PaymentMethodID *uuid.UUID // Use saved payment method
	CallbackURL     string     // Post-payment redirect

	// Context
	Description string
	Metadata    map[string]string
}

// Validate validates the create payment input
func (i *CreatePaymentInput) Validate() error {
	i.Market = NormalizeMarket(i.Market)
	if i.Amount <= 0 {
		return ErrInvalidPaymentAmount
	}
	if !i.Currency.IsValid() {
		return ErrInvalidPaymentCurrency
	}
	if i.PayerID == uuid.Nil {
		return ErrMissingRequiredField
	}
	if i.PayerEmail == "" {
		return ErrMissingRequiredField
	}
	if i.ResourceType == "" {
		i.ResourceType = ResourceTypeGeneral
	}
	if !i.ResourceType.IsValid() {
		return ErrInvalidResourceType
	}
	if i.ResourceID != nil && *i.ResourceID == uuid.Nil {
		i.ResourceID = nil
	}
	if !IsCurrencySupported(i.Market, i.Currency) {
		return ErrInvalidPaymentCurrency
	}
	return nil
}

// RefundPaymentInput represents input for refunding a payment
type RefundPaymentInput struct {
	PaymentID  uuid.UUID
	Amount     *int64 // Partial refund amount (nil for full refund)
	Reason     string // Refund reason
	RefundedBy uuid.UUID
}

// Validate validates the refund input
func (i *RefundPaymentInput) Validate() error {
	if i.PaymentID == uuid.Nil {
		return ErrMissingRequiredField
	}
	if i.Amount != nil && *i.Amount <= 0 {
		return ErrInvalidPaymentAmount
	}
	return nil
}

// CreatePaymentMethodInput represents input for saving a payment method
type CreatePaymentMethodInput struct {
	// Required fields
	UserID            uuid.UUID
	AuthorizationCode string
	Currency          payment.Currency
	Provider          string
	SetAsDefault      bool

	// Optional card metadata (extracted from webhooks for better UX)
	Last4Digits  *string
	CardType     *string
	Brand        *string
	ExpiryMonth  *int
	ExpiryYear   *int
	BankName     *string
	CustomerCode *string // Provider's customer reference
}

// Validate validates the create payment method input
func (i *CreatePaymentMethodInput) Validate() error {
	if i.UserID == uuid.Nil {
		return ErrMissingRequiredField
	}
	if i.AuthorizationCode == "" {
		return ErrMissingRequiredField
	}
	if !i.Currency.IsValid() {
		return ErrInvalidPaymentCurrency
	}
	return nil
}

// CreatePayoutDetailInput represents input for adding payout details
type CreatePayoutDetailInput struct {
	// Owner
	UserID     *uuid.UUID
	BusinessID *uuid.UUID

	// Bank details
	BankCode      string
	AccountNumber string
	AccountName   string
	Currency      payment.Currency
	Market        Market

	// Options
	SetAsDefault bool
}

// Validate validates the create payout detail input
func (i *CreatePayoutDetailInput) Validate() error {
	i.Market = NormalizeMarket(i.Market)
	if i.UserID == nil && i.BusinessID == nil {
		return ErrMissingRequiredField
	}
	if i.BankCode == "" || i.AccountNumber == "" || i.AccountName == "" {
		return ErrMissingRequiredField
	}
	if !i.Currency.IsValid() {
		return ErrInvalidPaymentCurrency
	}
	if !IsCurrencySupported(i.Market, i.Currency) {
		return ErrInvalidPaymentCurrency
	}
	return nil
}

// VerifyPaymentInput represents input for verifying a payment
type VerifyPaymentInput struct {
	Reference string
	Currency  payment.Currency
}

// ProcessPayoutInput represents input for processing a payout
type ProcessPayoutInput struct {
	Amount         int64
	Currency       payment.Currency
	PayoutDetailID uuid.UUID
	BookingID      *uuid.UUID
	BusinessID     *uuid.UUID
	Description    string
	Metadata       map[string]string
}

// Validate validates the process payout input
func (i *ProcessPayoutInput) Validate() error {
	if i.Amount <= 0 {
		return ErrInvalidPaymentAmount
	}
	if !i.Currency.IsValid() {
		return ErrInvalidPaymentCurrency
	}
	if i.PayoutDetailID == uuid.Nil {
		return ErrMissingRequiredField
	}
	return nil
}

// PaymentFilter represents filter criteria for querying payments
type PaymentFilter struct {
	PayerID      *uuid.UUID
	BookingID    *uuid.UUID
	BusinessID   *uuid.UUID
	ResourceID   *uuid.UUID
	ResourceType *ResourceType
	Status       *PaymentStatus
	Currency     *payment.Currency
	Market       *Market
}

// PaymentMethodFilter represents filter criteria for querying payment methods
type PaymentMethodFilter struct {
	UserID    uuid.UUID
	IsActive  *bool
	IsDefault *bool
	Currency  *payment.Currency
}
