package kyc

import "context"

// KYCProvider defines the interface that all KYC verification providers must implement
type KYCProvider interface {
	// Name returns the provider identifier (e.g., "dojah", "veriff")
	Name() string

	// SupportedCountries returns list of ISO 3166-1 alpha-2 country codes
	SupportedCountries() []string

	// SupportedDocuments returns list of document types supported by the provider
	SupportedDocuments() []DocumentType

	// SubmitVerification submits verification data to the provider
	// Returns provider reference and initial status
	SubmitVerification(ctx context.Context, req VerificationRequest) (*VerificationResponse, error)

	// ParseWebhook parses incoming webhook payload from provider
	// Returns normalized webhook event
	ParseWebhook(payload []byte, headers map[string]string) (*WebhookEvent, error)

	// VerifySignature validates webhook signature to prevent spoofing
	VerifySignature(payload []byte, signature string) bool

	// HealthCheck verifies provider API is reachable and credentials are valid
	HealthCheck(ctx context.Context) error

	// EstimateCost returns estimated cost for verification in this country
	EstimateCost(country string) (float64, error)

	// VerifyBusiness performs a business/KYB verification lookup.
	// Currently supported by Dojah for Nigerian businesses (CAC lookup).
	// Returns nil response if not supported by the provider.
	VerifyBusiness(ctx context.Context, registrationNumber, businessType string) (*BusinessVerificationResponse, error)
}

// ProviderFactory creates and routes to appropriate KYC providers
type ProviderFactory interface {
	// GetProvider returns the best provider for the given country
	// Routing priority: Dojah for Nigeria + supported African countries, Veriff for global
	GetProvider(country string) (KYCProvider, error)

	// GetWebhookHandler returns the provider by name for webhook processing
	GetWebhookHandler(providerName string) (KYCProvider, error)

	// ListProviders returns all configured providers
	ListProviders() []KYCProvider
}
