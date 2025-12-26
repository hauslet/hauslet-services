package domain

import (
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// Transaction represents a ledger entry for all payment operations
type Transaction struct {
	ID uuid.UUID

	// Related entities
	PaymentID  *uuid.UUID // Related payment (if applicable)
	BookingID  *uuid.UUID // Related booking
	BusinessID *uuid.UUID // Related business

	// Transaction details
	Type      TransactionType
	Reference string               // Unique transaction reference
	Amount    int64                // Amount in minor units
	Currency  payment.Currency     // Transaction currency
	Status    TransactionStatus    // Current status
	Provider  string               // Payment provider used
	ProviderTxID *string           // Provider's transaction ID

	// Context
	Description string
	Metadata    map[string]string

	// Error tracking
	ErrorMessage *string
	ErrorCode    *string

	// Audit fields
	ProcessedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsSucceeded checks if transaction succeeded
func (t *Transaction) IsSucceeded() bool {
	return t.Status == TransactionStatusSucceeded
}

// IsFailed checks if transaction failed
func (t *Transaction) IsFailed() bool {
	return t.Status == TransactionStatusFailed
}

// IsPending checks if transaction is pending
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending
}

// SetError sets error details on transaction
func (t *Transaction) SetError(code, message string) {
	t.ErrorCode = &code
	t.ErrorMessage = &message
	t.Status = TransactionStatusFailed
}

// MarkSucceeded marks transaction as succeeded
func (t *Transaction) MarkSucceeded() {
	t.Status = TransactionStatusSucceeded
	now := time.Now()
	t.ProcessedAt = &now
}

// TransactionFilter for querying transactions
type TransactionFilter struct {
	PaymentID  *uuid.UUID
	BookingID  *uuid.UUID
	BusinessID *uuid.UUID
	Type       *TransactionType
	Status     *TransactionStatus
	StartDate  *time.Time
	EndDate    *time.Time
}
