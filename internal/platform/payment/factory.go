package payment

import (
	"fmt"
	"hauslet/config"
	"hauslet/internal/platform/breaker"
)

// ProviderFactory creates and returns the appropriate payment provider based on currency
type ProviderFactory interface {
	// GetTransactionClient returns the transaction client for the given currency
	GetTransactionClient(currency Currency) (TransactionClient, error)

	// GetPayoutClient returns the payout client for the given currency
	GetPayoutClient(currency Currency) (PayoutClient, error)

	// GetWebhookHandler returns the webhook handler for the specified provider
	GetWebhookHandler(provider string) (WebhookHandler, error)
}

// DefaultProviderFactory implements ProviderFactory with currency-based routing
type DefaultProviderFactory struct {
	paystackAdapter    *PaystackAdapter
	flutterwaveAdapter *FlutterwaveAdapter
}

// NewProviderFactory creates a new provider factory with configured adapters
func NewProviderFactory(cfg config.PaymentConfig, cb breaker.CircuitBreaker) *DefaultProviderFactory {
	fw := NewFlutterwaveAdapter(cfg.FlutterwaveWebhookSecret, "", cb).
		WithOAuthCredentials(cfg.FlutterwaveOAuthClientID, cfg.FlutterwaveOAuthSecret)

	if cfg.FlutterwaveBaseURL != "" {
		fw = fw.WithBaseURL(cfg.FlutterwaveBaseURL)
	}

	return &DefaultProviderFactory{
		paystackAdapter:    NewPaystackAdapter(cfg.PaystackSecretKey, cb),
		flutterwaveAdapter: fw,
	}
}

// GetTransactionClient returns the appropriate transaction client based on currency
// Currency routing:
// - NGN → Paystack
// - USD, GHS → Flutterwave
func (f *DefaultProviderFactory) GetTransactionClient(currency Currency) (TransactionClient, error) {
	if !currency.IsValid() {
		return nil, ErrInvalidCurrency
	}

	switch currency {
	case NGN:
		return f.paystackAdapter, nil
	case USD, GHS:
		return f.flutterwaveAdapter, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrNoProvider, currency)
	}
}

// GetPayoutClient returns the appropriate payout client based on currency
// Currency routing:
// - NGN → Paystack
// - USD, GHS → Flutterwave
func (f *DefaultProviderFactory) GetPayoutClient(currency Currency) (PayoutClient, error) {
	if !currency.IsValid() {
		return nil, ErrInvalidCurrency
	}

	switch currency {
	case NGN:
		return f.paystackAdapter, nil
	case USD, GHS:
		return f.flutterwaveAdapter, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrNoProvider, currency)
	}
}

// GetWebhookHandler returns the webhook handler for a specific provider
// Provider names: "paystack", "flutterwave"
func (f *DefaultProviderFactory) GetWebhookHandler(provider string) (WebhookHandler, error) {
	switch provider {
	case "paystack":
		return f.paystackAdapter, nil
	case "flutterwave":
		return f.flutterwaveAdapter, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// GetPaystackAdapter returns the Paystack adapter (for direct access if needed)
func (f *DefaultProviderFactory) GetPaystackAdapter() *PaystackAdapter {
	return f.paystackAdapter
}

// GetFlutterwaveAdapter returns the Flutterwave adapter (for direct access if needed)
func (f *DefaultProviderFactory) GetFlutterwaveAdapter() *FlutterwaveAdapter {
	return f.flutterwaveAdapter
}
