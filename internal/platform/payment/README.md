# Payment Platform Abstraction

A unified payment provider abstraction layer that supports **Paystack** (NGN) and **Flutterwave** (USD/GHS).

## Architecture

```txt
internal/platform/payment/
├── interface.go              # Core interfaces (TransactionClient, PayoutClient, WebhookHandler)
├── dto.go                    # Unified domain models (Currency, PaymentRequest, PaymentResponse, etc.)
├── client.go                 # High-level wrapper with delegation pattern
├── factory.go                # Provider selection based on currency
├── helpers.go                # Utility functions (amount conversion, validation)
├── errors.go                 # Custom error types
├── paystack_adapter.go       # Paystack implementation (NGN)
├── flutterwave_adapter.go    # Flutterwave v4 implementation (USD, GHS)
├── example_usage.go          # Usage examples (DO NOT IMPORT)
└── README.md                 # This file
```

## Key Features

### 1. **Provider Abstraction**

Your business logic stays provider-agnostic. Just specify currency:

- **NGN** → Automatically routes to Paystack
- **USD, GHS** → Automatically routes to Flutterwave

### 2. **Unified Interfaces**

#### TransactionClient (Guest Payments)

```go
Initialize()           // Create checkout URL or Payment Intent
ChargeAuthorization()  // Charge saved card (one-click payment)
Verify()              // Check transaction status
Refund()              // Process refund
```

#### PayoutClient (Host Disbursements)

```go
ValidateAccount()     // Verify bank account
CreateRecipient()     // Register beneficiary
Transfer()            // Send money
VerifyTransfer()      // Check transfer status
```

#### WebhookHandler (Event Processing)

```go
VerifySignature()     // Validate webhook authenticity
ParseEvent()          // Convert to unified event structure
```

### 3. **Normalized Data Models**

- `PaymentRequest` / `PaymentResponse` - Transaction operations
- `PayoutRequest` / `PayoutResponse` - Disbursement operations
- `UnifiedEvent` - Webhook events from any provider

## Quick Start

### 1. Initialize Client

```go
import (
    "hauslet/config"
    "hauslet/internal/platform/payment"
)

// In your app initialization
cfg := config.Load()
factory := payment.NewProviderFactory(cfg.Services.Payment)
paymentClient := payment.New(factory)
```

### 2. Process Payment

```go
// Guest pays for booking
resp, err := paymentClient.Initialize(ctx, payment.PaymentRequest{
    Amount:      payment.ToMinorUnits(1000.50, payment.NGN),
    Currency:    payment.NGN,
    Reference:   "BKG-12345",
    Email:       "guest@example.com",
    CallbackURL: "https://app.hauslet.com/callback",
    Metadata: map[string]string{
        "booking_id": "12345",
    },
})

// NGN → Paystack: redirect to resp.RedirectURL
// USD/GHS → Flutterwave: redirect to resp.RedirectURL
```

### 3. Process Payout

```go
// Release escrow to host
resp, err := paymentClient.Transfer(ctx, payment.PayoutRequest{
    Amount:        payment.ToMinorUnits(4500, payment.NGN),
    Currency:      payment.NGN,
    RecipientCode: host.RecipientCode, // From CreateRecipient()
    Reference:     "PAYOUT-BKG-12345",
    Narration:     "Payout for booking #12345",
})
```

### 4. Handle Webhooks

```go
// Verify and parse webhook
valid, _ := paymentClient.VerifyWebhookSignature(
    "paystack",
    r.Header.Get("X-Paystack-Signature"),
    payload,
)

event, _ := paymentClient.ParseWebhookEvent("paystack", payload)

switch event.Type {
case "charge.success":
    // Payment successful
case "transfer.success":
    // Payout successful
}
```

## Configuration

The client uses your existing config at `config.Services.Payment`:

```go
// .env file
PAYSTACK_SECRET_KEY=sk_test_...
PAYSTACK_PUBLIC_KEY=pk_test_...
FLUTTERWAVE_WEBHOOK_SECRET=your-flutterwave-webhook-secret
FLUTTERWAVE_OAUTH_CLIENT_ID=your-flutterwave-oauth-client-id
FLUTTERWAVE_OAUTH_SECRET=your-flutterwave-oauth-secret
```

Already configured in [config/config.go:106-112](../../config/config.go#L106-L112)

## Currency Routing

| Currency | Provider | Use Case |
|----------|----------|----------|
| NGN | Paystack | Nigerian market |
| USD | Flutterwave | International |
| GHS | Flutterwave | Ghanaian market |

## Helper Functions

```go
// Amount conversion
kobo := payment.ToMinorUnits(1000.50, payment.NGN)  // 100050
naira := payment.FromMinorUnits(100050, payment.NGN) // 1000.50

// Display formatting
formatted := payment.FormatAmount(100050, payment.NGN) // "NGN 1,000.50"
symbol := payment.GetCurrencySymbol(payment.NGN)       // "₦"

// Validation
err := payment.ValidateReference("BKG-12345")

// Status normalization (automatic)
status := payment.NormalizeStatus("success", "paystack")      // StatusSuccess
status := payment.NormalizeStatus("successful", "flutterwave")  // StatusSuccess
```

## Error Handling

The platform defines custom errors for common scenarios:

```go
ErrProviderUnavailable     // Payment provider unreachable
ErrInvalidCurrency         // Unsupported currency
ErrInsufficientFunds       // Insufficient balance
ErrInvalidReference        // Transaction not found
ErrDuplicateReference      // Reference already used
ErrInvalidAccount          // Bank account validation failed
ErrAuthorizationFailed     // Saved card charge failed
ErrWebhookVerificationFailed // Invalid webhook signature
ErrRefundFailed            // Refund operation failed
ErrTransferFailed          // Payout operation failed
ErrNoProvider              // No provider for currency
```

## Integration with Future Modules

This platform layer will be used by your upcoming `internal/modules/payment` (transaction) module:

```txt
┌─────────────────┐
│ Booking Module  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Payment Module  │ ← Business logic (transactions, escrow, reconciliation)
│ (To be built)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ platform/payment│ ← Technical abstraction (THIS PACKAGE)
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌─────────┐ ┌──────┐
│Paystack │ │Flutterwave│
└─────────┘ └──────┘
```

## Testing

The adapters support HTTP client injection for testing:

```go
mockClient := &http.Client{...}

adapter := payment.NewPaystackAdapter(secretKey).
    WithHTTPClient(mockClient)
```

## Next Steps

1. **Build the Payment/Transaction Module** - Business logic layer
2. **Implement webhook endpoints** - Process payment events
3. **Add transaction logging** - Record all payment operations
4. **Implement escrow logic** - Hold and release funds
5. **Add reconciliation** - Match payments with bookings

## API Documentation

- [Paystack API Docs](https://paystack.com/docs/api/)
- [Flutterwave API Docs](https://developer.flutterwave.com/reference)

## Notes

- All amounts are in **minor units** (kobo/cents)
- Use `ToMinorUnits()` / `FromMinorUnits()` helpers
- References must be **unique** per transaction
- Store recipient codes in database for reuse
- Always verify webhooks before processing
- Handle redirect flows for both Paystack and Flutterwave
