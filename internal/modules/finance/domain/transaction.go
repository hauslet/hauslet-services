package domain

import (
	"time"

	"github.com/google/uuid"
)

// Transaction represents a high-level financial operation
// Each transaction owns one or more ledger entries
type Transaction struct {
	ID            uuid.UUID
	Type          TransactionType
	Status        TransactionStatus
	ResourceType  ResourceType
	ResourceID    uuid.UUID
	Amount        int64 // Total transaction amount in minor currency units
	Currency      string
	PaymentID     *uuid.UUID // Reference to payment module
	LedgerEntries []LedgerEntry
	ErrorMessage  *string
	Metadata      map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// MarkCompleted marks the transaction as completed
func (t *Transaction) MarkCompleted() {
	t.Status = TransactionStatusCompleted
	t.UpdatedAt = time.Now()
}

// MarkFailed marks the transaction as failed with an error message
func (t *Transaction) MarkFailed(errorMsg string) {
	t.Status = TransactionStatusFailed
	t.ErrorMessage = &errorMsg
	t.UpdatedAt = time.Now()
}

// MarkReversed marks the transaction as reversed (e.g., due to dispute)
func (t *Transaction) MarkReversed() {
	t.Status = TransactionStatusReversed
	t.UpdatedAt = time.Now()
}

// IsCompleted returns true if the transaction completed successfully
func (t *Transaction) IsCompleted() bool {
	return t.Status == TransactionStatusCompleted
}

// IsFailed returns true if the transaction failed
func (t *Transaction) IsFailed() bool {
	return t.Status == TransactionStatusFailed
}

// IsReversed returns true if the transaction was reversed
func (t *Transaction) IsReversed() bool {
	return t.Status == TransactionStatusReversed
}

// Validate ensures the transaction is valid
func (t *Transaction) Validate() error {
	if t.Amount <= 0 {
		return ErrInvalidAmount
	}

	if t.Currency == "" {
		return ErrInvalidCurrency
	}

	return nil
}
