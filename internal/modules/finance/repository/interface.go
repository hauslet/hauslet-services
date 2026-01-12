package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WalletRepository handles wallet data persistence
type WalletRepository interface {
	Create(ctx context.Context, wallet *schema.Wallet) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Wallet, error)
	GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID, walletType string) (*schema.Wallet, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, newBalance int64) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	ListByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]*schema.Wallet, error)
	// FindBalanceMismatches finds wallets where balance doesn't match sum of ledger entries
	FindBalanceMismatches(ctx context.Context) ([]*schema.WalletBalanceMismatch, error)
	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) WalletRepository
}

// LedgerRepository handles ledger entry data persistence
type LedgerRepository interface {
	CreateEntry(ctx context.Context, entry *schema.LedgerEntry) error
	CreateEntries(ctx context.Context, entries []*schema.LedgerEntry) error
	GetByReference(ctx context.Context, reference string) (*schema.LedgerEntry, error)
	ListByTransaction(ctx context.Context, txID uuid.UUID) ([]*schema.LedgerEntry, error)
	ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*schema.LedgerEntry, error)
	ListByWallet(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*schema.LedgerEntry, error)
	// FindImbalancedTransactions finds internal transactions where debits don't equal credits
	// External transactions (charges with only credits, refunds with only debits) are excluded
	FindImbalancedTransactions(ctx context.Context) ([]*schema.TransactionBalance, error)
	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) LedgerRepository
}

// TransactionRepository handles transaction data persistence
type TransactionRepository interface {
	Create(ctx context.Context, tx *schema.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Transaction, error)
	GetByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) (*schema.Transaction, error)
	ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*schema.Transaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error
	ListByStatus(ctx context.Context, status string, limit int) ([]*schema.Transaction, error)
	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) TransactionRepository
}

// DisbursementRepository handles disbursement data persistence
type DisbursementRepository interface {
	Create(ctx context.Context, disbursement *schema.Disbursement) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Disbursement, error)
	GetByTransferCode(ctx context.Context, code string) (*schema.Disbursement, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, response *string) error
	IncrementAttempts(ctx context.Context, id uuid.UUID, nextRetry *time.Time) error
	ListPendingRetries(ctx context.Context) ([]*schema.Disbursement, error)
	ListByWallet(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*schema.Disbursement, error)
	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) DisbursementRepository
}

// DisputeRepository handles dispute data persistence
type DisputeRepository interface {
	Create(ctx context.Context, dispute *schema.Dispute) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Dispute, error)
	GetByBookingID(ctx context.Context, bookingID uuid.UUID) (*schema.Dispute, error)
	Update(ctx context.Context, dispute *schema.Dispute) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	ListByStatus(ctx context.Context, status string, limit, offset int) ([]*schema.Dispute, error)
	ListByFiledBy(ctx context.Context, filedByID uuid.UUID, limit, offset int) ([]*schema.Dispute, error)
	ListAll(ctx context.Context, limit, offset int) ([]*schema.Dispute, error)
	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) DisputeRepository
}

// ReconciliationRepository handles reconciliation report and discrepancy persistence
type ReconciliationRepository interface {
	// Report operations
	CreateReport(ctx context.Context, report *schema.ReconciliationReport) error
	GetReportByID(ctx context.Context, id uuid.UUID) (*schema.ReconciliationReport, error)
	UpdateReport(ctx context.Context, report *schema.ReconciliationReport) error
	ListReports(ctx context.Context, limit, offset int) ([]*schema.ReconciliationReport, error)
	GetLatestReport(ctx context.Context) (*schema.ReconciliationReport, error)
	GetRunningReport(ctx context.Context) (*schema.ReconciliationReport, error)

	// Discrepancy operations
	CreateDiscrepancy(ctx context.Context, discrepancy *schema.Discrepancy) error
	ListDiscrepanciesByReport(ctx context.Context, reportID uuid.UUID) ([]*schema.Discrepancy, error)
	ListDiscrepanciesBySeverity(ctx context.Context, reportID uuid.UUID, severity string) ([]*schema.Discrepancy, error)

	// WithTx returns a new repository instance using the provided transaction
	WithTx(tx *gorm.DB) ReconciliationRepository
}
