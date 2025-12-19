package ai

import (
	"regexp"
)

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

// Clean up common field name patterns
func sanitizeReason(reason string) string {
	// Replace snake_case with spaces
	re := regexp.MustCompile(`\b(\w+)_(\w+)\b`)

	// Apply repeatedly to handle multiple underscores
	for re.MatchString(reason) {
		reason = re.ReplaceAllString(reason, "$1 $2")
	}

	return reason
}
