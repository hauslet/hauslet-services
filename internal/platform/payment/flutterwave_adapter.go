package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hauslet/internal/platform/breaker"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	flutterwaveV4SandboxBaseURL = "https://developersandbox-api.flutterwave.com"
	flutterwaveV4LiveBaseURL    = "https://f4bexperience.flutterwave.com"
	flutterwaveV4TokenURL       = "https://idp.flutterwave.com/realms/flutterwave/protocol/openid-connect/token"
)

// FlutterwaveAdapter implements TransactionClient, PayoutClient, and WebhookHandler for Flutterwave.
//
// v4 notes:
//   - Authentication is OAuth2 access token based. Use WithOAuthCredentials for
//     automatic token fetch/refresh.
//   - If only secretKey is provided, it is treated as a static bearer token for
//     compatibility with earlier integrations.
type FlutterwaveAdapter struct {
	secretKey      string // static bearer token fallback
	webhookSecret  string
	baseURL        string
	tokenURL       string
	oauthClientID  string
	oauthSecret    string
	httpClient     *http.Client
	circuitBreaker breaker.CircuitBreaker

	mu                sync.Mutex
	cachedAccessToken string
	tokenExpiresAt    time.Time
}

// Ensure FlutterwaveAdapter implements all interfaces.
var _ TransactionClient = (*FlutterwaveAdapter)(nil)
var _ PayoutClient = (*FlutterwaveAdapter)(nil)
var _ WebhookHandler = (*FlutterwaveAdapter)(nil)

// NewFlutterwaveAdapter creates a new Flutterwave adapter.
//
// It defaults to the v4 sandbox base URL for safety in incremental migrations.
// Use WithBaseURL(flutterwaveV4LiveBaseURL) for live traffic.
func NewFlutterwaveAdapter(secretKey, webhookSecret string, cb breaker.CircuitBreaker) *FlutterwaveAdapter {
	return &FlutterwaveAdapter{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		baseURL:       flutterwaveV4SandboxBaseURL,
		tokenURL:      flutterwaveV4TokenURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		circuitBreaker: cb,
	}
}

// WithHTTPClient allows overriding the HTTP client (useful for testing).
func (f *FlutterwaveAdapter) WithHTTPClient(client *http.Client) *FlutterwaveAdapter {
	if client != nil {
		f.httpClient = client
	}
	return f
}

// WithBaseURL overrides the API base URL.
func (f *FlutterwaveAdapter) WithBaseURL(baseURL string) *FlutterwaveAdapter {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" {
		f.baseURL = strings.TrimRight(baseURL, "/")
	}
	return f
}

// WithTokenEndpoint overrides the OAuth token endpoint.
func (f *FlutterwaveAdapter) WithTokenEndpoint(tokenURL string) *FlutterwaveAdapter {
	tokenURL = strings.TrimSpace(tokenURL)
	if tokenURL != "" {
		f.tokenURL = strings.TrimRight(tokenURL, "/")
	}
	return f
}

// WithOAuthCredentials enables automatic v4 access token fetch/refresh.
func (f *FlutterwaveAdapter) WithOAuthCredentials(clientID, clientSecret string) *FlutterwaveAdapter {
	f.oauthClientID = strings.TrimSpace(clientID)
	f.oauthSecret = strings.TrimSpace(clientSecret)
	return f
}

// WithStaticAccessToken sets a static bearer token.
func (f *FlutterwaveAdapter) WithStaticAccessToken(accessToken string) *FlutterwaveAdapter {
	f.secretKey = strings.TrimSpace(accessToken)
	return f
}

// ============================================================================
// TransactionClient Implementation
// ============================================================================

