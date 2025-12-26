package domain

import (
	"time"

	"github.com/google/uuid"
)

// LedgerEntry represents a single entry in the double-entry ledger
// Every transaction creates at least two entries (debit and credit)
type LedgerEntry struct {
	ID             uuid.UUID
	TransactionID  uuid.UUID
	Reference      string // Idempotency key - hash of transaction parameters
	DebitWalletID  *uuid.UUID
	CreditWalletID *uuid.UUID
	Amount         int64 // Amount in minor currency units
	Currency       string
	ResourceType   ResourceType
	ResourceID     uuid.UUID
	Memo           string
	CreatedAt      time.Time
}

// IsDebit returns true if this entry debits a wallet
func (l *LedgerEntry) IsDebit() bool {
	return l.DebitWalletID != nil
}

// IsCredit returns true if this entry credits a wallet
func (l *LedgerEntry) IsCredit() bool {
	return l.CreditWalletID != nil
}

// Validate ensures the ledger entry is valid
func (l *LedgerEntry) Validate() error {
	if l.Amount <= 0 {
		return ErrInvalidAmount
	}

	if l.DebitWalletID == nil && l.CreditWalletID == nil {
		return ErrLedgerImbalance
	}

	if l.Currency == "" {
		return ErrInvalidCurrency
	}

	if l.Reference == "" {
		return ErrDuplicateTransaction // Reference is required for idempotency
	}

	return nil
}
