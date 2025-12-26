package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"
	"time"

	"github.com/google/uuid"
)

// WalletRepository handles wallet data persistence
type WalletRepository interface {
	Create(ctx context.Context, wallet *schema.Wallet) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Wallet, error)
	GetByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID, walletType string) (*schema.Wallet, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, newBalance int64) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	ListByOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]*schema.Wallet, error)
}

// LedgerRepository handles ledger entry data persistence
type LedgerRepository interface {
	CreateEntry(ctx context.Context, entry *schema.LedgerEntry) error
	CreateEntries(ctx context.Context, entries []*schema.LedgerEntry) error
	GetByReference(ctx context.Context, reference string) (*schema.LedgerEntry, error)
	ListByTransaction(ctx context.Context, txID uuid.UUID) ([]*schema.LedgerEntry, error)
	ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*schema.LedgerEntry, error)
	ListByWallet(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*schema.LedgerEntry, error)
}

// TransactionRepository handles transaction data persistence
type TransactionRepository interface {
	Create(ctx context.Context, tx *schema.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*schema.Transaction, error)
	GetByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) (*schema.Transaction, error)
	ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*schema.Transaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMsg *string) error
	ListByStatus(ctx context.Context, status string, limit int) ([]*schema.Transaction, error)
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
}