// AuthorizePayment initiates a minimal card authorization charge.
//
// v4 flow:
//  1. Create (or reuse) a customer by email.
//  2. Initiate a minimal-amount charge; the response contains a redirect URL
//     or next_action that the caller must present to the user.
//  3. After the user completes authorization, verify the charge to extract
//     the payment_method_id for recurring ChargeAuthorization calls.
func (f *FlutterwaveAdapter) AuthorizePayment(ctx context.Context, req AuthorizationRequest) (*AuthorizationResponse, error) {
	if strings.TrimSpace(req.Email) == "" {
		return nil, fmt.Errorf("flutterwave: email is required for authorization")
	}
	if req.Currency == "" {
		req.Currency = "NGN"
	}

	// Step 1: Create a customer entity.
	customerID := strings.TrimSpace(req.CustomerID)
	if customerID == "" {
		custPayload := map[string]any{
			"email": strings.TrimSpace(req.Email),
		}
		custResp, err := f.makeRequest(ctx, "POST", "/customers", custPayload)
		if err != nil {
			return nil, fmt.Errorf("flutterwave: create customer failed: %w", err)
		}
		customerID = asString(toMap(custResp["data"])["id"])
		if customerID == "" {
			return nil, fmt.Errorf("flutterwave: empty customer ID in response")
		}
	}

	// Step 2: Initiate a minimal authorization charge.
	// The amount is the smallest chargeable unit (e.g. ₦1 = 100 kobo).
	minimalAmount := FromMinorUnits(100, Currency(req.Currency))
	reference := makeRequestID("flwauth")

	callbackURL := ""
	if len(req.Metadata) > 0 {
		callbackURL = req.Metadata["callback_url"]
	}

	chargePayload := map[string]any{
		"amount":      minimalAmount,
		"currency":    req.Currency,
		"customer_id": customerID,
		"reference":   reference,
	}
	if callbackURL != "" {
		chargePayload["redirect_url"] = callbackURL
	}
	if len(req.Metadata) > 0 {
		chargePayload["meta"] = req.Metadata
	}

	resp, err := f.makeRequest(ctx, "POST", "/charges", chargePayload)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: authorization charge failed: %w", err)
	}

	data := toMap(resp["data"])
	chargeID := asString(data["id"])
	respRef := firstNonEmpty(asString(data["reference"]), reference)

	// Extract redirect URL from v4 next_action.
	nextAction := toMap(data["next_action"])
	authURL := asString(toMap(nextAction["redirect_url"])["url"])

	return &AuthorizationResponse{
		AuthorizationCode: chargeID,
		Reusable:          false, // becomes reusable after user completes & verify extracts payment_method_id
		Provider:          "flutterwave",
		Raw: map[string]any{
			"authorization_url": authURL,
			"charge_id":         chargeID,
			"customer_id":       customerID,
			"reference":         respRef,
			"next_action":       data["next_action"],
		},
	}, nil
}

// Initialize initiates a charge via Flutterwave v4.
//
// It supports two modes, chosen automatically based on what the caller provides:
//
// 1. First-time / one-off payment (Paystack-like simplicity):
//   - Just pass amount, currency, email, reference, and callback URL.
//   - Uses the orchestrator endpoint (POST /orchestration/direct-charges) which
//     creates customer + payment method + charge in a single call.
//   - Returns a redirect URL for the user to complete card entry / 3DS / OTP.
//
// 2. Returning customer with saved payment method:
//   - Pass customer_id and payment_method_id in req.Metadata.
//   - Uses the general flow endpoint (POST /charges) for a direct charge.
//   - Faster path for repeat payments with stored credentials.
//
// After a successful first-time charge, the verify response includes
// payment_method_details.id and customer_id — store these for future
// ChargeAuthorization calls.
func (f *FlutterwaveAdapter) Initialize(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("flutterwave: validation failed: %w", err)
	}

	customerID := ""
	paymentMethodID := ""
	if len(req.Metadata) > 0 {
		customerID = req.Metadata["customer_id"]
		paymentMethodID = req.Metadata["payment_method_id"]
	}

	var resp map[string]any
	var err error

	if customerID != "" && paymentMethodID != "" {
		// Returning customer: direct charge via General Flow.
		resp, err = f.initializeGeneralFlow(ctx, req, customerID, paymentMethodID)
	} else {
		// First-time / one-off: use orchestrator.
		resp, err = f.initializeOrchestrator(ctx, req)
	}
	if err != nil {
		return nil, err
	}

	return f.buildPaymentResponse(resp, req), nil
}

