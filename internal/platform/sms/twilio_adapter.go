package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	twilioTimeout = 30 * time.Second
)

// TwilioAdapter implements Provider for Twilio (Global SMS provider)
type TwilioAdapter struct {
	accountSID string
	authToken  string
	baseURL    string
	fromNumber string
	httpClient *http.Client
}

// NewTwilioAdapter creates a new Twilio adapter
func NewTwilioAdapter(accountSID, authToken, fromNumber, baseURL string) *TwilioAdapter {
	return &TwilioAdapter{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		baseURL:    baseURL,
		httpClient: &http.Client{
			Timeout: twilioTimeout,
		},
	}
}

// Provider returns the provider name
func (tw *TwilioAdapter) Provider() string {
	return "twilio"
}

// SupportedCountries returns countries supported by Twilio
func (tw *TwilioAdapter) SupportedCountries() []string {
	// Twilio is truly global, so we can return an empty list
	// or specific regions if you want to use this for routing logic.
	return []string{} // Implies Global/All
}

// Send sends an SMS via Twilio
func (tw *TwilioAdapter) Send(ctx context.Context, req SMSRequest) (*SMSResponse, error) {
	// Prepare Twilio API request (Form Data, NOT JSON)
	// Reference: https://www.twilio.com/docs/sms/api/message-resource
	formData := url.Values{}
	formData.Set("To", req.To)
	formData.Set("From", tw.getFromNumber(req.From))
	formData.Set("Body", req.Message)

	// Make HTTP request
	path := fmt.Sprintf("/Accounts/%s/Messages.json", tw.accountSID)
	respData, err := tw.makeRequest(ctx, "POST", path, formData)
	if err != nil {
		return nil, fmt.Errorf("twilio api request failed: %w", err)
	}

	// Parse Twilio response
	var twilioResp struct {
		SID          string  `json:"sid"`
		Status       string  `json:"status"`
		ErrorCode    *int    `json:"error_code"`
		ErrorMessage *string `json:"error_message"`
		Price        *string `json:"price"` // Can be null immediately after send
		PriceUnit    string  `json:"price_unit"`
		NumSegments  string  `json:"num_segments"` // Twilio returns this as a string
	}

	if err := json.Unmarshal(respData, &twilioResp); err != nil {
		return nil, fmt.Errorf("failed to parse twilio response: %w", err)
	}

	// Check for logic error inside 200 OK (rare but possible)
	if twilioResp.ErrorCode != nil {
		errorMsg := "unknown error"
		if twilioResp.ErrorMessage != nil {
			errorMsg = *twilioResp.ErrorMessage
		}
		return nil, fmt.Errorf("twilio error %d: %s", *twilioResp.ErrorCode, errorMsg)
	}

	// Parse segments (Safe conversion)
	segments := 1
	if twilioResp.NumSegments != "" {
		if val, err := strconv.Atoi(twilioResp.NumSegments); err == nil {
			segments = val
		}
	}

	// Parse cost
	// Note: Twilio 'Price' is often NULL in the immediate response.
	// It populates asynchronously.
	cost := 0.0
	if twilioResp.Price != nil {
		// Remove negative sign (Twilio returns "-0.0075")
		priceStr := strings.TrimPrefix(*twilioResp.Price, "-")
		if val, err := strconv.ParseFloat(priceStr, 64); err == nil {
			cost = val
		}
	}

	return &SMSResponse{
		Success:   true,
		MessageID: twilioResp.SID,
		Status:    twilioResp.Status,
		Provider:  "twilio",
		Cost:      cost,
		Segments:  segments,
		Message:   "SMS sent successfully",
	}, nil
}

// HealthCheck verifies Twilio API is reachable
func (tw *TwilioAdapter) HealthCheck(ctx context.Context) error {
	// Fetch account details to verify credentials
	path := fmt.Sprintf("/Accounts/%s.json", tw.accountSID)
	_, err := tw.makeRequest(ctx, "GET", path, nil)
	return err
}

// Helper: makeRequest handles HTTP requests to Twilio API
func (tw *TwilioAdapter) makeRequest(ctx context.Context, method, path string, formData url.Values) ([]byte, error) {
	var reqBody io.Reader
	if formData != nil {
		reqBody = strings.NewReader(formData.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, tw.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Headers
	if formData != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Accept", "application/json")

	// Set Basic Auth
	req.SetBasicAuth(tw.accountSID, tw.authToken)

	// Execute request
	resp, err := tw.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		// Try to parse Twilio's standard error format
		var twilioError struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  int    `json:"status"`
		}
		if json.Unmarshal(respBody, &twilioError) == nil && twilioError.Message != "" {
			return nil, fmt.Errorf("twilio api error %d: %s", twilioError.Code, twilioError.Message)
		}
		return nil, fmt.Errorf("twilio api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Helper: getFromNumber returns the from number to use
func (tw *TwilioAdapter) getFromNumber(requestFromNumber string) string {
	if requestFromNumber != "" {
		return requestFromNumber
	}
	return tw.fromNumber
}
