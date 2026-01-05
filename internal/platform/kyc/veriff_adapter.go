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
	veriffBaseURL = "https://stationapi.veriff.com"
	veriffTimeout = 60 * time.Second
)

// VeriffAdapter implements KYCProvider for Veriff
type VeriffAdapter struct {
	apiKey        string // X-AUTH-CLIENT
	secretKey     string // Used for X-HMAC-SIGNATURE (requests) and Webhook verification
	webhookSecret string // Webhook signature verification (same as secretKey for Veriff)
	httpClient    *http.Client
}

// NewVeriffAdapter creates a new Veriff adapter
// Note: Veriff uses the same Private Key (Secret) for both request signing and webhook validation.
func NewVeriffAdapter(apiKey, secretKey, webhookSecret string) *VeriffAdapter {
	return &VeriffAdapter{
		apiKey:        apiKey,
		secretKey:     secretKey,
		webhookSecret: webhookSecret, // For Veriff, this is typically the same as secretKey
		httpClient: &http.Client{
			Timeout: veriffTimeout,
		},
	}
}

// Name returns the provider identifier
func (v *VeriffAdapter) Name() string {
	return "veriff"
}

// SupportedCountries returns all countries
func (v *VeriffAdapter) SupportedCountries() []string {
	// Veriff supports global coverage (190+ countries)
	return []string{}
}

// SupportedDocuments returns document types supported by Veriff
func (v *VeriffAdapter) SupportedDocuments() []DocumentType {
	return []DocumentType{
		DocumentPassport,
		DocumentDriversLicense,
		DocumentNationalID,
		DocumentResidencePermit,
	}
}

