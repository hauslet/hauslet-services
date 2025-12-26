package payment

import (
	"fmt"
	"strings"
)

// ToMinorUnits converts a major unit amount to minor units (e.g., NGN 1000.50 → 100050 kobo)
// Different currencies have different minor unit divisions:
// - NGN: 1 Naira = 100 kobo
// - USD: 1 Dollar = 100 cents
// - GHS: 1 Cedi = 100 pesewas
func ToMinorUnits(amount float64, currency Currency) int64 {
	switch currency {
	case NGN, USD, GHS:
		return int64(amount * 100)
	default:
		return int64(amount * 100) // Default to 2 decimal places
	}
}

// FromMinorUnits converts minor units back to major units (e.g., 100050 kobo → NGN 1000.50)
func FromMinorUnits(amount int64, currency Currency) float64 {
	switch currency {
	case NGN, USD, GHS:
		return float64(amount) / 100.0
	default:
		return float64(amount) / 100.0 // Default to 2 decimal places
	}
}

// ValidateReference checks if a transaction reference is valid
// Reference should be alphanumeric with hyphens/underscores, max 200 chars
func ValidateReference(ref string) error {
	if ref == "" {
		return fmt.Errorf("reference cannot be empty")
	}
	if len(ref) > 200 {
		return fmt.Errorf("reference exceeds maximum length of 200 characters")
	}
	// Basic validation - adjust as needed
	if strings.ContainsAny(ref, " \t\n\r") {
		return fmt.Errorf("reference cannot contain whitespace")
	}
	return nil
}

// NormalizeStatus converts provider-specific statuses to unified TransactionStatus
func NormalizeStatus(providerStatus string, provider string) TransactionStatus {
	status := strings.ToLower(strings.TrimSpace(providerStatus))

	switch provider {
	case "paystack":
		switch status {
		case "success":
			return StatusSuccess
		case "failed":
			return StatusFailed
		case "pending", "ongoing", "send_otp", "otp":
			return StatusPending
		default:
			return StatusFailed
		}

	case "stripe":
		switch status {
		case "succeeded":
			return StatusSuccess
		case "failed", "canceled":
			return StatusFailed
		case "pending", "processing", "requires_action", "requires_payment_method":
			return StatusPending
		default:
			return StatusFailed
		}

	default:
		return StatusFailed
	}
}

// NormalizeTransferStatus converts provider-specific transfer statuses to unified TransferStatus
func NormalizeTransferStatus(providerStatus string, provider string) TransferStatus {
	status := strings.ToLower(strings.TrimSpace(providerStatus))

	switch provider {
	case "paystack":
		switch status {
		case "success":
			return TransferSuccess
		case "failed", "reversed":
			return TransferFailed
		case "pending", "queued", "otp", "processing":
			return TransferPending
		default:
			return TransferFailed
		}

	case "stripe":
		switch status {
		case "paid":
			return TransferSuccess
		case "failed", "canceled":
			return TransferFailed
		case "pending", "in_transit":
			return TransferPending
		default:
			return TransferFailed
		}

	default:
		return TransferFailed
	}
}

// FormatAmount formats amount with currency for display
// Example: FormatAmount(100050, NGN) → "NGN 1,000.50"
func FormatAmount(amount int64, currency Currency) string {
	majorUnits := FromMinorUnits(amount, currency)
	return fmt.Sprintf("%s %.2f", currency, majorUnits)
}

// GetCurrencySymbol returns the symbol for a given currency
func GetCurrencySymbol(currency Currency) string {
	switch currency {
	case NGN:
		return "₦"
	case USD:
		return "$"
	case GHS:
		return "GH₵"
	default:
		return string(currency)
	}
}

// SanitizeMetadata removes sensitive data from metadata before logging
func SanitizeMetadata(metadata map[string]string) map[string]string {
	sanitized := make(map[string]string)
	sensitiveKeys := map[string]bool{
		"card_number": true,
		"cvv":         true,
		"pin":         true,
		"password":    true,
	}

	for key, value := range metadata {
		lowerKey := strings.ToLower(key)
		if sensitiveKeys[lowerKey] {
			sanitized[key] = "***REDACTED***"
		} else {
			sanitized[key] = value
		}
	}
	return sanitized
}
