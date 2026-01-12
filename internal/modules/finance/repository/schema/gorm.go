package schema

import (
	"time"

	"github.com/google/uuid"
)

// Wallet represents a balance bucket in the database
type Wallet struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerType  string    `gorm:"type:varchar(50);not null;index:idx_wallet_owner"`
	OwnerID    uuid.UUID `gorm:"type:uuid;not null;index:idx_wallet_owner"`
	WalletType string    `gorm:"type:varchar(50);not null;index:idx_wallet_owner"`
	Balance    int64     `gorm:"not null;default:0"`
	Currency   string    `gorm:"type:varchar(3);not null"`
	Status     string    `gorm:"type:varchar(20);not null;default:'active'"`
	Metadata   *string   `gorm:"type:jsonb"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
	UpdatedAt  time.Time `gorm:"not null;default:now()"`
}

// TableName specifies the table name for Wallet
func (Wallet) TableName() string {
	return "wallets"
}

// LedgerEntry represents a single ledger entry in the database
type LedgerEntry struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID  uuid.UUID  `gorm:"type:uuid;not null;index"`
	Reference      string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	DebitWalletID  *uuid.UUID `gorm:"type:uuid;index"`
	CreditWalletID *uuid.UUID `gorm:"type:uuid;index"`
	Amount         int64      `gorm:"not null"`
	Currency       string     `gorm:"type:varchar(3);not null"`
	ResourceType   string     `gorm:"type:varchar(50);not null;index:idx_ledger_resource"`
	ResourceID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_ledger_resource"`
	Memo           string     `gorm:"type:text"`
	CreatedAt      time.Time  `gorm:"not null;default:now();index"`
}

// TableName specifies the table name for LedgerEntry
func (LedgerEntry) TableName() string {
	return "ledger_entries"
}

// Transaction represents a financial transaction in the database
type Transaction struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Type         string     `gorm:"type:varchar(50);not null;index"`
	Status       string     `gorm:"type:varchar(20);not null;index"`
	ResourceType string     `gorm:"type:varchar(50);not null;index:idx_tx_resource"`
	ResourceID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_tx_resource"`
	Amount       int64      `gorm:"not null"`
	Currency     string     `gorm:"type:varchar(3);not null"`
	PaymentID    *uuid.UUID `gorm:"type:uuid;index"`
	ErrorMessage *string    `gorm:"type:text"`
	Metadata     *string    `gorm:"type:jsonb"`
	CreatedAt    time.Time  `gorm:"not null;default:now();index"`
	UpdatedAt    time.Time  `gorm:"not null;default:now()"`
}

// TableName specifies the table name for Transaction
func (Transaction) TableName() string {
	return "finance_transactions"
}

// Disbursement represents a payout disbursement in the database
type Disbursement struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID         uuid.UUID  `gorm:"type:uuid;not null;index"`
	TransactionID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Amount           int64      `gorm:"not null"`
	Currency         string     `gorm:"type:varchar(3);not null"`
	Provider         string     `gorm:"type:varchar(50);not null"`
	TransferCode     *string    `gorm:"type:varchar(255);index"`
	ProviderResponse *string    `gorm:"type:text"`
	Status           string     `gorm:"type:varchar(20);not null;index"`
	Attempts         int        `gorm:"not null;default:0"`
	NextRetryAt      *time.Time `gorm:"index"`
	CompletedAt      *time.Time
	FailureReason    *string   `gorm:"type:text"`
	CreatedAt        time.Time `gorm:"not null;default:now();index"`
	UpdatedAt        time.Time `gorm:"not null;default:now()"`
}

// TableName specifies the table name for Disbursement
func (Disbursement) TableName() string {
	return "disbursements"
}

// Dispute represents a financial dispute in the database
type Dispute struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BookingID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex;index"` // Unique: one dispute per booking
	WalletID    uuid.UUID `gorm:"type:uuid;not null;index"`
	PaymentID   uuid.UUID `gorm:"type:uuid;not null;index"`  // Payment associated with booking
	FiledBy     string    `gorm:"type:varchar(20);not null"` // guest or host
	FiledByID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Reason      string    `gorm:"type:varchar(50);not null;index"`
	Status      string    `gorm:"type:varchar(30);not null;index"`
	Description string    `gorm:"type:text;not null"`
	Amount      int64     `gorm:"not null"` // Amount in dispute
	Currency    string    `gorm:"type:varchar(3);not null"`

	// Evidence and resolution (stored as JSONB)
	Evidence   *string `gorm:"type:jsonb"` // JSON array of DisputeEvidence
	AdminNotes *string `gorm:"type:text"`
	Resolution *string `gorm:"type:jsonb"` // DisputeResolution as JSON

	ResolvedByID  *uuid.UUID `gorm:"type:uuid;index"`
	ResolvedAt    *time.Time `gorm:"index"`
	RefundAmount  *int64
	TransactionID *uuid.UUID `gorm:"type:uuid;index"` // Resolution transaction

	CreatedAt time.Time `gorm:"not null;default:now();index"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

// TableName specifies the table name for Dispute
func (Dispute) TableName() string {
	return "disputes"
}

// ReconciliationReport represents a reconciliation run in the database
type ReconciliationReport struct {
	ID                       uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Status                   string     `gorm:"type:varchar(20);not null;index"`
	StartedAt                time.Time  `gorm:"not null;index"`
	CompletedAt              *time.Time `gorm:"index"`
	TotalWalletsChecked      int        `gorm:"not null;default:0"`
	TotalTransactionsChecked int        `gorm:"not null;default:0"`
	DiscrepanciesFound       int        `gorm:"not null;default:0"`
	Summary                  string     `gorm:"type:text"`
	ErrorMessage             *string    `gorm:"type:text"`
	CreatedAt                time.Time  `gorm:"not null;default:now();index"`
	UpdatedAt                time.Time  `gorm:"not null;default:now()"`
}

// TableName specifies the table name for ReconciliationReport
func (ReconciliationReport) TableName() string {
	return "reconciliation_reports"
}

// Discrepancy represents a found issue during reconciliation in the database
type Discrepancy struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReportID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	Type          string     `gorm:"type:varchar(50);not null;index"`
	Severity      string     `gorm:"type:varchar(20);not null;index"`
	WalletID      *uuid.UUID `gorm:"type:uuid;index"`
	TransactionID *uuid.UUID `gorm:"type:uuid;index"`
	Description   string     `gorm:"type:text;not null"`
	ExpectedValue *int64
	ActualValue   *int64
	Details       *string   `gorm:"type:jsonb"` // Additional context
	CreatedAt     time.Time `gorm:"not null;default:now();index"`
}

// TableName specifies the table name for Discrepancy
func (Discrepancy) TableName() string {
	return "discrepancies"
}

// TransactionBalance represents the result of ledger balance validation query
// Used to find transactions where total debits don't equal total credits
type TransactionBalance struct {
	TransactionID uuid.UUID
	TotalDebit    int64
	TotalCredit   int64
}

// WalletBalanceMismatch represents the result of wallet balance validation query
// Used to find wallets where the balance doesn't match the sum of ledger entries
type WalletBalanceMismatch struct {
	WalletID      uuid.UUID
	ActualBalance int64
	LedgerBalance int64
	Currency      string
}