// initializeOrchestrator creates a charge using the orchestrator endpoint.
// This combines customer creation, payment method setup, and charge initiation
// into a single API call — no pre-existing IDs required.
func (f *FlutterwaveAdapter) initializeOrchestrator(ctx context.Context, req PaymentRequest) (map[string]any, error) {
	payload := map[string]any{
		"amount":       FromMinorUnits(req.Amount, req.Currency),
		"currency":     req.Currency.String(),
		"reference":    req.Reference,
		"redirect_url": req.CallbackURL,
		"customer": map[string]any{
			"email": req.Email,
		},
		"payment_method": map[string]any{
			"type": "card",
			"card": map[string]any{},
		},
	}
	if len(req.Metadata) > 0 {
		payload["meta"] = req.Metadata
	}

	resp, err := f.makeRequest(ctx, "POST", "/orchestration/direct-charges", payload)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: orchestrator charge failed: %w", err)
	}
	return resp, nil
}

// initializeGeneralFlow creates a charge using pre-existing customer and
// payment method IDs via the standard /charges endpoint.
func (f *FlutterwaveAdapter) initializeGeneralFlow(ctx context.Context, req PaymentRequest, customerID, paymentMethodID string) (map[string]any, error) {
	payload := map[string]any{
		"reference":         req.Reference,
		"amount":            FromMinorUnits(req.Amount, req.Currency),
		"currency":          req.Currency.String(),
		"customer_id":       customerID,
		"payment_method_id": paymentMethodID,
		"redirect_url":      req.CallbackURL,
	}
	if len(req.Metadata) > 0 {
		payload["meta"] = req.Metadata
	}

	resp, err := f.makeRequest(ctx, "POST", "/charges", payload)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: charge failed: %w", err)
	}
	return resp, nil
}

// buildPaymentResponse converts a raw Flutterwave charge/orchestrator response
// into a unified PaymentResponse.
func (f *FlutterwaveAdapter) buildPaymentResponse(resp map[string]any, req PaymentRequest) *PaymentResponse {
	data := toMap(resp["data"])
	status := asString(data["status"])
	normalizedStatus := normalizeFlutterwaveStatus(status)

	nextAction := toMap(data["next_action"])
	actionURL := asString(toMap(nextAction["redirect_url"])["url"])

	currency := Currency(strings.ToUpper(firstNonEmpty(asString(data["currency"]), req.Currency.String())))
	amountMajor := asFloat64(data["amount"])
	if amountMajor == 0 {
		amountMajor = FromMinorUnits(req.Amount, req.Currency)
	}

	ref := firstNonEmpty(asString(data["reference"]), req.Reference)

	return &PaymentResponse{
		Success:       normalizedStatus == StatusSuccess || normalizedStatus == StatusPending,
		Status:        normalizedStatus,
		TransactionID: asString(data["id"]),
		Reference:     ref,
		RedirectURL:   actionURL,
		RequiresAction: normalizedStatus == StatusPending ||
			actionURL != "",
		ActionPayload: map[string]any{
			"redirect_url": actionURL,
			"next_action":  nextAction,
			"customer_id":  asString(data["customer_id"]),
		},
		Amount:   ToMinorUnits(amountMajor, currency),
		Currency: currency,
		Message:  firstNonEmpty(asString(resp["message"]), "Charge initiated"),
	}
}

// ChargeAuthorization charges a saved payment method using the v4 /charges
// endpoint with the recurring flag.
//
// Pass the payment_method_id as AuthToken and include customer_id in
// req.Metadata["customer_id"].
func (f *FlutterwaveAdapter) ChargeAuthorization(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("flutterwave: validation failed: %w", err)
	}
	if req.AuthToken == nil || strings.TrimSpace(*req.AuthToken) == "" {
		return nil, fmt.Errorf("flutterwave: payment_method_id (AuthToken) is required")
	}

	customerID := ""
	if len(req.Metadata) > 0 {
		customerID = req.Metadata["customer_id"]
	}
	if customerID == "" {
		return nil, fmt.Errorf("flutterwave: customer_id is required in metadata")
	}

	payload := map[string]any{
		"payment_method_id": strings.TrimSpace(*req.AuthToken),
		"customer_id":       customerID,
		"amount":            FromMinorUnits(req.Amount, req.Currency),
		"currency":          req.Currency.String(),
		"reference":         req.Reference,
		"recurring":         true,
	}
	if len(req.Metadata) > 0 {
		payload["meta"] = req.Metadata
	}

	resp, err := f.makeRequest(ctx, "POST", "/charges", payload)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: %w", ErrAuthorizationFailed)
	}

	data := toMap(resp["data"])
	status := normalizeFlutterwaveStatus(asString(data["status"]))
	currency := Currency(strings.ToUpper(firstNonEmpty(asString(data["currency"]), req.Currency.String())))
	amountMajor := asFloat64(data["amount"])
	if amountMajor == 0 {
		amountMajor = FromMinorUnits(req.Amount, req.Currency)
	}

	return &PaymentResponse{
		Success:        status == StatusSuccess,
		Status:         status,
		TransactionID:  asString(data["id"]),
		Reference:      firstNonEmpty(asString(data["reference"]), req.Reference),
		RequiresAction: status == StatusPending,
		Amount:         ToMinorUnits(amountMajor, currency),
		Currency:       currency,
		Message:        firstNonEmpty(asString(resp["message"]), "Charge initiated"),
	}, nil
}

