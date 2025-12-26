package schema

// PaymentStatus represents the database enum for payment status
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// PaymentMethodType represents the database enum for payment method type
type PaymentMethodType string

const (
	PaymentMethodCard PaymentMethodType = "card"
	PaymentMethodBank PaymentMethodType = "bank"
)

// TransactionType represents the database enum for transaction type
type TransactionType string

const (
	TransactionTypePayment TransactionType = "payment"
	TransactionTypeRefund  TransactionType = "refund"
	TransactionTypePayout  TransactionType = "payout"
)

// TransactionStatus represents the database enum for transaction status
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusSucceeded TransactionStatus = "succeeded"
	TransactionStatusFailed    TransactionStatus = "failed"
)

// Market represents supported markets
type Market string

const (
	MarketGhana       Market = "GH"
	MarketKenya       Market = "KE"
	MarketNigeria     Market = "NG"
	MarketSouthAfrica Market = "ZA"
	MarketOther       Market = "OTHER"
)

// ResourceType represents the type of resource linked to a payment
type ResourceType string

const (
	ResourceTypeGeneral        ResourceType = "general"
	ResourceTypeBooking        ResourceType = "booking"
	ResourceTypeIDVerification ResourceType = "id_verification"
	ResourceTypeSubscription   ResourceType = "subscription"
	ResourceTypeRentalDraft    ResourceType = "rental_draft"
)
