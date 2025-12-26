package domain

import (
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// PayoutDetail represents bank account details for receiving payouts
type PayoutDetail struct {
	ID uuid.UUID

	// Owner (host/business)
	UserID     *uuid.UUID
	BusinessID *uuid.UUID

	// Bank account details
	BankCode       string
	BankName       string
	AccountNumber  string
	AccountName    string
	Currency       payment.Currency
	Market         Market

	// Provider details
	Provider      string  // "paystack"
	RecipientCode string  // Provider's recipient identifier
	IsVerified    bool    // Whether account was validated
	VerifiedAt    *time.Time

	// Status
	IsDefault bool
	IsActive  bool

	// Metadata
	Metadata map[string]string

	// Audit fields
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// CanReceivePayouts checks if detail can receive payouts
func (pd *PayoutDetail) CanReceivePayouts() bool {
	if !pd.IsActive {
		return false
	}
	if !pd.IsVerified {
		return false
	}
	return pd.RecipientCode != ""
}

// GetDisplayName returns a display-friendly name
func (pd *PayoutDetail) GetDisplayName() string {
	masked := MaskAccountNumber(pd.AccountNumber)
	return pd.BankName + " - " + masked
}

// MaskAccountNumber masks account number for display
func MaskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return accountNumber
	}
	last4 := accountNumber[len(accountNumber)-4:]
	return "****" + last4
}

// BelongsToUser checks if payout detail belongs to user
func (pd *PayoutDetail) BelongsToUser(userID uuid.UUID) bool {
	return pd.UserID != nil && *pd.UserID == userID
}

// BelongsToBusiness checks if payout detail belongs to business
func (pd *PayoutDetail) BelongsToBusiness(businessID uuid.UUID) bool {
	return pd.BusinessID != nil && *pd.BusinessID == businessID
}
