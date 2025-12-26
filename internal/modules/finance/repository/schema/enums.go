package schema

// Database enum types matching domain enums

const (
	// Wallet types
	WalletTypeEscrow        = "escrow"
	WalletTypeHostAvailable = "host_available"
	WalletTypePlatformFee   = "platform_fee"
	WalletTypeRefundPool    = "refund_pool"

	// Wallet statuses
	WalletStatusActive = "active"
	WalletStatusFrozen = "frozen"
	WalletStatusClosed = "closed"

	// Transaction types
	TransactionTypeCharge     = "charge"
	TransactionTypeRefund     = "refund"
	TransactionTypePayout     = "payout"
	TransactionTypeCommission = "commission"
	TransactionTypeReversal   = "reversal"

	// Transaction statuses
	TransactionStatusPending   = "pending"
	TransactionStatusCompleted = "completed"
	TransactionStatusFailed    = "failed"
	TransactionStatusReversed  = "reversed"

	// Disbursement statuses
	DisbursementStatusPending    = "pending"
	DisbursementStatusProcessing = "processing"
	DisbursementStatusCompleted  = "completed"
	DisbursementStatusFailed     = "failed"
	DisbursementStatusCancelled  = "cancelled"

	// Resource types
	ResourceTypeBooking      = "booking"
	ResourceTypeSubscription = "subscription"
	ResourceTypeVerification = "verification"
	ResourceTypeOther        = "other"

	// Owner types
	OwnerTypeUser     = "user"
	OwnerTypeBusiness = "business"
	OwnerTypePlatform = "platform"
)
