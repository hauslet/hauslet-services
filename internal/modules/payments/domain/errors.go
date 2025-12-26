package domain

import "errors"

var (
	// Payment errors
	ErrPaymentNotFound        = errors.New("payment not found")
	ErrPaymentAlreadyExists   = errors.New("payment already exists")
	ErrInvalidPaymentAmount   = errors.New("invalid payment amount")
	ErrInvalidPaymentCurrency = errors.New("invalid payment currency for market")
	ErrPaymentAlreadyPaid     = errors.New("payment already completed")
	ErrPaymentCancelled       = errors.New("payment has been cancelled")
	ErrCannotRefundPayment    = errors.New("payment cannot be refunded")
	ErrRefundAmountExceeded   = errors.New("refund amount exceeds payment amount")

	// Payment method errors
	ErrPaymentMethodNotFound      = errors.New("payment method not found")
	ErrPaymentMethodAlreadyExists = errors.New("payment method already exists")
	ErrInvalidPaymentMethod       = errors.New("invalid payment method")
	ErrPaymentMethodExpired       = errors.New("payment method has expired")

	// Transaction errors
	ErrTransactionNotFound      = errors.New("transaction not found")
	ErrTransactionAlreadyExists = errors.New("transaction already exists")
	ErrInvalidTransactionAmount = errors.New("invalid transaction amount")
	ErrTransactionFailed        = errors.New("transaction failed")

	// Payout detail errors
	ErrPayoutDetailNotFound      = errors.New("payout detail not found")
	ErrPayoutDetailAlreadyExists = errors.New("payout detail already exists")
	ErrInvalidBankDetails        = errors.New("invalid bank account details")
	ErrBankAccountNotVerified    = errors.New("bank account not verified")

	// Authorization errors
	ErrUnauthorized           = errors.New("unauthorized to perform this operation")
	ErrInsufficientPermission = errors.New("insufficient permission")

	// Validation errors
	ErrInvalidInput         = errors.New("invalid input provided")
	ErrMissingRequiredField = errors.New("required field is missing")
	ErrInvalidReference     = errors.New("invalid payment reference")
	ErrInvalidResourceType  = errors.New("invalid resource type for payment")
)
