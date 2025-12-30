package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ReconciliationReport represents a financial reconciliation run
type ReconciliationReport struct {
	ID                 uuid.UUID
	Status             ReconciliationStatus
	StartedAt          time.Time
	CompletedAt        *time.Time
	TotalWalletsChecked int
	TotalTransactionsChecked int
	DiscrepanciesFound int
	Discrepancies      []Discrepancy
	Summary            string
	ErrorMessage       *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Discrepancy represents a found issue during reconciliation
type Discrepancy struct {
	ID           uuid.UUID
	ReportID     uuid.UUID
	Type         DiscrepancyType
	Severity     DiscrepancySeverity
	WalletID     *uuid.UUID
	TransactionID *uuid.UUID
	Description  string
	ExpectedValue *int64
	ActualValue   *int64
	Details      map[string]interface{} // Additional context (JSONB)
	CreatedAt    time.Time
}

// DiscrepancyType represents the type of reconciliation issue
type DiscrepancyType string

const (
	DiscrepancyTypeLedgerImbalance    DiscrepancyType = "ledger_imbalance"     // Debit ≠ Credit
	DiscrepancyTypeWalletMismatch     DiscrepancyType = "wallet_mismatch"      // Wallet balance ≠ sum of ledger
	DiscrepancyTypeOrphanedWallet     DiscrepancyType = "orphaned_wallet"      // Wallet with balance but no ledger
	DiscrepancyTypeMissingTransaction DiscrepancyType = "missing_transaction"  // Transaction in provider but not in system
	DiscrepancyTypeDuplicateTransaction DiscrepancyType = "duplicate_transaction" // Duplicate ledger entries
	DiscrepancyTypeProviderMismatch   DiscrepancyType = "provider_mismatch"    // System vs Provider settlement mismatch
)

func (d DiscrepancyType) String() string {
	return string(d)
}

// DiscrepancySeverity represents how critical a discrepancy is
type DiscrepancySeverity string

const (
	DiscrepancySeverityLow      DiscrepancySeverity = "low"      // Minor, can be ignored
	DiscrepancySeverityMedium   DiscrepancySeverity = "medium"   // Should be reviewed
	DiscrepancySeverityHigh     DiscrepancySeverity = "high"     // Requires immediate attention
	DiscrepancySeverityCritical DiscrepancySeverity = "critical" // Major financial integrity issue
)

func (d DiscrepancySeverity) String() string {
	return string(d)
}

// ReconciliationStatus represents the status of a reconciliation run
type ReconciliationStatus string

const (
	ReconciliationStatusRunning   ReconciliationStatus = "running"
	ReconciliationStatusCompleted ReconciliationStatus = "completed"
	ReconciliationStatusFailed    ReconciliationStatus = "failed"
)

func (r ReconciliationStatus) String() string {
	return string(r)
}

// ReconciliationResult holds the result of validation checks
type ReconciliationResult struct {
	WalletsChecked      int
	TransactionsChecked int
	Discrepancies       []Discrepancy
}

// AddDiscrepancy adds a discrepancy to the result
func (r *ReconciliationResult) AddDiscrepancy(discrepancy Discrepancy) {
	r.Discrepancies = append(r.Discrepancies, discrepancy)
}

// HasIssues returns true if any discrepancies were found
func (r *ReconciliationResult) HasIssues() bool {
	return len(r.Discrepancies) > 0
}

// GetSummary generates a human-readable summary of the reconciliation
func (r *ReconciliationResult) GetSummary() string {
	if !r.HasIssues() {
		return "All checks passed. No discrepancies found."
	}

	critical := 0
	high := 0
	medium := 0
	low := 0

	for _, d := range r.Discrepancies {
		switch d.Severity {
		case DiscrepancySeverityCritical:
			critical++
		case DiscrepancySeverityHigh:
			high++
		case DiscrepancySeverityMedium:
			medium++
		case DiscrepancySeverityLow:
			low++
		}
	}

	return fmt.Sprintf("Found %d discrepancies: %d critical, %d high, %d medium, %d low",
		len(r.Discrepancies), critical, high, medium, low)
}