// Verify checks the status of a charge.
//
// It accepts either:
// - provider charge ID (preferred), or
// - internal reference (lookup via list endpoint fallback).
func (f *FlutterwaveAdapter) Verify(ctx context.Context, reference string) (*PaymentResponse, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, fmt.Errorf("flutterwave: reference is required")
	}

	charge, err := f.findCharge(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: verify failed: %w", err)
	}

	status := normalizeFlutterwaveStatus(asString(charge["status"]))

	currency := Currency(strings.ToUpper(asString(charge["currency"])))
	if currency == "" {
		currency = NGN
	}

	amountMajor := asFloat64(charge["amount"])

	return &PaymentResponse{
		Success:       status == StatusSuccess,
		Status:        status,
		TransactionID: asString(charge["id"]),
		Reference:     firstNonEmpty(asString(charge["reference"]), reference),
		Amount:        ToMinorUnits(amountMajor, currency),
		Currency:      currency,
		Message:       fmt.Sprintf("Charge is %s", asString(charge["status"])),
	}, nil
}

// Refund initiates a refund for a completed charge.
// NOTE: v4 create-refund expects a charge identifier.
func (f *FlutterwaveAdapter) Refund(ctx context.Context, originalTxID string, amount int64, reason string) (*RefundResponse, error) {
	if strings.TrimSpace(originalTxID) == "" {
		return nil, fmt.Errorf("flutterwave: charge ID is required")
	}

	payload := map[string]any{
		"charge_id": originalTxID,
	}
	if amount > 0 {
		payload["amount"] = float64(amount) / 100.0
	}
	if reason != "" {
		payload["reason"] = reason
	}

	resp, err := f.makeRequest(ctx, "POST", "/refunds", payload)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: %w", ErrRefundFailed)
	}

	data := toMap(resp["data"])
	currency := Currency(strings.ToUpper(asString(data["currency"])))
	if currency == "" {
		currency = NGN
	}
	amountMajor := asFloat64(data["amount"])
	if amountMajor == 0 && amount > 0 {
		amountMajor = float64(amount) / 100.0
	}

	return &RefundResponse{
		Success:   true,
		RefundID:  asString(data["id"]),
		Status:    normalizeFlutterwaveStatus(asString(data["status"])),
		Amount:    ToMinorUnits(amountMajor, currency),
		Currency:  currency,
		Message:   firstNonEmpty(asString(resp["message"]), "Refund initiated"),
		CreatedAt: asString(data["created_datetime"]),
	}, nil
}

// ============================================================================
// PayoutClient Implementation
// ============================================================================

// ValidateAccount resolves and validates a bank account.
func (f *FlutterwaveAdapter) ValidateAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	if strings.TrimSpace(bankCode) == "" || strings.TrimSpace(accountNumber) == "" {
		return "", fmt.Errorf("flutterwave: bank code and account number are required")
	}

	payload := map[string]any{
		"account_bank":   strings.TrimSpace(bankCode),
		"account_number": strings.TrimSpace(accountNumber),
	}

	resp, err := f.makeRequest(ctx, "POST", "/banks/account-resolve", payload)
	if err != nil {
		return "", fmt.Errorf("flutterwave: %w", ErrInvalidAccount)
	}

	data := toMap(resp["data"])
	name := asString(data["account_name"])
	if name == "" {
		return "", ErrInvalidAccount
	}
	return name, nil
}

