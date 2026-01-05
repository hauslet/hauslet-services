package kyc

import (
	"slices"
	"fmt"
	"hauslet/config"
)

// DefaultProviderFactory implements ProviderFactory with country-based routing
type DefaultProviderFactory struct {
	dojahAdapter  *DojahAdapter
	veriffAdapter *VeriffAdapter
}

// NewProviderFactory creates a new provider factory with configured adapters
func NewProviderFactory(cfg config.KYCConfig) *DefaultProviderFactory {
	return &DefaultProviderFactory{
		dojahAdapter:  NewDojahAdapter(cfg.DojahAPIKey, cfg.DojahSecretKey, cfg.DojahWebhookSecret),
		veriffAdapter: NewVeriffAdapter(cfg.VeriffAPIKey, cfg.VeriffSecretKey, cfg.VeriffWebhookSecret),
	}
}

// GetProvider returns the appropriate provider based on country
// Routing logic:
// - Nigeria (NG) and supported African countries → Dojah
// - All other countries → Veriff (global coverage)
func (f *DefaultProviderFactory) GetProvider(country string) (KYCProvider, error) {
	if country == "" {
		return nil, fmt.Errorf("country code is required")
	}

	// Check if Dojah supports this country
	dojahCountries := f.dojahAdapter.SupportedCountries()
	if slices.Contains(dojahCountries, country) {
			return f.dojahAdapter, nil
		}

	// Default to Veriff for global coverage
	return f.veriffAdapter, nil
}

// GetWebhookHandler returns the provider by name for webhook processing
func (f *DefaultProviderFactory) GetWebhookHandler(providerName string) (KYCProvider, error) {
	switch providerName {
	case "dojah":
		return f.dojahAdapter, nil
	case "veriff":
		return f.veriffAdapter, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidProvider, providerName)
	}
}

// ListProviders returns all configured providers
func (f *DefaultProviderFactory) ListProviders() []KYCProvider {
	return []KYCProvider{
		f.dojahAdapter,
		f.veriffAdapter,
	}
}

// GetDojahAdapter returns the Dojah adapter (for direct access if needed)
func (f *DefaultProviderFactory) GetDojahAdapter() *DojahAdapter {
	return f.dojahAdapter
}

// GetVeriffAdapter returns the Veriff adapter (for direct access if needed)
func (f *DefaultProviderFactory) GetVeriffAdapter() *VeriffAdapter {
	return f.veriffAdapter
}
