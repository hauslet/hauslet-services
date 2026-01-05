package domain

import (
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// PaymentMethod represents a saved payment method for a user
type PaymentMethod struct {
	ID uuid.UUID

	// Owner
	UserID uuid.UUID

	// Payment method details
	Type              PaymentMethodType
	Provider          string               // "paystack"
	AuthorizationCode string               // Provider's token for charging
	Currency          payment.Currency     // Supported currency

	// Card details (masked)
	Last4Digits  *string // Last 4 digits of card
	CardType     *string // visa, mastercard, etc.
	Brand        *string // Card brand
	ExpiryMonth  *int    // Card expiry month
	ExpiryYear   *int    // Card expiry year
	BankName     *string // Issuing bank

	// Status
	IsDefault bool
	IsActive  bool

	// Metadata
	CustomerCode *string // Provider's customer code

	// Audit fields
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// IsExpired checks if card is expired
func (pm *PaymentMethod) IsExpired() bool {
	if pm.ExpiryMonth == nil || pm.ExpiryYear == nil {
		return false // Can't determine, assume not expired
	}

	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// Card is expired if the expiry year is in the past
	if *pm.ExpiryYear < currentYear {
		return true
	}

	// Card is expired if it's the same year but the expiry month has passed
	if *pm.ExpiryYear == currentYear && *pm.ExpiryMonth < currentMonth {
		return true
	}

	return false
}

// CanCharge checks if method can be charged
func (pm *PaymentMethod) CanCharge() bool {
	if !pm.IsActive {
		return false
	}
	if pm.IsExpired() {
		return false
	}
	return pm.AuthorizationCode != ""
}

// MaskCardNumber masks card number for display
func MaskCardNumber(last4 string) string {
	return "**** **** **** " + last4
}

// GetDisplayName returns a display-friendly name for the payment method
func (pm *PaymentMethod) GetDisplayName() string {
	if pm.Last4Digits != nil {
		brand := "Card"
		if pm.Brand != nil {
			brand = *pm.Brand
		}
		return brand + " ending in " + *pm.Last4Digits
	}
	return string(pm.Type)
}
