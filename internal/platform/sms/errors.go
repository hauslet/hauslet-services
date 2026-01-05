package sms

import "errors"

var (
	// ErrProviderUnavailable indicates the SMS provider is unreachable
	ErrProviderUnavailable = errors.New("sms provider is currently unavailable")

	// ErrInvalidPhoneNumber indicates the phone number format is invalid
	ErrInvalidPhoneNumber = errors.New("invalid phone number format")

	// ErrMessageTooLong indicates the message exceeds provider limits
	ErrMessageTooLong = errors.New("message exceeds maximum length")

	// ErrInsufficientBalance indicates provider account has insufficient credits
	ErrInsufficientBalance = errors.New("insufficient sms credits")

	// ErrRateLimited indicates too many requests to provider
	ErrRateLimited = errors.New("sms rate limit exceeded")

	// ErrAllProvidersFailed indicates both primary and fallback providers failed
	ErrAllProvidersFailed = errors.New("all sms providers failed")

	// ErrInvalidSenderID indicates the sender ID is not configured
	ErrInvalidSenderID = errors.New("invalid sender id")
)