// CreateRecipient creates a transfer recipient and returns recipient ID.
// If recipient creation fails due corridor-specific required fields, it returns
// a local compatibility code so downstream transfer can still run in fallback mode.
func (f *FlutterwaveAdapter) CreateRecipient(ctx context.Context, bankCode, accountNumber, accountName string) (string, error) {
	if strings.TrimSpace(bankCode) == "" || strings.TrimSpace(accountNumber) == "" {
		return "", fmt.Errorf("flutterwave: bank code and account number are required")
	}

	payload := map[string]any{
		"type": "bank_ngn",
		"bank": map[string]any{
			"account_number": strings.TrimSpace(accountNumber),
			"code":           strings.TrimSpace(bankCode),
		},
	}

	resp, err := f.makeRequest(ctx, "POST", "/transfers/recipients", payload)
	if err != nil {
		return "", fmt.Errorf("flutterwave: recipient create failed: %w", err)
	}

	recipientID := asString(toMap(resp["data"])["id"])
	if recipientID == "" {
		return "", fmt.Errorf("flutterwave: empty recipient ID in response")
	}
	return recipientID, nil
}

// Transfer initiates a payout to a recipient.
func (f *FlutterwaveAdapter) Transfer(ctx context.Context, req PayoutRequest) (*PayoutResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("flutterwave: validation failed: %w", err)
	}

	// v4 recipient-id flow
	payload := map[string]any{
		"action":    "instant",
		"reference": req.Reference,
		"narration": req.Narration,
		"meta":      req.Metadata,
		"payment_instruction": map[string]any{
			"source_currency":      req.Currency.String(),
			"destination_currency": req.Currency.String(),
			"recipient_id":         req.RecipientCode,
			"amount": map[string]any{
				"value":      FromMinorUnits(req.Amount, req.Currency),
				"applies_to": "destination_currency",
			},
		},
	}

	resp, err := f.makeRequest(ctx, "POST", "/transfers", payload)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: %w", ErrTransferFailed)
	}

	data := toMap(resp["data"])
	return &PayoutResponse{
		Success:    true,
		TransferID: asString(data["id"]),
		Status:     normalizeFlutterwaveTransferStatus(asString(data["status"])),
		Amount:     req.Amount,
		Currency:   req.Currency,
		Reference:  firstNonEmpty(asString(data["reference"]), req.Reference),
		Message:    firstNonEmpty(asString(resp["message"]), "Transfer created"),
		CreatedAt:  asString(data["created_datetime"]),
	}, nil
}

// VerifyTransfer checks the status of a transfer.
//
// It accepts either transfer ID (preferred) or reference (best-effort lookup).
func (f *FlutterwaveAdapter) VerifyTransfer(ctx context.Context, reference string) (*PayoutResponse, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, fmt.Errorf("flutterwave: transfer reference is required")
	}

	data, err := f.findTransfer(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: verify transfer failed: %w", err)
	}

	currency := Currency(strings.ToUpper(asString(data["currency"])))
	if currency == "" {
		currency = NGN
	}
	amount := ToMinorUnits(asFloat64(data["amount"]), currency)

	return &PayoutResponse{
		Success:    true,
		TransferID: firstNonEmpty(asString(data["id"]), reference),
		Status:     normalizeFlutterwaveTransferStatus(asString(data["status"])),
		Amount:     amount,
		Currency:   currency,
		Reference:  asString(data["reference"]),
		Message:    fmt.Sprintf("Transfer is %s", asString(data["status"])),
		CreatedAt:  asString(data["created_datetime"]),
	}, nil
}

// ListBanks returns banks supported by Flutterwave.
func (f *FlutterwaveAdapter) ListBanks(ctx context.Context, currency Currency, country string) ([]Bank, error) {
	query := url.Values{}
	if cc := normalizeCountryCode(country, currency); cc != "" {
		query.Set("country", cc)
	}
	if currency != "" {
		query.Set("currency", currency.String())
	}

	endpoint := "/banks"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	resp, err := f.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: list banks failed: %w", err)
	}

	rawBanks, ok := resp["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("flutterwave: invalid bank list response")
	}

	banks := make([]Bank, 0, len(rawBanks))
	for _, item := range rawBanks {
		entry := toMap(item)
		banks = append(banks, Bank{
			Name:      asString(entry["name"]),
			Code:      asString(entry["code"]),
			Country:   normalizeCountryCode(country, currency),
			Currency:  currency,
			Type:      "bank",
			Active:    true,
			IsDeleted: false,
		})
	}

	return banks, nil
}