// SubmitVerification submits verification to Veriff
func (v *VeriffAdapter) SubmitVerification(ctx context.Context, req VerificationRequest) (*VerificationResponse, error) {
	// Step 1: Create Verification Session
	// Ref: https://developers.veriff.com/#create-verification-session

	// Prepare timestamp for uniqueness if needed, or rely on Veriff
	ts := time.Now().UTC().Format(time.RFC3339)

	sessionReq := map[string]any{
		"verification": map[string]any{
			"person": map[string]any{
				"firstName": req.FirstName,
				"lastName":  req.LastName,
			},
			"document": map[string]any{
				"type":    mapDocumentTypeToVeriff(req.DocumentType),
				"country": req.Country,
			},
			"vendorData": req.Metadata,
			"timestamp":  ts,
		},
	}

	// Add DOB if present
	if req.DateOfBirth != nil {
		sessionReq["verification"].(map[string]any)["person"].(map[string]any)["dateOfBirth"] = req.DateOfBirth.Format("2006-01-02")
	}

	// Create session
	sessionData, err := v.makeRequest(ctx, "POST", "/v1/sessions", sessionReq)
	if err != nil {
		return nil, fmt.Errorf("veriff session creation failed: %w", err)
	}

	var sessionResp struct {
		Status       string `json:"status"`
		Verification struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"verification"`
	}

	if err := json.Unmarshal(sessionData, &sessionResp); err != nil {
		return nil, fmt.Errorf("failed to parse veriff session response: %w", err)
	}

	sessionID := sessionResp.Verification.ID

	// Step 2: Submit Document Image (Front)
	// Ref: https://developers.veriff.com/#submit-media
	if len(req.DocumentImage) > 0 {
		mediaReq := map[string]any{
			"image": map[string]any{
				"context": "document-front",
				"content": req.DocumentImage, // Veriff expects Base64
			},
		}
		_, err = v.makeRequest(ctx, "POST", fmt.Sprintf("/v1/sessions/%s/media", sessionID), mediaReq)
		if err != nil {
			return nil, fmt.Errorf("veriff document upload failed: %w", err)
		}
	}

	// Step 3: Submit Selfie
	if len(req.SelfieImage) > 0 {
		selfieReq := map[string]any{
			"image": map[string]any{
				"context": "face",
				"content": req.SelfieImage,
			},
		}
		_, err = v.makeRequest(ctx, "POST", fmt.Sprintf("/v1/sessions/%s/media", sessionID), selfieReq)
		if err != nil {
			return nil, fmt.Errorf("veriff selfie upload failed: %w", err)
		}
	}

	// Step 4: Submit the Session (Start Analysis)
	// This tells Veriff "uploads are done, please analyze"
	submitReq := map[string]any{
		"verification": map[string]any{
			"status":    "submitted",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}
	_, err = v.makeRequest(ctx, "PATCH", fmt.Sprintf("/v1/sessions/%s", sessionID), submitReq)
	if err != nil {
		return nil, fmt.Errorf("failed to submit session for analysis: %w", err)
	}

	// Return response
	response := &VerificationResponse{
		Success:       true,
		ProviderRef:   sessionID,
		Provider:      "veriff",
		Status:        StatusPending,
		EstimatedCost: v.getCostForCountry(req.Country),
		Message:       "Verification submitted successfully",
	}

	return response, nil
}

// ParseWebhook parses Veriff webhook payload
func (v *VeriffAdapter) ParseWebhook(payload []byte, headers map[string]string) (*WebhookEvent, error) {
	// Verify Signature
	// Ref: https://developers.veriff.com/#webhook-security
	signature := headers["x-hmac-signature"] // Veriff uses lowercase usually, but standardizes
	if signature == "" {
		signature = headers["X-Hmac-Signature"]
	}

	if !v.VerifySignature(payload, signature) {
		return nil, fmt.Errorf("invalid veriff webhook signature")
	}

	// Parse Body
	// Ref: https://developers.veriff.com/#decision-webhook
	var veriffWebhook struct {
		Action       string `json:"action"` // "decision" or "event"
		Status       string `json:"status"` // Overall status
		Verification struct {
			ID     string `json:"id"`
			Status string `json:"status"` // approved, declined, resubmission_requested
			Code   int    `json:"code"`   // Reason code
			Reason string `json:"reason"` // Reason text
			Person struct {
				FirstName   string `json:"firstName"`
				LastName    string `json:"lastName"`
				DateOfBirth string `json:"dateOfBirth"`
			} `json:"person"`
			Document struct {
				Type    string `json:"type"`
				Country string `json:"country"`
				Number  string `json:"number"`
			} `json:"document"`
		} `json:"verification"`
	}

	if err := json.Unmarshal(payload, &veriffWebhook); err != nil {
		return nil, fmt.Errorf("failed to parse veriff webhook: %w", err)
	}

	// Map Status
	status := mapVeriffStatus(veriffWebhook.Verification.Status)

	event := &WebhookEvent{
		ProviderRef: veriffWebhook.Verification.ID,
		Provider:    "veriff",
		Status:      status,
		RawPayload:  payload,
		ExtractedData: map[string]string{
			"first_name":    veriffWebhook.Verification.Person.FirstName,
			"last_name":     veriffWebhook.Verification.Person.LastName,
			"date_of_birth": veriffWebhook.Verification.Person.DateOfBirth,
			"doc_type":      veriffWebhook.Verification.Document.Type,
			"doc_country":   veriffWebhook.Verification.Document.Country,
			"doc_number":    veriffWebhook.Verification.Document.Number,
		},
	}

	// Handle Rejection/Resubmission details
	if status == StatusRejected || status == StatusResubmitted {
		reason := veriffWebhook.Verification.Reason
		// If reason is empty (sometimes true for approved), use status
		if reason == "" {
			reason = veriffWebhook.Verification.Status
		}

		failureCode := mapVeriffFailureCode(veriffWebhook.Verification.Code, reason)
		event.FailureReason = &reason
		event.FailureCode = &failureCode
	}

	return event, nil
}

// VerifySignature verifies Veriff webhook signature
func (v *VeriffAdapter) VerifySignature(payload []byte, signature string) bool {
	// Veriff signs the JSON payload using the Webhook Secret (HMAC-SHA256)
	// Ref: https://developers.veriff.com/#webhook-security
	if signature == "" {
		return false
	}

	// Use webhookSecret for webhook verification (typically same as secretKey for Veriff)
	mac := hmac.New(sha256.New, []byte(v.webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// HealthCheck verifies Veriff API is reachable
func (v *VeriffAdapter) HealthCheck(ctx context.Context) error {
	// Simple check: create a dummy session or just valid auth check.
	// Since Veriff strictly validates bodies, a GET to /v1/sessions isn't standard
	// but a signed request with bad data should return 400, not 401.
	// However, standard practice is checking IP or basic connectivity.
	// We will attempt to reach the base URL.
	req, _ := http.NewRequestWithContext(ctx, "GET", veriffBaseURL, nil)
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// As long as we get a response (even 404), the API is reachable
	return nil
}

// EstimateCost returns estimated cost
func (v *VeriffAdapter) EstimateCost(country string) (float64, error) {
	return v.getCostForCountry(country), nil
}

// Helper: makeRequest handles HTTP requests to Veriff API
func (v *VeriffAdapter) makeRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBodyBytes []byte
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBodyBytes = jsonData
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, veriffBaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AUTH-CLIENT", v.apiKey) // API Key (Public)

	// Generate Signature
	// Veriff requires signing the Request Body.
	// If body is nil, sign empty string? Docs say request body.
	// Ref: https://developers.veriff.com/#authentication
	signature := v.generateSignature(reqBodyBytes)
	req.Header.Set("X-HMAC-SIGNATURE", signature)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("veriff api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Helper: generateSignature generates HMAC signature for Veriff API
func (v *VeriffAdapter) generateSignature(body []byte) string {
	// Veriff Signature Logic: HMAC-SHA256(Secret Key, Request Body)
	// Important: The body must be the exact byte slice sent over the wire.
	mac := hmac.New(sha256.New, []byte(v.secretKey))
	if len(body) > 0 {
		mac.Write(body)
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func (v *VeriffAdapter) getCostForCountry(country string) float64 {
	switch country {
	case "US", "GB", "CA", "AU":
		return 2.50
	case "DE", "FR", "ES", "IT":
		return 2.00
	default:
		return 3.00
	}
}

func mapDocumentTypeToVeriff(docType DocumentType) string {
	switch docType {
	case DocumentPassport:
		return "PASSPORT"
	case DocumentDriversLicense:
		return "DRIVERS_LICENSE"
	case DocumentNationalID:
		return "ID_CARD"
	case DocumentResidencePermit:
		return "RESIDENCE_PERMIT"
	default:
		return "ID_CARD"
	}
}

func mapVeriffStatus(status string) VerificationStatus {
	switch status {
	case "approved":
		return StatusApproved
	case "declined":
		return StatusRejected
	case "resubmission_requested":
		return StatusResubmitted
	case "expired":
		return StatusExpired
	case "review", "submitted":
		return StatusInReview
	default:
		return StatusPending
	}
}

func mapVeriffFailureCode(code int, reason string) FailureCode {
	// Veriff Decision Codes
	// Ref: https://developers.veriff.com/#decision-codes
	switch code {
	case 101, 102: // Face not visible / not found
		return FailureNoFaceDetected
	case 103: // Face doesn't match document
		return FailureFaceMismatch
	case 200, 201: // Document unreadable/blurry
		return FailureImageBlurry
	case 205: // Document expired
		return FailureDocExpired
	case 207: // Document type not supported
		return FailureInvalidDocument
	case 900, 901: // Suspected Fraud
		return FailureSuspectedFraud
	default:
		// Fallback string matching
		r := strings.ToLower(reason)
		if strings.Contains(r, "blur") || strings.Contains(r, "quality") {
			return FailureImageBlurry
		}
		if strings.Contains(r, "match") {
			return FailureFaceMismatch
		}
		return FailureProviderError
	}
}
