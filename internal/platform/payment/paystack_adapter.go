package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	paystackBaseURL = "https://api.paystack.co"
)

// PaystackAdapter implements TransactionClient, PayoutClient, and WebhookHandler for Paystack
type PaystackAdapter struct {
	secretKey  string
	httpClient *http.Client
}

// Ensure PaystackAdapter implements all interfaces
var _ TransactionClient = (*PaystackAdapter)(nil)
var _ PayoutClient = (*PaystackAdapter)(nil)
var _ WebhookHandler = (*PaystackAdapter)(nil)

// NewPaystackAdapter creates a new Paystack payment adapter
func NewPaystackAdapter(secretKey string) *PaystackAdapter {
	return &PaystackAdapter{
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithHTTPClient allows overriding the HTTP client (useful for testing)
func (p *PaystackAdapter) WithHTTPClient(client *http.Client) *PaystackAdapter {
	if client != nil {
		p.httpClient = client
	}
	return p
}

// ============================================================================
// TransactionClient Implementation
// ============================================================================

// AuthorizePayment verifies a payment method without capturing full funds
// Paystack implementation: Creates a minimal authorization charge (100 kobo = ₦1)
// Returns authorization code that can be used for future charges
func (p *PaystackAdapter) AuthorizePayment(ctx context.Context, req AuthorizationRequest) (*AuthorizationResponse, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("paystack: email is required for authorization")
	}
	if req.Currency == "" {
		req.Currency = "NGN" // Default to NGN for Paystack
	}

	// Create a minimal authorization transaction (100 kobo = ₦1)
	payload := map[string]any{
		"amount":   100, // Minimal amount for authorization
		"email":    req.Email,
		"currency": req.Currency,
	}

	if req.CustomerID != "" {
		payload["customer"] = req.CustomerID
	}

	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	resp, err := p.makeRequest(ctx, "POST", "/transaction/initialize", payload)
	if err != nil {
		return nil, fmt.Errorf("paystack: authorization failed: %w", err)
	}

	data := resp["data"].(map[string]any)
	authURL, _ := data["authorization_url"].(string)
	accessCode, _ := data["access_code"].(string)
	reference, _ := data["reference"].(string)

	// Note: The actual authorization code is obtained after the customer completes the payment
	// This returns the checkout URL that the customer needs to complete
	// After completion, verify the transaction to get the authorization code
	return &AuthorizationResponse{
		AuthorizationCode: accessCode, // This is the access code, not the final auth code
		Reusable:          false,      // Will be true after verification
		Provider:          "paystack",
		Raw: map[string]any{
			"authorization_url": authURL,
			"access_code":       accessCode,
			"reference":         reference,
		},
	}, nil
}

// Initialize generates a payment checkout URL
func (p *PaystackAdapter) Initialize(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("paystack: validation failed: %w", err)
	}

	payload := map[string]any{
		"amount":       req.Amount,
		"email":        req.Email,
		"reference":    req.Reference,
		"currency":     req.Currency,
		"callback_url": req.CallbackURL,
	}

	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	resp, err := p.makeRequest(ctx, "POST", "/transaction/initialize", payload)
	if err != nil {
		return nil, fmt.Errorf("paystack: initialize failed: %w", err)
	}

	data := resp["data"].(map[string]any)
	authURL, _ := data["authorization_url"].(string)
	accessCode, _ := data["access_code"].(string)

	return &PaymentResponse{
		Success:       true,
		Status:        StatusPending,
		TransactionID: accessCode,
		Reference:     req.Reference,
		RedirectURL:   authURL,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Message:       "Redirect user to complete payment",
	}, nil
}

// ChargeAuthorization charges a saved authorization code (tokenized payment)
func (p *PaystackAdapter) ChargeAuthorization(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("paystack: validation failed: %w", err)
	}

	if req.AuthToken == nil || *req.AuthToken == "" {
		return nil, fmt.Errorf("paystack: authorization code is required")
	}

	payload := map[string]any{
		"amount":             req.Amount,
		"email":              req.Email,
		"authorization_code": *req.AuthToken,
		"reference":          req.Reference,
		"currency":           req.Currency,
	}

	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	resp, err := p.makeRequest(ctx, "POST", "/transaction/charge_authorization", payload)
	if err != nil {
		return nil, fmt.Errorf("paystack: %w", ErrAuthorizationFailed)
	}

	data := resp["data"].(map[string]any)
	status, _ := data["status"].(string)
	txID, _ := data["id"].(float64)
	message, _ := data["message"].(string)

	normalizedStatus := NormalizeStatus(status, "paystack")

	return &PaymentResponse{
		Success:       normalizedStatus == StatusSuccess,
		Status:        normalizedStatus,
		TransactionID: fmt.Sprintf("%.0f", txID),
		Reference:     req.Reference,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Message:       message,
	}, nil
}

