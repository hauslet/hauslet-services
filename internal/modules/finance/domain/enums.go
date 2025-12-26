package domain

// WalletType defines the type of wallet
type WalletType string

const (
	WalletTypeEscrow        WalletType = "escrow"          // Holds guest funds until release
	WalletTypeHostAvailable WalletType = "host_available"  // Host's available balance for withdrawal
	WalletTypePlatformFee   WalletType = "platform_fee"    // Platform commission wallet
	WalletTypeRefundPool    WalletType = "refund_pool"     // Pool for refunds
)

// WalletStatus defines the status of a wallet
type WalletStatus string

const (
	WalletStatusActive WalletStatus = "active" // Normal operations allowed
	WalletStatusFrozen WalletStatus = "frozen" // Locked during disputes
	WalletStatusClosed WalletStatus = "closed" // Permanently closed
)

// TransactionType defines the type of financial transaction
type TransactionType string

const (
	TransactionTypeCharge     TransactionType = "charge"     // Guest payment received
	TransactionTypeRefund     TransactionType = "refund"     // Refund to guest
	TransactionTypePayout     TransactionType = "payout"     // Payout to host
	TransactionTypeCommission TransactionType = "commission" // Platform fee deduction
	TransactionTypeReversal   TransactionType = "reversal"   // Dispute reversal
)

// TransactionStatus defines the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"   // Awaiting processing
	TransactionStatusCompleted TransactionStatus = "completed" // Successfully completed
	TransactionStatusFailed    TransactionStatus = "failed"    // Failed to process
	TransactionStatusReversed  TransactionStatus = "reversed"  // Reversed due to dispute
)

// DisbursementStatus defines the status of a payout disbursement
type DisbursementStatus string

const (
	DisbursementStatusPending    DisbursementStatus = "pending"    // Queued for processing
	DisbursementStatusProcessing DisbursementStatus = "processing" // Transfer in progress
	DisbursementStatusCompleted  DisbursementStatus = "completed"  // Successfully transferred
	DisbursementStatusFailed     DisbursementStatus = "failed"     // Transfer failed
	DisbursementStatusCancelled  DisbursementStatus = "cancelled"  // Cancelled by admin/system
)

// ResourceType defines what resource a transaction is associated with
type ResourceType string

const (
	ResourceTypeBooking      ResourceType = "booking"      // Booking payment
	ResourceTypeSubscription ResourceType = "subscription" // Subscription payment
	ResourceTypeVerification ResourceType = "verification" // ID verification fee
	ResourceTypeOther        ResourceType = "other"        // Other transactions
)

// OwnerType defines who owns a wallet
type OwnerType string

const (
	OwnerTypeUser     OwnerType = "user"     // Individual user
	OwnerTypeBusiness OwnerType = "business" // Business account
	OwnerTypePlatform OwnerType = "platform" // Platform-owned wallet
)

// String methods for enums
func (w WalletType) String() string {
	return string(w)
}

func (w WalletStatus) String() string {
	return string(w)
}

func (t TransactionType) String() string {
	return string(t)
}

func (t TransactionStatus) String() string {
	return string(t)
}

func (d DisbursementStatus) String() string {
	return string(d)
}

func (r ResourceType) String() string {
	return string(r)
}

func (o OwnerType) String() string {
	return string(o)
}
