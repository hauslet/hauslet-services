package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hauslet/internal/platform/breaker"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	stripeBaseURL    = "https://api.stripe.com/v1"
	stripeAPIVersion = "2023-10-16"
)

// StripeAdapter implements TransactionClient, PayoutClient, and WebhookHandler for Stripe
type StripeAdapter struct {
	secretKey      string
	webhookSecret  string
	httpClient     *http.Client
	circuitBreaker breaker.CircuitBreaker
}

// Ensure StripeAdapter implements all interfaces
var _ TransactionClient = (*StripeAdapter)(nil)
var _ PayoutClient = (*StripeAdapter)(nil)
var _ WebhookHandler = (*StripeAdapter)(nil)

// NewStripeAdapter creates a new Stripe payment adapter
func NewStripeAdapter(secretKey, webhookSecret string, cb breaker.CircuitBreaker) *StripeAdapter {
	return &StripeAdapter{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		circuitBreaker: cb,
	}
}

// WithHTTPClient allows overriding the HTTP client (useful for testing)
func (s *StripeAdapter) WithHTTPClient(client *http.Client) *StripeAdapter {
	if client != nil {
		s.httpClient = client
	}
	return s
}

// ============================================================================
// TransactionClient Implementation
// ============================================================================

// AuthorizePayment verifies a payment method without capturing funds
// Stripe implementation: Not yet implemented - requires SetupIntent integration
func (s *StripeAdapter) AuthorizePayment(ctx context.Context, req AuthorizationRequest) (*AuthorizationResponse, error) {
	return nil, fmt.Errorf("stripe: AuthorizePayment not implemented - requires SetupIntent integration")
}

// Initialize creates a Payment Intent
func (s *StripeAdapter) Initialize(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("stripe: validation failed: %w", err)
	}

	params := url.Values{}
	params.Set("amount", strconv.FormatInt(req.Amount, 10))
	params.Set("currency", strings.ToLower(req.Currency.String()))
	params.Set("description", fmt.Sprintf("Payment for %s", req.Reference))

	// Add metadata
	if len(req.Metadata) > 0 {
		for k, v := range req.Metadata {
			params.Set(fmt.Sprintf("metadata[%s]", k), v)
		}
	}
	params.Set("metadata[reference]", req.Reference)
	params.Set("metadata[email]", req.Email)

	// Set return URL if provided
	if req.CallbackURL != "" {
		params.Set("return_url", req.CallbackURL)
	}

	result, err := s.makeRequest(ctx, "POST", "/payment_intents", params)
	if err != nil {
		return nil, fmt.Errorf("stripe: initialize failed: %w", err)
	}

	intentID, _ := result["id"].(string)
	clientSecret, _ := result["client_secret"].(string)
	status, _ := result["status"].(string)
	amount, _ := result["amount"].(float64)
	currency, _ := result["currency"].(string)

	normalizedStatus := NormalizeStatus(status, "stripe")

	return &PaymentResponse{
		Success:        true,
		Status:         normalizedStatus,
		TransactionID:  intentID,
		Reference:      req.Reference,
		RequiresAction: status == "requires_action" || status == "requires_payment_method",
		ActionPayload: map[string]interface{}{
			"client_secret": clientSecret,
		},
		Amount:   int64(amount),
		Currency: Currency(strings.ToUpper(currency)),
		Message:  "Payment intent created - use client_secret to complete payment",
	}, nil
}

