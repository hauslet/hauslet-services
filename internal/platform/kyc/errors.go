package kyc

import "errors"

var (
	// ErrProviderUnavailable indicates the KYC provider is unreachable
	ErrProviderUnavailable = errors.New("kyc provider is currently unavailable")

	// ErrUnsupportedCountry indicates the country is not supported by available providers
	ErrUnsupportedCountry = errors.New("country not supported by any kyc provider")

	// ErrInvalidDocument indicates the document provided is invalid or unreadable
	ErrInvalidDocument = errors.New("invalid or unreadable document")

	// ErrInvalidImage indicates the image quality is too poor
	ErrInvalidImage = errors.New("image quality too poor for verification")

	// ErrDocumentExpired indicates the provided document has expired
	ErrDocumentExpired = errors.New("document has expired")

	// ErrFaceMismatch indicates the selfie doesn't match the document photo
	ErrFaceMismatch = errors.New("face does not match document photo")

	// ErrVerificationFailed indicates the verification was rejected by the provider
	ErrVerificationFailed = errors.New("verification failed")

	// ErrWebhookVerificationFailed indicates webhook signature validation failed
	ErrWebhookVerificationFailed = errors.New("webhook signature verification failed")

	// ErrInvalidProvider indicates an unknown provider was requested
	ErrInvalidProvider = errors.New("invalid provider name")

	// ErrNoProviderForCountry indicates no provider is configured for the country
	ErrNoProviderForCountry = errors.New("no kyc provider configured for country")

	// ErrProviderTimeout indicates the provider request timed out
	ErrProviderTimeout = errors.New("provider request timed out")
)
