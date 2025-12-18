package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/moderation/repository/schema"
	"hauslet/internal/platform/storage"
)

type AnthropicClient struct {
	apiKey     string
	storage    *storage.R2Storage
	model      string
	apiBaseURL string
	httpClient *http.Client
}

// Ensure interface compliance
var _ AIClient = (*AnthropicClient)(nil)

// NewAnthropicClient creates a new Anthropic Claude client
func NewAnthropicClient(ctx context.Context, cfg config.AnthropicConfig, store *storage.R2Storage) (*AnthropicClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}

	modelName := "claude-sonnet-4-20250514"
	if cfg.APIModel != "" {
		modelName = cfg.APIModel
	}

	baseURL := "https://api.anthropic.com/v1"
	if cfg.APIURL != "" {
		baseURL = cfg.APIURL
	}

	return &AnthropicClient{
		apiKey:     cfg.APIKey,
		storage:    store,
		model:      modelName,
		apiBaseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}, nil
}

func (a *AnthropicClient) Provider() string {
	return "anthropic"
}

func (a *AnthropicClient) HealthCheck(ctx context.Context) error {
	// Simple health check: try to make a minimal API call
	req := anthropicRequest{
		Model:     a.model,
		MaxTokens: 10,
		Messages: []anthropicMessage{
			{
				Role: "user",
				Content: []anthropicContent{
					{Type: "text", Text: "Hi"},
				},
			},
		},
	}

	_, err := a.makeRequest(ctx, req)
	return err
}

// ---------------------------------------------------------
// CORE MODERATION LOGIC
// ---------------------------------------------------------

func (a *AnthropicClient) Moderate(ctx context.Context, input AIModerationInput) (*AIModerationResult, error) {
	var contentParts []anthropicContent

	// 1. Handle Text
	if input.Text != nil {
		contentParts = append(contentParts, anthropicContent{
			Type: "text",
			Text: fmt.Sprintf("Analyze this text: %s", *input.Text),
		})
	}

	// 2. Handle Image
	if input.Image != nil && input.Image.Key != "" {
		imageContent, err := a.handleImage(ctx, input.Image.Key, input.Image.MimeType)
		if err != nil {
			return nil, fmt.Errorf("image processing failed: %w", err)
		}
		contentParts = append(contentParts, *imageContent)
	}

	// 3. Video not supported - return error immediately
	if input.Video != nil && input.Video.Key != "" {
		return nil, fmt.Errorf("video moderation not supported by Anthropic provider")
	}

	if len(contentParts) == 0 {
		return nil, fmt.Errorf("no content provided")
	}

	// 4. Build System Prompt
	systemPrompt := `You are a content safety engine. Analyze the input against these rules:
1. Reject: Hate speech, explicit nudity, gore, logo or branding on media, harassment, spam, or intent to mask contact details.
2. Escalate: Ambiguous content requiring human judgment.
3. Accept: Safe content.


Respond ONLY with valid JSON in this exact format:
{
  "status": "accepted|rejected|escalated",
  "confidence": 0.95,
  "reason": "Brief explanation, Do not repeat or rephrase the same issue."
}`

	// 5. Build Request
	req := anthropicRequest{
		Model:     a.model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: contentParts,
			},
		},
	}

	// 6. Execute
	resp, err := a.makeRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("anthropic generation failed: %w", err)
	}

	return a.parseResponse(resp)
}

// ---------------------------------------------------------
// HELPERS
// ---------------------------------------------------------

func (a *AnthropicClient) parseResponse(resp *anthropicResponse) (*AIModerationResult, error) {
	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("empty response from AI")
	}

	// Extract text from response
	var rawText string
	for _, content := range resp.Content {
		if content.Type == "text" {
			rawText = content.Text
			break
		}
	}

	if rawText == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	// Parse JSON response
	var parsed ModerationResponseSchema
	if err := json.Unmarshal([]byte(rawText), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON: %w (raw: %s)", err, rawText)
	}

	var status schema.ModerationStatus
	switch strings.ToLower(parsed.Status) {
	case "accepted":
		status = schema.ModerationStatusAccepted
	case "rejected":
		status = schema.ModerationStatusRejected
	default:
		status = schema.ModerationStatusEscalated
	}

	return &AIModerationResult{
		Status:     status,
		Confidence: parsed.Confidence,
		Reason:     parsed.Reason,
		RawResponse: map[string]any{
			"raw":   rawText,
			"model": resp.Model,
			"usage": resp.Usage,
		},
	}, nil
}

// ---------------------------------------------------------
// MEDIA HANDLING
// ---------------------------------------------------------

func (a *AnthropicClient) handleImage(ctx context.Context, key, mimeType string) (*anthropicContent, error) {
	// Download image from R2
	data, detectedMimeType, err := a.storage.GetObject(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("image download failed: %w", err)
	}

	// Use detected mime type if provided, otherwise use input
	if detectedMimeType != "" {
		mimeType = detectedMimeType
	}

	// Validate mime type
	if !isValidImageMimeType(mimeType) {
		return nil, fmt.Errorf("unsupported image mime type: %s", mimeType)
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	return &anthropicContent{
		Type: "image",
		Source: &anthropicImageSource{
			Type:      "base64",
			MediaType: mimeType,
			Data:      base64Data,
		},
	}, nil
}

func isValidImageMimeType(mimeType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	for _, valid := range validTypes {
		if mimeType == valid {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------
// API REQUEST/RESPONSE STRUCTURES
// ---------------------------------------------------------

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type   string                `json:"type"` // "text" or "image"
	Text   string                `json:"text,omitempty"`
	Source *anthropicImageSource `json:"source,omitempty"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", etc.
	Data      string `json:"data"`       // base64 encoded
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// ---------------------------------------------------------
// HTTP CLIENT
// ---------------------------------------------------------

func (a *AnthropicClient) makeRequest(ctx context.Context, req anthropicRequest) (*anthropicResponse, error) {
	// Serialize request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.apiBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Execute request
	httpResp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle errors
	if httpResp.StatusCode != http.StatusOK {
		var apiErr anthropicError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", httpResp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("API error (%s): %s", apiErr.Error.Type, apiErr.Error.Message)
	}

	// Parse success response
	var resp anthropicResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}
