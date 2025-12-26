package domain

import "errors"

var (
	// Wallet errors
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrWalletFrozen        = errors.New("wallet is frozen")
	ErrWalletClosed        = errors.New("wallet is closed")
	ErrInvalidWalletType   = errors.New("invalid wallet type")

	// Transaction errors
	ErrDuplicateTransaction = errors.New("duplicate transaction detected")
	ErrInvalidAmount        = errors.New("amount must be positive")
	ErrLedgerImbalance      = errors.New("ledger entries don't balance")
	ErrTransactionNotFound  = errors.New("transaction not found")

	// Disbursement errors
	ErrDisbursementNotFound = errors.New("disbursement not found")
	ErrDisbursementFailed   = errors.New("disbursement failed")
	ErrMaxRetriesExceeded   = errors.New("maximum retry attempts exceeded")

	// General errors
	ErrInvalidCurrency = errors.New("invalid currency")
	ErrInvalidOwner    = errors.New("invalid owner type or ID")
)
