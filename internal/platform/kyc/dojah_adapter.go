package kyc

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	dojahBaseURL = "https://api.dojah.io"
	dojahTimeout = 60 * time.Second
	// DojahWebhookIP is the whitelisted IP from Dojah documentation
	DojahWebhookIP = "20.112.64.208"
)

// Ensure these are defined in your common package, or uncomment here if standalone
/*
const (
    DocumentNIN DocumentType = "NIN"
    DocumentBVN DocumentType = "BVN"
    DocumentVIN DocumentType = "VIN"
)
*/

// DojahAdapter implements KYCProvider for Dojah
type DojahAdapter struct {
	appID         string
	secretKey     string
	webhookSecret string
	httpClient    *http.Client
}

// NewDojahAdapter creates a new Dojah adapter
func NewDojahAdapter(apiKey, secretKey, webhookSecret string) *DojahAdapter {
	return &DojahAdapter{
		appID:         apiKey,    // Dojah AppID
		secretKey:     secretKey, // Dojah Private Key (prod_sk_...)
		webhookSecret: webhookSecret,
		httpClient: &http.Client{
			Timeout: dojahTimeout,
		},
	}
}

// Name returns the provider identifier
func (d *DojahAdapter) Name() string {
	return "dojah"
}

// SupportedCountries returns countries supported by Dojah
func (d *DojahAdapter) SupportedCountries() []string {
	return []string{
		"NG", "GH", "KE", "ZA", "UG", "RW", "UK", "US",
	}
}

// SupportedDocuments returns document types supported by Dojah
func (d *DojahAdapter) SupportedDocuments() []DocumentType {
	return []DocumentType{
		DocumentPassport,
		DocumentDriversLicense,
		DocumentNationalID,
		DocumentVotersCard,
		DocumentNIN, // Added
		DocumentBVN, // Added
		DocumentVIN, // Added
	}
}

// SubmitVerification submits verification to Dojah
func (d *DojahAdapter) SubmitVerification(ctx context.Context, req VerificationRequest) (*VerificationResponse, error) {
	// Route based on available data: PhotoID Verify vs Data Lookup
	if len(req.DocumentImage) > 0 && len(req.SelfieImage) > 0 {
		return d.verifyPhotoID(ctx, req)
	} else if req.DocumentNumber != "" {
		return d.verifyGovtData(ctx, req)
	}
	return nil, fmt.Errorf("invalid request: requires either (DocumentImage + Selfie) or DocumentNumber")
}

// verifyPhotoID handles Biometric KYC (Face Match + ID)
func (d *DojahAdapter) verifyPhotoID(ctx context.Context, req VerificationRequest) (*VerificationResponse, error) {
	payload := map[string]any{
		"selfie_image":  req.SelfieImage,
		"photoid_image": req.DocumentImage,
		"id_type":       mapDocumentTypeToDojah(req.DocumentType),
	}

	respData, err := d.makeRequest(ctx, "POST", "/api/v1/kyc/photoid/verify", payload)
	if err != nil {
		return nil, err
	}

	var dojahResp struct {
		Entity struct {
			SelfieVerification struct {
				Match           bool    `json:"match"`
				ConfidenceValue float64 `json:"confidence_value"`
			} `json:"selfie_verification"`
			DocumentVerification struct {
				Status string `json:"status"`
			} `json:"document_verification"`
			ReferenceID string `json:"reference_id"`
		} `json:"entity"`
	}

	if err := json.Unmarshal(respData, &dojahResp); err != nil {
		return nil, fmt.Errorf("failed to parse dojah response: %w", err)
	}

	// Determine status
	status := StatusRejected
	if dojahResp.Entity.SelfieVerification.Match && dojahResp.Entity.DocumentVerification.Status == "valid" {
		status = StatusApproved
	}

	response := &VerificationResponse{
		Success:       status == StatusApproved,
		ProviderRef:   dojahResp.Entity.ReferenceID,
		Provider:      "dojah",
		Status:        status,
		QualityScore:  &dojahResp.Entity.SelfieVerification.ConfidenceValue,
		EstimatedCost: d.getCostForCountry(req.Country),
		Message:       "Verification completed",
	}

	if status == StatusRejected {
		reason := "Verification failed"
		code := FailureProviderError

		if !dojahResp.Entity.SelfieVerification.Match {
			reason = "Face match failed"
			code = FailureFaceMismatch
		} else if dojahResp.Entity.DocumentVerification.Status != "valid" {
			reason = "Invalid document"
			code = FailureInvalidDocument
		}
		response.FailureReason = &reason
		response.FailureCode = &code
	}

	return response, nil
}