// ChargeAuthorization charges a saved payment method
func (s *StripeAdapter) ChargeAuthorization(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("stripe: validation failed: %w", err)
	}

	if req.AuthToken == nil || *req.AuthToken == "" {
		return nil, fmt.Errorf("stripe: payment method ID is required")
	}

	params := url.Values{}
	params.Set("amount", strconv.FormatInt(req.Amount, 10))
	params.Set("currency", strings.ToLower(req.Currency.String()))
	params.Set("payment_method", *req.AuthToken)
	params.Set("confirm", "true")
	params.Set("description", fmt.Sprintf("Payment for %s", req.Reference))

	// Add metadata
	if len(req.Metadata) > 0 {
		for k, v := range req.Metadata {
			params.Set(fmt.Sprintf("metadata[%s]", k), v)
		}
	}
	params.Set("metadata[reference]", req.Reference)
	params.Set("metadata[email]", req.Email)

	result, err := s.makeRequest(ctx, "POST", "/payment_intents", params)
	if err != nil {
		return nil, fmt.Errorf("stripe: %w", ErrAuthorizationFailed)
	}

	intentID, _ := result["id"].(string)
	status, _ := result["status"].(string)
	amount, _ := result["amount"].(float64)
	currency, _ := result["currency"].(string)

	normalizedStatus := NormalizeStatus(status, "stripe")

	return &PaymentResponse{
		Success:        normalizedStatus == StatusSuccess,
		Status:         normalizedStatus,
		TransactionID:  intentID,
		Reference:      req.Reference,
		RequiresAction: status == "requires_action",
		Amount:         int64(amount),
		Currency:       Currency(strings.ToUpper(currency)),
		Message:        fmt.Sprintf("Payment intent %s", status),
	}, nil
}

// Verify retrieves a Payment Intent to check its status
func (s *StripeAdapter) Verify(ctx context.Context, reference string) (*PaymentResponse, error) {
	if err := ValidateReference(reference); err != nil {
		return nil, fmt.Errorf("stripe: %w", err)
	}

	// Search for payment intent by metadata reference
	params := url.Values{}
	params.Set("limit", "1")

	endpoint := "/payment_intents?limit=1"
	result, err := s.makeRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: verify failed: %w", err)
	}

	// For simplicity, we'll use the payment intent ID as reference in real implementation
	// In production, you'd search by metadata or store the intent ID
	// For now, assume reference is the intent ID
	endpoint = fmt.Sprintf("/payment_intents/%s", reference)
	result, err = s.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: verify failed: %w", err)
	}

	intentID, _ := result["id"].(string)
	status, _ := result["status"].(string)
	amount, _ := result["amount"].(float64)
	currency, _ := result["currency"].(string)

	metadata, _ := result["metadata"].(map[string]interface{})
	ref := reference
	if metadata != nil {
		if metaRef, ok := metadata["reference"].(string); ok {
			ref = metaRef
		}
	}

	normalizedStatus := NormalizeStatus(status, "stripe")

	return &PaymentResponse{
		Success:       normalizedStatus == StatusSuccess,
		Status:        normalizedStatus,
		TransactionID: intentID,
		Reference:     ref,
		Amount:        int64(amount),
		Currency:      Currency(strings.ToUpper(currency)),
		Message:       fmt.Sprintf("Payment intent is %s", status),
	}, nil
}

// Refund creates a refund for a Payment Intent
func (s *StripeAdapter) Refund(ctx context.Context, originalTxID string, amount int64, reason string) (*RefundResponse, error) {
	if originalTxID == "" {
		return nil, fmt.Errorf("stripe: payment intent ID is required")
	}

	params := url.Values{}
	params.Set("payment_intent", originalTxID)

	if amount > 0 {
		params.Set("amount", strconv.FormatInt(amount, 10))
	}

	if reason != "" {
		params.Set("reason", reason)
	}

	result, err := s.makeRequest(ctx, "POST", "/refunds", params)
	if err != nil {
		return nil, fmt.Errorf("stripe: %w", ErrRefundFailed)
	}

	refundID, _ := result["id"].(string)
	status, _ := result["status"].(string)
	refundAmount, _ := result["amount"].(float64)
	currency, _ := result["currency"].(string)
	created, _ := result["created"].(float64)

	return &RefundResponse{
		Success:   true,
		RefundID:  refundID,
		Status:    NormalizeStatus(status, "stripe"),
		Amount:    int64(refundAmount),
		Currency:  Currency(strings.ToUpper(currency)),
		Message:   "Refund initiated successfully",
		CreatedAt: time.Unix(int64(created), 0).Format(time.RFC3339),
	}, nil
}

// ============================================================================
// PayoutClient Implementation
// ============================================================================

// ValidateAccount validates external bank account (Stripe uses manual validation in many regions)
func (s *StripeAdapter) ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	return "", fmt.Errorf("Not Supported: Stripe Payouts require Connected Accounts setup")
}

// CreateRecipient creates an external account for payouts
func (s *StripeAdapter) CreateRecipient(ctx context.Context, bankCode, accountNumber, accountName string) (string, error) {
	return "", fmt.Errorf("Not Supported: Stripe Payouts require Connected Accounts setup")
}

