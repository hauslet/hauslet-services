package domain

import (
	"hauslet/internal/platform/payment"
	"strings"
)

// PaymentStatus represents the state of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodCard PaymentMethodType = "card"
	PaymentMethodBank PaymentMethodType = "bank"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypePayment TransactionType = "payment"
	TransactionTypeRefund  TransactionType = "refund"
	TransactionTypePayout  TransactionType = "payout"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusSucceeded TransactionStatus = "succeeded"
	TransactionStatusFailed    TransactionStatus = "failed"
)

// Market represents supported markets
type Market string

const (
	MarketGhana       Market = "ghana"
	MarketKenya       Market = "kenya"
	MarketNigeria     Market = "nigeria"
	MarketSouthAfrica Market = "south_africa"
	MarketOther       Market = "other"
)

// NormalizeMarket maps known market aliases to canonical values.
func NormalizeMarket(market Market) Market {
	switch market {
	case MarketGhana, MarketKenya, MarketNigeria, MarketSouthAfrica, MarketOther:
		return market
	}

	normalized := strings.ToLower(strings.TrimSpace(string(market)))
	switch normalized {
	case "ghana", "gh":
		return MarketGhana
	case "kenya", "ke":
		return MarketKenya
	case "nigeria", "ng":
		return MarketNigeria
	case "south_africa", "south africa", "south-africa", "southafrica", "za":
		return MarketSouthAfrica
	case "other":
		return MarketOther
	default:
		return market
	}
}

// IsValid reports whether the market is recognized.
func (m Market) IsValid() bool {
	switch NormalizeMarket(m) {
	case MarketGhana, MarketKenya, MarketNigeria, MarketSouthAfrica, MarketOther:
		return true
	default:
		return false
	}
}

// ResourceType represents the target entity for a payment
type ResourceType string

const (
	ResourceTypeGeneral        ResourceType = "general"
	ResourceTypeBooking        ResourceType = "booking"
	ResourceTypeIDVerification ResourceType = "id_verification"
	ResourceTypeSubscription   ResourceType = "subscription"
	ResourceTypePromotion      ResourceType = "promotion"
	ResourceTypeRentalDraft    ResourceType = "rental_draft"
)

// IsValid checks whether the resource type is known
func (rt ResourceType) IsValid() bool {
	switch rt {
	case ResourceTypeGeneral,
		ResourceTypeBooking,
		ResourceTypeIDVerification,
		ResourceTypeSubscription,
		ResourceTypePromotion,
		ResourceTypeRentalDraft:
		return true
	default:
		return false
	}
}

// GetMarketCurrencies returns supported currencies for a market
func GetMarketCurrencies(market Market) []payment.Currency {
	market = NormalizeMarket(market)
	switch market {
	case MarketGhana:
		return []payment.Currency{payment.GHS}
	case MarketKenya:
		return []payment.Currency{"KES", payment.USD}
	case MarketNigeria:
		return []payment.Currency{payment.NGN, payment.USD}
	case MarketSouthAfrica:
		return []payment.Currency{"ZAR", payment.USD}
	case MarketOther:
		return []payment.Currency{payment.USD}
	default:
		return []payment.Currency{payment.USD}
	}
}

// GetDefaultCurrency returns the default currency for a market
func GetDefaultCurrency(market Market) payment.Currency {
	currencies := GetMarketCurrencies(market)
	if len(currencies) > 0 {
		return currencies[0]
	}
	return payment.USD
}

// IsCurrencySupported checks if currency is supported in market
func IsCurrencySupported(market Market, currency payment.Currency) bool {
	supported := GetMarketCurrencies(market)
	for _, c := range supported {
		if c == currency {
			return true
		}
	}
	return false
}
