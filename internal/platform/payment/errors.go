package payment

import "errors"

var (
	// ErrProviderUnavailable indicates the payment provider is unreachable
	ErrProviderUnavailable = errors.New("payment provider is currently unavailable")

	// ErrInvalidCurrency indicates an unsupported currency was provided
	ErrInvalidCurrency = errors.New("currency not supported by this provider")

	// ErrInsufficientFunds indicates insufficient balance for the operation
	ErrInsufficientFunds = errors.New("insufficient funds to complete transaction")

	// ErrInvalidReference indicates the transaction reference was not found
	ErrInvalidReference = errors.New("transaction reference not found")

	// ErrDuplicateReference indicates the reference has already been used
	ErrDuplicateReference = errors.New("transaction reference already exists")

	// ErrInvalidAccount indicates bank account validation failed
	ErrInvalidAccount = errors.New("invalid bank account details")

	// ErrAuthorizationFailed indicates saved card charge failed
	ErrAuthorizationFailed = errors.New("authorization charge failed")

	// ErrWebhookVerificationFailed indicates webhook signature validation failed
	ErrWebhookVerificationFailed = errors.New("webhook signature verification failed")

	// ErrRefundFailed indicates refund operation failed
	ErrRefundFailed = errors.New("refund operation failed")

	// ErrTransferFailed indicates payout operation failed
	ErrTransferFailed = errors.New("transfer operation failed")

	// ErrInvalidAmount indicates amount is invalid or outside acceptable range
	ErrInvalidAmount = errors.New("invalid transaction amount")

	// ErrNoProvider indicates no provider configured for the given currency
	ErrNoProvider = errors.New("no payment provider configured for currency")
)
