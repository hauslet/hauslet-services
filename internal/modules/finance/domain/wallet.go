package domain

import (
	"time"

	"github.com/google/uuid"
)

// Wallet represents a balance bucket for financial transactions
type Wallet struct {
	ID         uuid.UUID
	OwnerType  OwnerType
	OwnerID    uuid.UUID
	WalletType WalletType
	Balance    int64 // Amount in minor currency units (e.g., cents, kobo)
	Currency   string
	Status     WalletStatus
	Metadata   map[string]interface{}
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Credit adds funds to the wallet
func (w *Wallet) Credit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if w.Status == WalletStatusClosed {
		return ErrWalletClosed
	}

	w.Balance += amount
	w.UpdatedAt = time.Now()
	return nil
}

// Debit removes funds from the wallet
func (w *Wallet) Debit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if w.Status == WalletStatusClosed {
		return ErrWalletClosed
	}

	if w.Status == WalletStatusFrozen {
		return ErrWalletFrozen
	}

	if w.Balance < amount {
		return ErrInsufficientBalance
	}

	w.Balance -= amount
	w.UpdatedAt = time.Now()
	return nil
}

// CanDebit checks if the wallet can be debited by the given amount
func (w *Wallet) CanDebit(amount int64) bool {
	return w.Status == WalletStatusActive && w.Balance >= amount && amount > 0
}

// Freeze locks the wallet (e.g., during disputes)
func (w *Wallet) Freeze() error {
	if w.Status == WalletStatusClosed {
		return ErrWalletClosed
	}

	w.Status = WalletStatusFrozen
	w.UpdatedAt = time.Now()
	return nil
}

// Unfreeze unlocks the wallet
func (w *Wallet) Unfreeze() error {
	if w.Status != WalletStatusFrozen {
		return nil // Already unfrozen
	}

	w.Status = WalletStatusActive
	w.UpdatedAt = time.Now()
	return nil
}

// Close permanently closes the wallet
func (w *Wallet) Close() error {
	if w.Balance > 0 {
		return ErrInsufficientBalance // Cannot close with balance
	}

	w.Status = WalletStatusClosed
	w.UpdatedAt = time.Now()
	return nil
}

// IsActive returns true if the wallet can perform normal operations
func (w *Wallet) IsActive() bool {
	return w.Status == WalletStatusActive
}

// IsFrozen returns true if the wallet is frozen
func (w *Wallet) IsFrozen() bool {
	return w.Status == WalletStatusFrozen
}

// IsClosed returns true if the wallet is closed
func (w *Wallet) IsClosed() bool {
	return w.Status == WalletStatusClosed
}