// ============================================================================
// WebhookHandler Implementation
// ============================================================================

// VerifySignature validates Flutterwave webhook signature.
//
// v4 docs indicate the header is `flutterwave-signature` and signature is
// HMAC-SHA256(rawBody, secret_hash) encoded in base64.
// Some examples also compare signature directly to secret hash, so we support
// both to avoid dropping legitimate callbacks during migration.
func (f *FlutterwaveAdapter) VerifySignature(headerSignature string, payload []byte) bool {
	sig := strings.TrimSpace(headerSignature)
	secret := strings.TrimSpace(f.webhookSecret)
	if sig == "" || secret == "" {
		return false
	}

	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(payload)
	expectedB64 := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	if hmac.Equal([]byte(sig), []byte(expectedB64)) {
		return true
	}

	return hmac.Equal([]byte(sig), []byte(secret))
}

// ParseEvent converts Flutterwave webhook payload to UnifiedEvent.
func (f *FlutterwaveAdapter) ParseEvent(payload []byte) (*UnifiedEvent, error) {
	var webhook map[string]any
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("flutterwave: failed to parse webhook: %w", err)
	}

	data := toMap(webhook["data"])
	currency := Currency(strings.ToUpper(asString(data["currency"])))
	if currency == "" {
		currency = NGN
	}

	return &UnifiedEvent{
		Type:         asString(webhook["type"]),
		Reference:    asString(data["reference"]),
		ProviderTxID: asString(data["id"]),
		Status:       asString(data["status"]),
		Amount:       ToMinorUnits(asFloat64(data["amount"]), currency),
		Currency:     currency,
		RawData:      json.RawMessage(payload),
	}, nil
}

// ============================================================================
// Internal helpers
// ============================================================================

func (f *FlutterwaveAdapter) findCharge(ctx context.Context, reference string) (map[string]any, error) {
	if looksLikeProviderID(reference, "chg_") {
		resp, err := f.makeRequest(ctx, "GET", "/charges/"+url.PathEscape(reference), nil)
		if err == nil {
			return toMap(resp["data"]), nil
		}
	}

	listURL := "/charges?reference=" + url.QueryEscape(reference) + "&page=1&size=1"
	resp, err := f.makeRequest(ctx, "GET", listURL, nil)
	if err != nil {
		return nil, err
	}
	if first := firstData(resp); len(first) > 0 {
		return first, nil
	}

	return nil, ErrInvalidReference
}

func (f *FlutterwaveAdapter) findTransfer(ctx context.Context, reference string) (map[string]any, error) {
	if looksLikeProviderID(reference, "trf_") {
		resp, err := f.makeRequest(ctx, "GET", "/transfers/"+url.PathEscape(reference), nil)
		if err == nil {
			return toMap(resp["data"]), nil
		}
	}

	// Try by ID/path first anyway.
	resp, err := f.makeRequest(ctx, "GET", "/transfers/"+url.PathEscape(reference), nil)
	if err == nil {
		return toMap(resp["data"]), nil
	}

	// v4: list transfers filtered by reference.
	listURL := "/transfers?reference=" + url.QueryEscape(reference) + "&page=1&size=1"
	resp, err = f.makeRequest(ctx, "GET", listURL, nil)
	if err == nil {
		if first := firstData(resp); len(first) > 0 {
			return first, nil
		}
	}

	return nil, ErrInvalidReference
}

func firstData(resp map[string]any) map[string]any {
	data := resp["data"]
	if m := toMap(data); len(m) > 0 {
		// Handles wrappers like {data:{items:[...]}}
		if items, ok := m["items"].([]any); ok && len(items) > 0 {
			return toMap(items[0])
		}
		return m
	}

	if list, ok := data.([]any); ok && len(list) > 0 {
		return toMap(list[0])
	}

	return nil
}

