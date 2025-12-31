package payment

import "context"

// TransactionClient handles guest-side payment operations (collecting money)
type TransactionClient interface {
	// Initialize generates a checkout link or client secret for first-time payment
	// Returns payment response with redirect URL (Paystack) or action payload (Stripe)
	Initialize(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)

	// AuthorizePayment verifies a payment method without capturing funds
	// Returns an authorization token/code for deferred charges
	// Paystack: zero-amount or minimal charge authorization
	// Stripe: SetupIntent / PaymentMethod
	AuthorizePayment(ctx context.Context, req AuthorizationRequest) (*AuthorizationResponse, error)

	// ChargeAuthorization charges a saved card token (one-click payments)
	// Used for recurring payments or saved payment methods
	ChargeAuthorization(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)

	// Verify checks the status of a transaction by reference
	// Critical for redirect flow callbacks and webhook verification
	Verify(ctx context.Context, reference string) (*PaymentResponse, error)

	// Refund returns money to the guest for a completed transaction
	Refund(ctx context.Context, originalTxID string, amount int64, reason string) (*RefundResponse, error)
}

// PayoutClient handles host-side payment operations (sending money)
type PayoutClient interface {
	// ValidateAccount checks if bank account exists and returns account name
	// Used before creating recipient to confirm account details
	ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error)

	// CreateRecipient registers a beneficiary with the payment provider
	// Returns a recipient code to store in database for future transfers
	CreateRecipient(ctx context.Context, bankCode, accountNumber, accountName string) (string, error)

	// Transfer moves money from platform balance to recipient account
	// Returns transfer response with transaction ID and status
	Transfer(ctx context.Context, req PayoutRequest) (*PayoutResponse, error)

	// VerifyTransfer checks the status of a payout transaction
	// Used to confirm transfer completion or failure
	VerifyTransfer(ctx context.Context, reference string) (*PayoutResponse, error)

	// ListBanks returns supported banks for the given currency/country.
	// Country is provider-specific (e.g., "nigeria").
	ListBanks(ctx context.Context, currency Currency, country string) ([]Bank, error)
}

// WebhookHandler processes and validates provider webhook events
type WebhookHandler interface {
	// VerifySignature validates webhook authenticity using HMAC/signature
	// Prevents webhook spoofing attacks
	VerifySignature(headerSignature string, payload []byte) bool

	// ParseEvent converts raw webhook JSON into unified event structure
	// Normalizes different provider event formats
	ParseEvent(payload []byte) (*UnifiedEvent, error)
}
