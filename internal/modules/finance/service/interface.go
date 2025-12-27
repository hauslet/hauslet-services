package service

import (
	"context"
	"hauslet/internal/modules/finance/domain"

	"github.com/google/uuid"
)

// WalletService handles wallet operations
type WalletService interface {
	// GetOrCreateWallet retrieves or creates a wallet for the given owner
	// Idempotent operation - safe to call multiple times
	GetOrCreateWallet(ctx context.Context, ownerType domain.OwnerType, ownerID uuid.UUID, walletType domain.WalletType, currency string) (*domain.Wallet, error)

	// GetWallet retrieves a wallet by ID
	GetWallet(ctx context.Context, walletID uuid.UUID) (*domain.Wallet, error)

	// GetBalance returns the current balance of a wallet
	GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error)

	// FreezeWallet locks a wallet (e.g., during disputes)
	FreezeWallet(ctx context.Context, walletID uuid.UUID, reason string) error

	// UnfreezeWallet unlocks a previously frozen wallet
	UnfreezeWallet(ctx context.Context, walletID uuid.UUID) error

	// ListUserWallets returns all wallets for a given user
	ListUserWallets(ctx context.Context, userID uuid.UUID) ([]*domain.Wallet, error)
}

// LedgerService handles financial transactions using double-entry bookkeeping
type LedgerService interface {
	// RecordCharge records a payment charge from guest
	// Creates ledger entries: Debit external → Credit booking escrow
	RecordCharge(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) (*domain.Transaction, error)

	// RecordRefund records a refund to guest
	// Creates ledger entries: Debit escrow → Credit external (refund pool)
	RecordRefund(ctx context.Context, bookingID, paymentID uuid.UUID, amount int64, currency string) (*domain.Transaction, error)

	// RecordCommission records platform commission
	// Creates ledger entries: Debit escrow → Credit platform fee wallet
	RecordCommission(ctx context.Context, bookingID uuid.UUID, amount int64, currency string) (*domain.Transaction, error)

	// RecordPayout records payout to host
	// Creates ledger entries: Debit escrow → Credit host available wallet
	RecordPayout(ctx context.Context, bookingID, hostID uuid.UUID, amount int64, currency string) (*domain.Transaction, error)

	// GetTransactionHistory returns all transactions for a resource
	GetTransactionHistory(ctx context.Context, resourceType domain.ResourceType, resourceID uuid.UUID) ([]*domain.Transaction, error)

	// GetWalletHistory returns ledger entries for a wallet
	GetWalletHistory(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*domain.LedgerEntry, error)
}

// PayoutService defines operations for automated host payouts
type PayoutService interface {
	// QueuePayout creates a pending payout for a completed booking
	QueuePayout(ctx context.Context, bookingID, hostID uuid.UUID, totalAmount int64, currency string) error

	// ProcessDuePayouts processes all pending payouts (called by cron job)
	ProcessDuePayouts(ctx context.Context) error

	// RetryFailedDisbursements retries failed disbursements (called by cron job)
	RetryFailedDisbursements(ctx context.Context) error

	// GetDisbursement retrieves a disbursement by ID
	GetDisbursement(ctx context.Context, disbursementID uuid.UUID) (*domain.Disbursement, error)

	// GetDisbursementByTransferCode retrieves a disbursement by transfer code
	GetDisbursementByTransferCode(ctx context.Context, code string) (*domain.Disbursement, error)

	// UpdateDisbursementStatus updates disbursement status (called by webhooks)
	UpdateDisbursementStatus(ctx context.Context, disbursementID uuid.UUID, status domain.DisbursementStatus, response *string) error
}

// BookingHooks defines callbacks for payout service to update booking status
type BookingHooks interface {
	// MarkAsSettled marks a booking as settled after payout completes
	MarkAsSettled(ctx context.Context, bookingID uuid.UUID) error
}

// FinanceService combines wallet and ledger services
type FinanceService interface {
	WalletService
	LedgerService
}
