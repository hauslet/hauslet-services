package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"time"

	"github.com/google/uuid"
)

// generateReference creates an idempotency key for transactions
// Format: hash(txType + resourceID + amount + nonce)
func generateReference(txType domain.TransactionType, resourceID uuid.UUID, amount int64, nonce string) string {
	data := fmt.Sprintf("%s:%s:%d:%s", txType, resourceID.String(), amount, nonce)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateTransactionReference creates a unique reference for a transaction
// Uses payment ID as nonce for charge/refund, timestamp for commission/payout
func generateTransactionReference(txType domain.TransactionType, resourceID, paymentID uuid.UUID) string {
	return generateReference(txType, resourceID, 0, paymentID.String())
}

// generateTimestampReference creates a reference using timestamp as nonce
func generateTimestampReference(txType domain.TransactionType, resourceID uuid.UUID, amount int64) string {
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	return generateReference(txType, resourceID, amount, nonce)
}

// validateAmount ensures the amount is positive
func validateAmount(amount int64) error {
	if amount <= 0 {
		return domain.ErrInvalidAmount
	}
	return nil
}

// calculateCommission calculates the platform commission
// commissionPercent should be from config (e.g., 12.5 for 12.5%)
func calculateCommission(amount int64, commissionPercent float64) int64 {
	return int64(float64(amount) * (commissionPercent / 100.0))
}

// buildLedgerEntry creates a single ledger entry
func buildLedgerEntry(
	transactionID uuid.UUID,
	reference string,
	debitWalletID, creditWalletID *uuid.UUID,
	amount int64,
	currency string,
	resourceType domain.ResourceType,
	resourceID uuid.UUID,
	memo string,
) *domain.LedgerEntry {
	return &domain.LedgerEntry{
		ID:             uuid.New(),
		TransactionID:  transactionID,
		Reference:      reference,
		DebitWalletID:  debitWalletID,
		CreditWalletID: creditWalletID,
		Amount:         amount,
		Currency:       currency,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		Memo:           memo,
		CreatedAt:      time.Now(),
	}
}

// buildDoubleEntry creates a pair of ledger entries (debit and credit)
func buildDoubleEntry(
	transactionID uuid.UUID,
	reference string,
	debitWalletID, creditWalletID uuid.UUID,
	amount int64,
	currency string,
	resourceType domain.ResourceType,
	resourceID uuid.UUID,
	memo string,
) []*domain.LedgerEntry {
	debitEntry := buildLedgerEntry(
		transactionID,
		reference+":debit",
		&debitWalletID,
		nil,
		amount,
		currency,
		resourceType,
		resourceID,
		memo,
	)

	creditEntry := buildLedgerEntry(
		transactionID,
		reference+":credit",
		nil,
		&creditWalletID,
		amount,
		currency,
		resourceType,
		resourceID,
		memo,
	)

	return []*domain.LedgerEntry{debitEntry, creditEntry}
}

// validateDoubleEntry ensures ledger entries balance
func validateDoubleEntry(entries []*domain.LedgerEntry) error {
	if len(entries) != 2 {
		return domain.ErrLedgerImbalance
	}

	var totalDebit, totalCredit int64
	for _, entry := range entries {
		if err := entry.Validate(); err != nil {
			return err
		}

		if entry.IsDebit() {
			totalDebit += entry.Amount
		}
		if entry.IsCredit() {
			totalCredit += entry.Amount
		}
	}

	if totalDebit != totalCredit {
		return domain.ErrLedgerImbalance
	}

	return nil
}

// maskAccountNumber masks all but the last 4 digits of an account number
func maskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return accountNumber
	}
	masked := ""
	for i := 0; i < len(accountNumber)-4; i++ {
		masked += "*"
	}
	return masked + accountNumber[len(accountNumber)-4:]
}