func (f *FlutterwaveAdapter) makeRequest(ctx context.Context, method, endpoint string, payload any) (map[string]any, error) {
	provider := "flutterwave"

	if f.circuitBreaker != nil {
		allowed, err := f.circuitBreaker.AllowRequest(ctx, provider)
		if err != nil {
			// Ignore breaker errors and continue request path.
		}
		if !allowed {
			return nil, fmt.Errorf("%w: circuit is open", ErrProviderUnavailable)
		}
	}

	token, err := f.accessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("flutterwave: auth failed: %w", err)
	}

	apiURL := strings.TrimRight(f.baseURL, "/") + endpoint

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", makeRequestID("flw-trace"))
	if method != http.MethodGet {
		req.Header.Set("X-Idempotency-Key", makeRequestID("flw-idem"))
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		if f.circuitBreaker != nil {
			_ = f.circuitBreaker.RecordFailure(ctx, provider)
		}
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		if f.circuitBreaker != nil {
			_ = f.circuitBreaker.RecordFailure(ctx, provider)
		}
	} else {
		if f.circuitBreaker != nil {
			_ = f.circuitBreaker.RecordSuccess(ctx, provider)
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := strings.ToLower(strings.TrimSpace(asString(result["status"])))
	if resp.StatusCode >= 400 || (status != "" && status != "success") {
		message := asString(result["message"])
		if message == "" {
			message = "unknown error"
		}

		lowerMsg := strings.ToLower(message)
		switch {
		case strings.Contains(lowerMsg, "duplicate"):
			return nil, ErrDuplicateReference
		case strings.Contains(lowerMsg, "insufficient"):
			return nil, ErrInsufficientFunds
		case strings.Contains(lowerMsg, "not found"):
			return nil, ErrInvalidReference
		case strings.Contains(lowerMsg, "invalid account"):
			return nil, ErrInvalidAccount
		}

		return nil, fmt.Errorf("flutterwave error: %s", message)
	}

	return result, nil
}

func (f *FlutterwaveAdapter) accessToken(ctx context.Context) (string, error) {
	if strings.TrimSpace(f.oauthClientID) == "" || strings.TrimSpace(f.oauthSecret) == "" {
		if strings.TrimSpace(f.secretKey) == "" {
			return "", fmt.Errorf("no OAuth credentials or static bearer token configured")
		}
		return strings.TrimSpace(f.secretKey), nil
	}

	f.mu.Lock()
	if f.cachedAccessToken != "" && time.Now().Add(60*time.Second).Before(f.tokenExpiresAt) {
		tok := f.cachedAccessToken
		f.mu.Unlock()
		return tok, nil
	}
	f.mu.Unlock()

	form := url.Values{}
	form.Set("client_id", f.oauthClientID)
	form.Set("client_secret", f.oauthSecret)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("missing access_token in token response")
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 600
	}

	f.mu.Lock()
	f.cachedAccessToken = tokenResp.AccessToken
	f.tokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	tok := f.cachedAccessToken
	f.mu.Unlock()

	return tok, nil
}

func normalizeFlutterwaveStatus(status string) TransactionStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "successful", "success", "completed", "succeeded":
		return StatusSuccess
	case "failed", "cancelled", "canceled", "error":
		return StatusFailed
	case "pending", "processing", "new":
		return StatusPending
	default:
		return StatusFailed
	}
}

func normalizeFlutterwaveTransferStatus(status string) TransferStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "successful", "success", "completed", "succeeded":
		return TransferSuccess
	case "failed", "error", "reversed", "cancelled", "canceled":
		return TransferFailed
	case "pending", "processing", "queued", "new":
		return TransferPending
	default:
		return TransferFailed
	}
}

func normalizeCountryCode(country string, currency Currency) string {
	c := strings.TrimSpace(strings.ToUpper(country))
	switch c {
	case "NIGERIA":
		return "NG"
	case "GHANA":
		return "GH"
	}
	if len(c) == 2 {
		return c
	}
	switch currency {
	case GHS:
		return "GH"
	default:
		return "NG"
	}
}

func makeRequestID(prefix string) string {
	// v4 requires alphanumeric ASCII only (12-255 chars) for X-Trace-Id
	// and X-Idempotency-Key headers.
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

func looksLikeProviderID(val, prefix string) bool {
	if strings.HasPrefix(val, prefix) {
		return true
	}
	_, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
	return err == nil
}

func toMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case float32:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	default:
		return ""
	}
}

func asFloat64(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case int32:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
