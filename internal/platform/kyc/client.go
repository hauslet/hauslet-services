package kyc

import (
	"context"
	"fmt"
	"time"
)

// Client is the high-level KYC service used by modules
type Client struct {
	factory ProviderFactory
}

// New creates a Client with the provider factory
func New(factory ProviderFactory) *Client {
	return &Client{
		factory: factory,
	}
}

// SubmitVerification submits a verification request using the appropriate provider
// Automatically selects provider based on country
func (c *Client) SubmitVerification(ctx context.Context, req VerificationRequest) (*VerificationResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid verification request: %w", err)
	}

	// Get appropriate provider for country
	provider, err := c.factory.GetProvider(req.Country)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Track start time for processing duration
	startTime := time.Now()

	// Submit to provider
	resp, err := provider.SubmitVerification(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("provider submission failed: %w", err)
	}

	// Set processing time
	resp.ProcessingTime = time.Since(startTime)
	resp.Provider = provider.Name()

	return resp, nil
}

// ParseWebhook parses and validates a webhook from a specific provider
func (c *Client) ParseWebhook(ctx context.Context, providerName string, payload []byte, headers map[string]string) (*WebhookEvent, error) {
	// Get provider by name
	provider, err := c.factory.GetWebhookHandler(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook handler: %w", err)
	}

	// Parse webhook
	event, err := provider.ParseWebhook(payload, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	event.Provider = providerName
	event.ReceivedAt = time.Now()

	return event, nil
}

// VerifyWebhookSignature verifies the authenticity of a webhook
func (c *Client) VerifyWebhookSignature(providerName string, payload []byte, signature string) (bool, error) {
	// Get provider by name
	provider, err := c.factory.GetWebhookHandler(providerName)
	if err != nil {
		return false, fmt.Errorf("failed to get webhook handler: %w", err)
	}

	// Verify signature
	return provider.VerifySignature(payload, signature), nil
}

// EstimateCost estimates the cost for verification in a given country
func (c *Client) EstimateCost(country string) (float64, string, error) {
	// Get appropriate provider for country
	provider, err := c.factory.GetProvider(country)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get provider: %w", err)
	}

	// Get cost estimate
	cost, err := provider.EstimateCost(country)
	if err != nil {
		return 0, "", fmt.Errorf("failed to estimate cost: %w", err)
	}

	return cost, provider.Name(), nil
}

// HealthCheck verifies all providers are healthy
func (c *Client) HealthCheck(ctx context.Context) map[string]error {
	providers := c.factory.ListProviders()
	results := make(map[string]error)

	for _, provider := range providers {
		results[provider.Name()] = provider.HealthCheck(ctx)
	}

	return results
}