// Verify checks the status of a transaction
func (p *PaystackAdapter) Verify(ctx context.Context, reference string) (*PaymentResponse, error) {
	if err := ValidateReference(reference); err != nil {
		return nil, fmt.Errorf("paystack: %w", err)
	}

	endpoint := fmt.Sprintf("/transaction/verify/%s", reference)
	resp, err := p.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("paystack: verify failed: %w", err)
	}

	data := resp["data"].(map[string]any)
	status, _ := data["status"].(string)
	txID, _ := data["id"].(float64)
	amount, _ := data["amount"].(float64)
	currency, _ := data["currency"].(string)
	ref, _ := data["reference"].(string)

	normalizedStatus := NormalizeStatus(status, "paystack")

	return &PaymentResponse{
		Success:       normalizedStatus == StatusSuccess,
		Status:        normalizedStatus,
		TransactionID: fmt.Sprintf("%.0f", txID),
		Reference:     ref,
		Amount:        int64(amount),
		Currency:      Currency(currency),
		Message:       fmt.Sprintf("Transaction is %s", status),
	}, nil
}

// Refund initiates a refund for a completed transaction
func (p *PaystackAdapter) Refund(ctx context.Context, originalTxID string, amount int64, reason string) (*RefundResponse, error) {
	if originalTxID == "" {
		return nil, fmt.Errorf("paystack: transaction ID is required")
	}

	payload := map[string]any{
		"transaction": originalTxID,
	}

	if amount > 0 {
		payload["amount"] = amount
	}

	if reason != "" {
		payload["merchant_note"] = reason
	}

	resp, err := p.makeRequest(ctx, "POST", "/refund", payload)
	if err != nil {
		return nil, fmt.Errorf("paystack: %w", ErrRefundFailed)
	}

	data := resp["data"].(map[string]any)
	refundID, _ := data["id"].(float64)
	status, _ := data["status"].(string)
	refundAmount, _ := data["amount"].(float64)
	currency, _ := data["currency"].(string)
	createdAt, _ := data["created_at"].(string)

	return &RefundResponse{
		Success:   true,
		RefundID:  fmt.Sprintf("%.0f", refundID),
		Status:    NormalizeStatus(status, "paystack"),
		Amount:    int64(refundAmount),
		Currency:  Currency(currency),
		Message:   "Refund initiated successfully",
		CreatedAt: createdAt,
	}, nil
}

// ============================================================================
// PayoutClient Implementation
// ============================================================================

// ValidateAccount resolves and validates a bank account
func (p *PaystackAdapter) ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	if bankCode == "" || accountNumber == "" {
		return "", fmt.Errorf("paystack: bank code and account number are required")
	}

	endpoint := fmt.Sprintf("/bank/resolve?account_number=%s&bank_code=%s", accountNumber, bankCode)
	resp, err := p.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("paystack: %w", ErrInvalidAccount)
	}

	data := resp["data"].(map[string]any)
	accountName, ok := data["account_name"].(string)
	if !ok || accountName == "" {
		return "", ErrInvalidAccount
	}

	return accountName, nil
}

// CreateRecipient creates a transfer recipient
func (p *PaystackAdapter) CreateRecipient(ctx context.Context, bankCode, accountNumber, accountName string) (string, error) {
	if bankCode == "" || accountNumber == "" || accountName == "" {
		return "", fmt.Errorf("paystack: all recipient details are required")
	}

	payload := map[string]any{
		"type":           "nuban",
		"name":           accountName,
		"account_number": accountNumber,
		"bank_code":      bankCode,
		"currency":       "NGN",
	}

	resp, err := p.makeRequest(ctx, "POST", "/transferrecipient", payload)
	if err != nil {
		return "", fmt.Errorf("paystack: create recipient failed: %w", err)
	}

	data := resp["data"].(map[string]any)
	recipientCode, ok := data["recipient_code"].(string)
	if !ok || recipientCode == "" {
		return "", fmt.Errorf("paystack: invalid recipient code in response")
	}

	return recipientCode, nil
}

