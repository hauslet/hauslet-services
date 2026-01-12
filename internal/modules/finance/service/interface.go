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

	// ListOwnerWallets returns all wallets for a given owner
	ListOwnerWallets(ctx context.Context, ownerType domain.OwnerType, ownerID uuid.UUID) ([]*domain.Wallet, error)
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

	// ListTransactionsByOwner returns transactions for a user or business
	ListTransactionsByOwner(ctx context.Context, ownerType domain.OwnerType, ownerID uuid.UUID, txType *domain.TransactionType, status *domain.TransactionStatus, limit, offset int) ([]*domain.Transaction, error)
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

	// ListDisbursementsByOwner lists disbursements for a user or business
	ListDisbursementsByOwner(ctx context.Context, ownerType domain.OwnerType, ownerID uuid.UUID, status *domain.DisbursementStatus, limit, offset int) ([]*domain.Disbursement, error)
}

// BookingHooks defines callbacks for payout service to update booking status
type BookingHooks interface {
	// MarkAsSettled marks a booking as settled after payout completes
	MarkAsSettled(ctx context.Context, bookingID uuid.UUID) error
}

// BookingPartyQuerier provides minimal booking authorization queries for finance
type BookingPartyQuerier interface {
	// GetBookingParty determines if a user is the guest or host of a booking
	// Returns DisputePartyGuest, DisputePartyHost, or error if user not involved
	GetBookingParty(ctx context.Context, bookingID, userID uuid.UUID) (domain.DisputeParty, error)

	// GetBookingPaymentID retrieves the last payment ID for a booking
	GetBookingPaymentID(ctx context.Context, bookingID uuid.UUID) (uuid.UUID, error)
}

// DisputeService handles dispute operations
type DisputeService interface {
	// FileDispute creates a new dispute and freezes the wallet
	// The service internally determines if the user is guest or host and retrieves the payment ID
	FileDispute(ctx context.Context, bookingID, userID uuid.UUID, reason domain.DisputeReason, description string, amount int64, currency string) (*domain.Dispute, error)

	// InvestigateDispute marks a dispute as under investigation
	InvestigateDispute(ctx context.Context, disputeID, adminID uuid.UUID) error

	// ResolveDispute resolves a dispute with refund or release
	ResolveDispute(ctx context.Context, disputeID, adminID uuid.UUID, outcome domain.DisputeStatus, refundAmount int64, reason, notes string) error

	// CancelDispute cancels/withdraws a dispute
	CancelDispute(ctx context.Context, disputeID, cancelledByID uuid.UUID) error

	// AddEvidence adds evidence to a dispute (checks user authorization internally)
	AddEvidence(ctx context.Context, disputeID, userID uuid.UUID, evidenceType, url, description string) error

	// GetDispute retrieves a dispute by ID
	GetDispute(ctx context.Context, disputeID uuid.UUID) (*domain.Dispute, error)

	// GetDisputeByBooking retrieves a dispute by booking ID
	GetDisputeByBooking(ctx context.Context, bookingID uuid.UUID) (*domain.Dispute, error)

	// ListDisputes lists disputes with optional status filter
	ListDisputes(ctx context.Context, status *domain.DisputeStatus, limit, offset int) ([]*domain.Dispute, error)

	// ListUserDisputes lists disputes filed by a specific user
	ListUserDisputes(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Dispute, error)
}

// ReconciliationService handles financial reconciliation
type ReconciliationService interface {
	// RunReconciliation executes a full reconciliation check
	RunReconciliation(ctx context.Context) (*domain.ReconciliationReport, error)

	// ValidateLedgerBalance checks that all transactions have balanced entries (debit = credit)
	ValidateLedgerBalance(ctx context.Context) ([]domain.Discrepancy, error)

	// ValidateWalletBalance checks that wallet balances match sum of ledger entries
	ValidateWalletBalance(ctx context.Context) ([]domain.Discrepancy, error)

	// GetReconciliationReport retrieves a reconciliation report by ID
	GetReconciliationReport(ctx context.Context, reportID uuid.UUID) (*domain.ReconciliationReport, error)

	// ListReconciliationReports lists reconciliation reports
	ListReconciliationReports(ctx context.Context, limit, offset int) ([]*domain.ReconciliationReport, error)

	// GetLatestReconciliation retrieves the most recent reconciliation report
	GetLatestReconciliation(ctx context.Context) (*domain.ReconciliationReport, error)
}

// AdminProvider provides admin user information for notifications
type AdminProvider interface {
	// GetAdminEmails returns emails of all users with configured admin roles
	GetAdminEmails(ctx context.Context) ([]string, error)
}

// FinanceService combines wallet and ledger services
type FinanceService interface {
	WalletService
	LedgerService
	DisputeService
	ReconciliationService
}
