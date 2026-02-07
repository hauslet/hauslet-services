package payment

import (
	"encoding/json"
	"errors"
)

// Currency represents supported payment currencies
type Currency string

const (
	NGN Currency = "NGN" // Nigerian Naira (Paystack)
	USD Currency = "USD" // US Dollar (Flutterwave)
	GHS Currency = "GHS" // Ghanaian Cedi (Flutterwave)
)

// Bank represents a payout bank supported by a provider.
type Bank struct {
	Name      string
	Code      string
	Country   string
	Currency  Currency
	Type      string
	Active    bool
	IsDeleted bool
}

// IsValid checks if currency is supported
func (c Currency) IsValid() bool {
	switch c {
	case NGN, USD, GHS:
		return true
	default:
		return false
	}
}

// String returns the string representation of currency
func (c Currency) String() string {
	return string(c)
}

// TransactionStatus normalizes different provider statuses
type TransactionStatus string

const (
	StatusSuccess TransactionStatus = "success" // Payment completed successfully
	StatusFailed  TransactionStatus = "failed"  // Payment failed permanently
	StatusPending TransactionStatus = "pending" // Awaiting OTP/3DS/confirmation
)

// TransferStatus represents payout transaction status
type TransferStatus string

const (
	TransferSuccess TransferStatus = "success" // Transfer completed
	TransferFailed  TransferStatus = "failed"  // Transfer failed
	TransferPending TransferStatus = "pending" // Transfer processing
)

// PaymentRequest is the unified input for payment operations
type PaymentRequest struct {
	Amount      int64             // Amount in minor units (kobo/cents)
	Currency    Currency          // Transaction currency
	Reference   string            // Unique internal reference (e.g., "BKG-123")
	Email       string            // Customer email (required by Paystack)
	CallbackURL string            // Post-payment redirect URL
	AuthToken   *string           // Optional: Saved card token for direct charge
	Metadata    map[string]string // Additional context (BookingID, UserID, etc.)
}

type AuthorizationRequest struct {
	CustomerID string
	Email      string
	Currency   string
	Metadata   map[string]string
}

type AuthorizationResponse struct {
	AuthorizationCode string
	Reusable          bool
	Provider          string
	Raw               any
}

type ChargeAuthorizationRequest struct {
	AuthorizationCode string
	Amount            int64
	Currency          string
	Reference         string
	Metadata          map[string]string
}

// Validate checks if payment request has required fields
func (r *PaymentRequest) Validate() error {
	if r.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if !r.Currency.IsValid() {
		return errors.New("invalid currency")
	}
	if r.Reference == "" {
		return errors.New("reference is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

// PaymentResponse tells the client what to do next after payment initialization
type PaymentResponse struct {
	Success       bool              // Whether the request was processed successfully
	Status        TransactionStatus // Current transaction status
	TransactionID string            // Provider's transaction ID
	Reference     string            // Internal reference echoed back

	// Redirect flow (Paystack standard checkout)
	RedirectURL string // URL to redirect user for payment

	// Action required flow (Flutterwave redirect, Paystack OTP)
	RequiresAction bool                   // Whether additional user action is needed
	ActionPayload  map[string]interface{} // Provider-specific instruction data

	// Additional context
	Amount   int64    // Amount charged
	Currency Currency // Transaction currency
	Message  string   // Human-readable message
}

// RefundResponse contains refund operation result
type RefundResponse struct {
	Success   bool              // Whether refund was initiated successfully
	RefundID  string            // Provider's refund ID
	Status    TransactionStatus // Refund status
	Amount    int64             // Refund amount
	Currency  Currency          // Refund currency
	Message   string            // Status message
	CreatedAt string            // Refund creation timestamp
}

// PayoutRequest is the unified input for disbursement operations
type PayoutRequest struct {
	Amount        int64             // Amount in minor units
	Currency      Currency          // Payout currency
	RecipientCode string            // Provider's recipient identifier
	Reference     string            // Unique internal reference
	Narration     string            // Transfer description/reason
	Metadata      map[string]string // Additional context
}

// Validate checks if payout request has required fields
func (r *PayoutRequest) Validate() error {
	if r.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if !r.Currency.IsValid() {
		return errors.New("invalid currency")
	}
	if r.RecipientCode == "" {
		return errors.New("recipient code is required")
	}
	if r.Reference == "" {
		return errors.New("reference is required")
	}
	return nil
}

// PayoutResponse contains payout operation result
type PayoutResponse struct {
	Success    bool           // Whether transfer was initiated successfully
	TransferID string         // Provider's transfer ID
	Status     TransferStatus // Transfer status
	Amount     int64          // Transfer amount
	Currency   Currency       // Transfer currency
	Reference  string         // Internal reference
	Message    string         // Status message
	CreatedAt  string         // Transfer creation timestamp
}

// UnifiedEvent abstracts provider-specific webhook events
type UnifiedEvent struct {
	Type         string          // Event type (e.g., "payment.success", "transfer.failed")
	Reference    string          // Internal reference from metadata
	ProviderTxID string          // Provider's transaction/transfer ID
	Status       string          // Event status
	Amount       int64           // Transaction amount
	Currency     Currency        // Transaction currency
	RawData      json.RawMessage // Original provider payload for debugging
}

// RecipientValidation contains bank account validation result
type RecipientValidation struct {
	AccountNumber string // Validated account number
	AccountName   string // Account holder name
	BankCode      string // Bank identifier code
}