// verifyGovtData handles direct DB lookups (NIN, BVN, VIN, DL)
func (d *DojahAdapter) verifyGovtData(ctx context.Context, req VerificationRequest) (*VerificationResponse, error) {
	var endpoint, paramKey string

	switch req.DocumentType {
	case DocumentNIN:
		endpoint = "/api/v1/kyc/nin"
		paramKey = "nin"
	case DocumentBVN:
		endpoint = "/api/v1/kyc/bvn/full"
		paramKey = "bvn"
	case DocumentVIN, DocumentVotersCard: // Support both types for VIN lookup
		endpoint = "/api/v1/kyc/vin"
		paramKey = "vin"
	case DocumentDriversLicense:
		endpoint = "/api/v1/kyc/dl"
		paramKey = "license_number"
	default:
		return nil, fmt.Errorf("unsupported document type for data lookup: %v", req.DocumentType)
	}

	// Construct URL with query params
	fullPath := fmt.Sprintf("%s?%s=%s", endpoint, paramKey, req.DocumentNumber)

	respData, err := d.makeRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, err
	}

	// Generic lookup response wrapper
	var lookupResp struct {
		Entity any `json:"entity"`
	}
	if err := json.Unmarshal(respData, &lookupResp); err != nil {
		return nil, fmt.Errorf("failed to parse lookup response: %w", err)
	}

	success := lookupResp.Entity != nil
	status := StatusRejected
	if success {
		status = StatusApproved
	}

	return &VerificationResponse{
		Success:       success,
		ProviderRef:   fmt.Sprintf("%s-%s", req.DocumentType, req.DocumentNumber),
		Provider:      "dojah",
		Status:        status,
		EstimatedCost: d.getCostForCountry(req.Country),
		Message:       "Govt data lookup completed",
	}, nil
}

// ParseWebhook parses Dojah webhook payload
func (d *DojahAdapter) ParseWebhook(payload []byte, headers map[string]string) (*WebhookEvent, error) {
	// 1. Extract Signature
	signature := headers["x-dojah-signature"]
	if signature == "" {
		signature = headers["x-dojah-signature-v2"]
	}

	// 2. Verify Signature
	if !d.VerifySignature(payload, signature) {
		return nil, fmt.Errorf("invalid dojah webhook signature")
	}

	// 3. Parse Body
	var dojahWebhook struct {
		EventName string          `json:"event"`
		Data      json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(payload, &dojahWebhook); err != nil {
		return nil, fmt.Errorf("failed to parse dojah webhook body: %w", err)
	}

	// Extract inner data
	var eventData struct {
		ReferenceID string `json:"reference_id"`
		Status      string `json:"status"`
		Message     string `json:"message"`
	}
	if len(dojahWebhook.Data) > 0 {
		_ = json.Unmarshal(dojahWebhook.Data, &eventData)
	}

	// 4. Map Status
	status := StatusPending
	eventName := strings.ToUpper(dojahWebhook.EventName)

	if strings.Contains(eventName, "SUCCESS") || strings.Contains(eventName, "VERIFIED") {
		status = StatusApproved
	} else if strings.Contains(eventName, "FAILED") || strings.Contains(eventName, "REJECTED") {
		status = StatusRejected
	}

	event := &WebhookEvent{
		ProviderRef: eventData.ReferenceID,
		Provider:    "dojah",
		Status:      status,
		RawPayload:  payload,
	}

	if status == StatusRejected {
		reason := eventData.Message
		code := mapDojahFailureCode(reason)
		event.FailureReason = &reason
		event.FailureCode = &code
	}

	return event, nil
}

// VerifySignature verifies Dojah webhook signature
func (d *DojahAdapter) VerifySignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Method 1: Check V1 (HMAC SHA256 of payload with secret)
	mac := hmac.New(sha256.New, []byte(d.secretKey))
	mac.Write(payload)
	v1Hash := hex.EncodeToString(mac.Sum(nil))

	if hmac.Equal([]byte(signature), []byte(v1Hash)) {
		return true
	}

	// Method 2: Check V2 (SHA256 of secret key only)
	hash := sha256.Sum256([]byte(d.secretKey))
	v2Hash := hex.EncodeToString(hash[:])

	return hmac.Equal([]byte(signature), []byte(v2Hash))
}

// HealthCheck verifies Dojah API is reachable
func (d *DojahAdapter) HealthCheck(ctx context.Context) error {
	_, err := d.makeRequest(ctx, "GET", "/api/v1/general/banks", nil)
	return err
}

// EstimateCost returns estimated cost
func (d *DojahAdapter) EstimateCost(country string) (float64, error) {
	return d.getCostForCountry(country), nil
}

// Helper: makeRequest handles HTTP requests
func (d *DojahAdapter) makeRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := path
	if !strings.HasPrefix(path, "http") {
		url = dojahBaseURL + path
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", d.secretKey)
	req.Header.Set("AppId", d.appID)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("dojah api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (d *DojahAdapter) getCostForCountry(country string) float64 {
	switch country {
	case "NG":
		return 0.15
	case "GH", "KE", "ZA":
		return 0.50
	default:
		return 1.00
	}
}

// mapDocumentTypeToDojah maps types for PHOTO ID Verification
func mapDocumentTypeToDojah(docType DocumentType) string {
	switch docType {
	case DocumentPassport:
		return "passport"
	case DocumentDriversLicense:
		return "drivers_license"
	case DocumentNationalID, DocumentNIN:
		return "national_id" // NIN often treated as National ID in photo context
	case DocumentVotersCard, DocumentVIN:
		return "voters_card" // VIN maps to Voters Card
	default:
		return "national_id"
	}
}

func mapDojahFailureCode(msg string) FailureCode {
	msg = strings.ToLower(msg)
	switch {
	case strings.Contains(msg, "blurry"):
		return FailureImageBlurry
	case strings.Contains(msg, "face") && strings.Contains(msg, "match"):
		return FailureFaceMismatch
	case strings.Contains(msg, "expired"):
		return FailureDocExpired
	case strings.Contains(msg, "not found"):
		return FailureDocNotFound
	default:
		return FailureProviderError
	}
}