// Transfer initiates a payout
func (s *StripeAdapter) Transfer(ctx context.Context, req PayoutRequest) (*PayoutResponse, error) {
	return nil, fmt.Errorf("Not Supported: Stripe Payouts require Connected Accounts setup")
}

// VerifyTransfer checks the status of a transfer
func (s *StripeAdapter) VerifyTransfer(ctx context.Context, reference string) (*PayoutResponse, error) {
	return nil, fmt.Errorf("Not Supported: Stripe Payouts require Connected Accounts setup")
}

// ListBanks returns available banks for payouts (not supported for Stripe here).
func (s *StripeAdapter) ListBanks(ctx context.Context, currency Currency, country string) ([]Bank, error) {
	_ = ctx
	_ = currency
	_ = country
	return nil, fmt.Errorf("Not Supported: Stripe Payouts require Connected Accounts setup")
}

// ============================================================================
// WebhookHandler Implementation
// ============================================================================

// VerifySignature validates Stripe webhook signature
func (s *StripeAdapter) VerifySignature(headerSignature string, payload []byte) bool {
	// Stripe signature format: t=timestamp,v1=signature
	parts := strings.Split(headerSignature, ",")
	var timestamp, signature string

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signature = kv[1]
		}
	}

	if timestamp == "" || signature == "" {
		return false
	}

	// Create signed payload: timestamp.payload
	signedPayload := timestamp + "." + string(payload)

	// Compute HMAC SHA256
	hash := hmac.New(sha256.New, []byte(s.webhookSecret))
	hash.Write([]byte(signedPayload))
	expectedSignature := hex.EncodeToString(hash.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// ParseEvent converts Stripe webhook payload to UnifiedEvent
func (s *StripeAdapter) ParseEvent(payload []byte) (*UnifiedEvent, error) {
	var webhook struct {
		Type string `json:"type"`
		Data struct {
			Object map[string]interface{} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("stripe: failed to parse webhook: %w", err)
	}

	obj := webhook.Data.Object
	id, _ := obj["id"].(string)
	status, _ := obj["status"].(string)
	amount, _ := obj["amount"].(float64)
	currency, _ := obj["currency"].(string)

	// Extract reference from metadata
	reference := ""
	if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
		if ref, ok := metadata["reference"].(string); ok {
			reference = ref
		}
	}

	return &UnifiedEvent{
		Type:         webhook.Type,
		Reference:    reference,
		ProviderTxID: id,
		Status:       status,
		Amount:       int64(amount),
		Currency:     Currency(strings.ToUpper(currency)),
		RawData:      json.RawMessage(payload),
	}, nil
}

// ============================================================================
// HTTP Client Helper
// ============================================================================

func (s *StripeAdapter) makeRequest(ctx context.Context, method, endpoint string, params url.Values) (map[string]interface{}, error) {
	provider := "stripe"

	if s.circuitBreaker != nil {
		allowed, err := s.circuitBreaker.AllowRequest(ctx, provider)
		if err != nil {
			// Log error but default to allowed
		}
		if !allowed {
			return nil, fmt.Errorf("%w: circuit is open", ErrProviderUnavailable)
		}
	}

	apiURL := stripeBaseURL + endpoint

	var body io.Reader
	if params != nil {
		body = strings.NewReader(params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(s.secretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", stripeAPIVersion)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if s.circuitBreaker != nil {
			_ = s.circuitBreaker.RecordFailure(ctx, provider)
		}
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		if s.circuitBreaker != nil {
			_ = s.circuitBreaker.RecordFailure(ctx, provider)
		}
	} else {
		if s.circuitBreaker != nil {
			_ = s.circuitBreaker.RecordSuccess(ctx, provider)
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if errObj, ok := result["error"].(map[string]interface{}); ok {
		message, _ := errObj["message"].(string)
		code, _ := errObj["code"].(string)

		// Map common Stripe errors
		switch code {
		case "resource_already_exists":
			return nil, ErrDuplicateReference
		case "insufficient_funds":
			return nil, ErrInsufficientFunds
		case "resource_missing":
			return nil, ErrInvalidReference
		}

		return nil, fmt.Errorf("stripe error: %s", message)
	}

	return result, nil
}
