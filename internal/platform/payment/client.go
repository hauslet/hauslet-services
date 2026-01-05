package payment

import "context"

// Client is a high-level wrapper that delegates to the appropriate payment provider
// based on currency. It provides a unified interface for payment operations.
type Client struct {
	factory ProviderFactory
}

// New creates a new payment client with the given provider factory
func New(factory ProviderFactory) *Client {
	return &Client{factory: factory}
}

// ============================================================================
// Transaction Operations (Guest Payment Flows)
// ============================================================================

// Initialize creates a new payment transaction
// Returns payment response with redirect URL or action payload based on provider
func (c *Client) Initialize(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	client, err := c.factory.GetTransactionClient(req.Currency)
	if err != nil {
		return nil, err
	}
	return client.Initialize(ctx, req)
}

// AuthorizePayment initiates a card authorization flow without full charge
// Returns authorization response with checkout URL to complete authorization
func (c *Client) AuthorizePayment(ctx context.Context, currency Currency, req AuthorizationRequest) (*AuthorizationResponse, error) {
	client, err := c.factory.GetTransactionClient(currency)
	if err != nil {
		return nil, err
	}
	return client.AuthorizePayment(ctx, req)
}

// ChargeAuthorization charges a saved payment method (one-click payment)
// Requires AuthToken to be set in the request
func (c *Client) ChargeAuthorization(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	client, err := c.factory.GetTransactionClient(req.Currency)
	if err != nil {
		return nil, err
	}
	return client.ChargeAuthorization(ctx, req)
}

// Verify checks the status of a payment transaction
// Important: Pass the currency in the context or use VerifyWithCurrency
func (c *Client) Verify(ctx context.Context, currency Currency, reference string) (*PaymentResponse, error) {
	client, err := c.factory.GetTransactionClient(currency)
	if err != nil {
		return nil, err
	}
	return client.Verify(ctx, reference)
}

// Refund processes a refund for a completed transaction
func (c *Client) Refund(ctx context.Context, currency Currency, originalTxID string, amount int64, reason string) (*RefundResponse, error) {
	client, err := c.factory.GetTransactionClient(currency)
	if err != nil {
		return nil, err
	}
	return client.Refund(ctx, originalTxID, amount, reason)
}

// ============================================================================
// Payout Operations (Host Payment Flows)
// ============================================================================

// ValidateAccount validates a bank account and returns the account name
// Used before creating a recipient to confirm account details
func (c *Client) ValidateAccount(ctx context.Context, currency Currency, bankCode, accountNumber string) (string, error) {
	client, err := c.factory.GetPayoutClient(currency)
	if err != nil {
		return "", err
	}
	return client.ValidateAccount(ctx, bankCode, accountNumber)
}

// CreateRecipient creates a transfer recipient/beneficiary
// Returns a recipient code to be stored in the database
func (c *Client) CreateRecipient(ctx context.Context, currency Currency, bankCode, accountNumber, accountName string) (string, error) {
	client, err := c.factory.GetPayoutClient(currency)
	if err != nil {
		return "", err
	}
	return client.CreateRecipient(ctx, bankCode, accountNumber, accountName)
}

// Transfer initiates a payout to a recipient
// Returns transfer response with status and transaction ID
func (c *Client) Transfer(ctx context.Context, req PayoutRequest) (*PayoutResponse, error) {
	client, err := c.factory.GetPayoutClient(req.Currency)
	if err != nil {
		return nil, err
	}
	return client.Transfer(ctx, req)
}

// VerifyTransfer checks the status of a payout transaction
func (c *Client) VerifyTransfer(ctx context.Context, currency Currency, reference string) (*PayoutResponse, error) {
	client, err := c.factory.GetPayoutClient(currency)
	if err != nil {
		return nil, err
	}
	return client.VerifyTransfer(ctx, reference)
}

// ListBanks returns banks supported by the payout provider for a currency/country.
func (c *Client) ListBanks(ctx context.Context, currency Currency, country string) ([]Bank, error) {
	client, err := c.factory.GetPayoutClient(currency)
	if err != nil {
		return nil, err
	}
	return client.ListBanks(ctx, currency, country)
}

// ============================================================================
// Webhook Operations
// ============================================================================

// VerifyWebhookSignature validates a webhook signature from a provider
// Provider should be "paystack" or "stripe"
func (c *Client) VerifyWebhookSignature(provider, headerSignature string, payload []byte) (bool, error) {
	handler, err := c.factory.GetWebhookHandler(provider)
	if err != nil {
		return false, err
	}
	return handler.VerifySignature(headerSignature, payload), nil
}

// ParseWebhookEvent converts a webhook payload to a unified event structure
// Provider should be "paystack" or "stripe"
func (c *Client) ParseWebhookEvent(provider string, payload []byte) (*UnifiedEvent, error) {
	handler, err := c.factory.GetWebhookHandler(provider)
	if err != nil {
		return nil, err
	}
	return handler.ParseEvent(payload)
}

// ============================================================================
// Convenience Methods
// ============================================================================

// ProcessPayment is a convenience method that handles the full payment flow
// It validates the request, initializes the payment, and returns the response
func (c *Client) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// If auth token is provided, use saved payment method
	if req.AuthToken != nil && *req.AuthToken != "" {
		return c.ChargeAuthorization(ctx, req)
	}

	// Otherwise, initialize new payment
	return c.Initialize(ctx, req)
}

// ProcessPayout is a convenience method that handles the full payout flow
// It validates the request and initiates the transfer
func (c *Client) ProcessPayout(ctx context.Context, req PayoutRequest) (*PayoutResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return c.Transfer(ctx, req)
}
