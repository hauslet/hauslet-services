package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	termiiTimeout = 30 * time.Second
)

// TermiiAdapter implements Provider for Termii
type TermiiAdapter struct {
	apiKey     string
	baseURL    string
	senderID   string
	httpClient *http.Client
}

// NewTermiiAdapter creates a new Termii adapter
func NewTermiiAdapter(apiKey, senderID, baseURL string) *TermiiAdapter {
	return &TermiiAdapter{
		apiKey:   apiKey,
		senderID: senderID,
		baseURL:  baseURL,
		httpClient: &http.Client{
			Timeout: termiiTimeout,
		},
	}
}

// Provider returns the provider name
func (t *TermiiAdapter) Provider() string {
	return "termii"
}

// SupportedCountries returns the list of countries Termii specializes in.
// Note: Termii supports global delivery (190+ countries), but these are their core African markets.
func (t *TermiiAdapter) SupportedCountries() []string {
	return []string{
		"NG", // Nigeria (Home Market)
		"GH", // Ghana
		"KE", // Kenya
		"ZA", // South Africa
		"UG", // Uganda
		"BJ", // Benin
		"TG", // Togo
		"US", // United States (International Route)
		"GB", // United Kingdom (International Route)
	}
}

// Send sends an SMS via Termii
func (t *TermiiAdapter) Send(ctx context.Context, req SMSRequest) (*SMSResponse, error) {
	// 1. Determine Channel
	// Termii has 'dnd' (for transactional/OTP) and 'generic' (for marketing).
	// We default to 'generic' unless it looks like an OTP/critical message.
	channel := "generic"
	if isTransactionalMessage(req.Message) {
		channel = "dnd"
	}

	// 2. Prepare Termii API request
	// Reference: https://developers.termii.com/messaging
	termiiReq := map[string]any{
		"to":      req.To,
		"from":    t.getSenderID(req.From),
		"sms":     req.Message,
		"type":    "plain",
		"channel": channel,
		"api_key": t.apiKey,
	}

	// 3. Make HTTP request
	respData, err := t.makeRequest(ctx, "POST", "/api/sms/send", termiiReq)
	if err != nil {
		return nil, fmt.Errorf("termii api request failed: %w", err)
	}

	// 4. Parse Termii response
	// Termii responses can vary slightly on failure, but successful sends look like this:
	var termiiResp struct {
		MessageID string `json:"message_id"`
		Message   string `json:"message"` // e.g., "Successfully Sent"
		Balance   any    `json:"balance"` // Can be float or string depending on account type
		User      string `json:"user"`
	}

	if err := json.Unmarshal(respData, &termiiResp); err != nil {
		return nil, fmt.Errorf("failed to parse termii response: %w", err)
	}

	// Termii returns 200 OK even for some logic errors, so check MessageID
	if termiiResp.MessageID == "" {
		// If ID is missing, treat the 'message' field as the error description
		return nil, fmt.Errorf("termii error: %s", termiiResp.Message)
	}

	// 5. Calculate Metrics
	segments := calculateSMSSegments(req.Message)
	cost := t.estimateCost(req.To, segments)

	return &SMSResponse{
		Success:   true,
		MessageID: termiiResp.MessageID,
		Status:    "sent",
		Provider:  "termii",
		Cost:      cost,
		Segments:  segments,
		Message:   termiiResp.Message,
	}, nil
}

// HealthCheck verifies Termii API is reachable and Key is valid
func (t *TermiiAdapter) HealthCheck(ctx context.Context) error {
	// The /api/get-balance endpoint is a lightweight way to verify credentials
	_, err := t.makeRequest(ctx, "GET", fmt.Sprintf("/api/get-balance?api_key=%s", t.apiKey), nil)
	return err
}

// Helper: makeRequest handles HTTP requests to Termii API
func (t *TermiiAdapter) makeRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Termii sometimes requires the Content-Type header to be strictly set
	req, err := http.NewRequestWithContext(ctx, method, t.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Termii often returns 400 for logic errors (like invalid phone number)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("termii api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Helper: getSenderID returns the sender ID to use
func (t *TermiiAdapter) getSenderID(requestSenderID string) string {
	if requestSenderID != "" {
		return requestSenderID
	}
	// Fallback to configured default
	if t.senderID != "" {
		return t.senderID
	}
	// Termii specific: "N-Alert" is a common generic sender ID if you haven't registered one yet
	return "N-Alert"
}

// Helper: estimateCost estimates the cost based on destination and segments
func (t *TermiiAdapter) estimateCost(to string, segments int) float64 {
	// Remove '+' if present for prefix matching
	cleanNumber := strings.TrimPrefix(to, "+")

	// Base rates in USD (Approximations)
	// Termii charges in NGN usually, these are converted estimates
	rates := map[string]float64{
		"234": 0.003, // Nigeria (~4-5 NGN) - Cheapest
		"233": 0.025, // Ghana
		"254": 0.030, // Kenya
		"27":  0.035, // South Africa
		"44":  0.045, // UK
		"1":   0.010, // US
	}

	var costPerSegment float64 = 0.050 // Default International Rate

	// Find matching prefix
	for prefix, rate := range rates {
		if strings.HasPrefix(cleanNumber, prefix) {
			costPerSegment = rate
			break
		}
	}

	return costPerSegment * float64(segments)
}

// Helper: calculateSMSSegments calculates the number of SMS segments
func calculateSMSSegments(message string) int {
	length := len(message)
	if length == 0 {
		return 0
	}
	// Standard GSM-7 encoding limits
	if length <= 160 {
		return 1
	}
	// Multipart SMS uses a User Data Header (UDH), reducing payload to 153 chars
	return (length + 152) / 153
}

func isTransactionalMessage(message string) bool {
	if message == "" {
		return false
	}
	lower := strings.ToLower(message)
	keywords := []string{
		"otp",
		"one-time",
		"one time",
		"verification",
		"Authentication",
		"verify",
		"code",
		"2FA",
		"passcode",
		"security",
		"login",
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}