// Transfer initiates a payout to a recipient
func (p *PaystackAdapter) Transfer(ctx context.Context, req PayoutRequest) (*PayoutResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("paystack: validation failed: %w", err)
	}

	payload := map[string]any{
		"source":    "balance",
		"amount":    req.Amount,
		"recipient": req.RecipientCode,
		"reference": req.Reference,
		"reason":    req.Narration,
		"currency":  req.Currency,
	}

	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	resp, err := p.makeRequest(ctx, "POST", "/transfer", payload)
	if err != nil {
		return nil, fmt.Errorf("paystack: %w", ErrTransferFailed)
	}

	data := resp["data"].(map[string]any)
	transferID, _ := data["id"].(float64)
	status, _ := data["status"].(string)
	reference, _ := data["reference"].(string)
	createdAt, _ := data["created_at"].(string)

	return &PayoutResponse{
		Success:    true,
		TransferID: fmt.Sprintf("%.0f", transferID),
		Status:     NormalizeTransferStatus(status, "paystack"),
		Amount:     req.Amount,
		Currency:   req.Currency,
		Reference:  reference,
		Message:    "Transfer initiated successfully",
		CreatedAt:  createdAt,
	}, nil
}

// VerifyTransfer checks the status of a transfer
func (p *PaystackAdapter) VerifyTransfer(ctx context.Context, reference string) (*PayoutResponse, error) {
	if err := ValidateReference(reference); err != nil {
		return nil, fmt.Errorf("paystack: %w", err)
	}

	endpoint := fmt.Sprintf("/transfer/verify/%s", reference)
	resp, err := p.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("paystack: verify transfer failed: %w", err)
	}

	data := resp["data"].(map[string]any)
	transferID, _ := data["id"].(float64)
	status, _ := data["status"].(string)
	amount, _ := data["amount"].(float64)
	currency, _ := data["currency"].(string)
	ref, _ := data["reference"].(string)
	createdAt, _ := data["created_at"].(string)

	return &PayoutResponse{
		Success:    true,
		TransferID: fmt.Sprintf("%.0f", transferID),
		Status:     NormalizeTransferStatus(status, "paystack"),
		Amount:     int64(amount),
		Currency:   Currency(currency),
		Reference:  ref,
		Message:    fmt.Sprintf("Transfer is %s", status),
		CreatedAt:  createdAt,
	}, nil
}

// ============================================================================
// WebhookHandler Implementation
// ============================================================================

// VerifySignature validates Paystack webhook signature using HMAC SHA512
func (p *PaystackAdapter) VerifySignature(headerSignature string, payload []byte) bool {
	hash := hmac.New(sha512.New, []byte(p.secretKey))
	hash.Write(payload)
	expectedSignature := hex.EncodeToString(hash.Sum(nil))
	return hmac.Equal([]byte(headerSignature), []byte(expectedSignature))
}

// ParseEvent converts Paystack webhook payload to UnifiedEvent
func (p *PaystackAdapter) ParseEvent(payload []byte) (*UnifiedEvent, error) {
	var webhook struct {
		Event string `json:"event"`
		Data  struct {
			ID                   json.RawMessage `json:"id"`
			Status               string          `json:"status"`
			Reference            string          `json:"reference"`
			TransactionReference string          `json:"transaction_reference"`
			Amount               float64         `json:"amount"`
			Currency             string          `json:"currency"`
			Metadata             json.RawMessage `json:"metadata"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("paystack: failed to parse webhook: %w", err)
	}

	reference := webhook.Data.Reference
	if reference == "" {
		reference = webhook.Data.TransactionReference
	}

	providerTxID := parsePaystackID(webhook.Data.ID)

	return &UnifiedEvent{
		Type:         webhook.Event,
		Reference:    reference,
		ProviderTxID: providerTxID,
		Status:       webhook.Data.Status,
		Amount:       int64(webhook.Data.Amount),
		Currency:     Currency(webhook.Data.Currency),
		RawData:      json.RawMessage(payload),
	}, nil
}

func parsePaystackID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString
	}

	var asNumber float64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return fmt.Sprintf("%.0f", asNumber)
	}

	return ""
}

// ============================================================================
// HTTP Client Helper
// ============================================================================

func (p *PaystackAdapter) makeRequest(ctx context.Context, method, endpoint string, payload any) (map[string]any, error) {
	url := paystackBaseURL + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if status, ok := result["status"].(bool); !ok || !status {
		message, _ := result["message"].(string)
		if message == "" {
			message = "unknown error"
		}

		// Map common Paystack errors
		lowerMsg := strings.ToLower(message)
		if strings.Contains(lowerMsg, "duplicate") {
			return nil, ErrDuplicateReference
		}
		if strings.Contains(lowerMsg, "insufficient") {
			return nil, ErrInsufficientFunds
		}
		if strings.Contains(lowerMsg, "not found") {
			return nil, ErrInvalidReference
		}

		return nil, fmt.Errorf("paystack error: %s", message)
	}

	return result, nil
}
