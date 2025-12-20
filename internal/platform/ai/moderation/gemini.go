package aimoderation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/moderation/repository/schema"
	"hauslet/internal/platform/storage"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client  *genai.Client
	storage *storage.R2Storage // Dependency injected
	model   string
}

// Ensure interface compliance
var _ AIClient = (*GeminiClient)(nil)

// NewGeminiClient now requires the Storage dependency
func NewGeminiClient(ctx context.Context, cfg config.GeminiConfig, store *storage.R2Storage) (*GeminiClient, error) {
	clientCfg := &genai.ClientConfig{}

	switch {
	case cfg.APIKey != "":
		// Prefer API key when both API key and project are provided to avoid mutually exclusive error.
		clientCfg.APIKey = cfg.APIKey
		clientCfg.Backend = genai.BackendGeminiAPI
	case cfg.Project != "":
		clientCfg.Project = cfg.Project
		clientCfg.Backend = genai.BackendVertexAI
	default:
		return nil, fmt.Errorf("gemini configuration missing both API key and project")
	}

	client, err := genai.NewClient(ctx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	ModelName := "gemini-2.5-flash"
	if cfg.APIModel != "" {
		ModelName = cfg.APIModel
	}

	return &GeminiClient{
		client:  client,
		storage: store,
		model:   ModelName,
	}, nil
}

func (g *GeminiClient) Provider() string {
	return "gemini"
}

func (g *GeminiClient) HealthCheck(ctx context.Context) error {
	_, err := g.client.Models.List(ctx, nil)
	return err
}

// ---------------------------------------------------------
// CORE MODERATION LOGIC
// ---------------------------------------------------------

type ModerationResponseSchema struct {
	Status     string  `json:"status" description:"The decision: accepted, rejected, or escalated"`
	Confidence float64 `json:"confidence" description:"A score between 0.0 and 1.0 indicating certainty"`
	Reason     string  `json:"reason" description:"A brief explanation for the decision"`
}

func (g *GeminiClient) Moderate(ctx context.Context, input AIModerationInput) (*AIModerationResult, error) {
	var parts []*genai.Part

	// 1. Handle Text
	if input.Text != nil {
		parts = append(parts, &genai.Part{Text: fmt.Sprintf("Analyze this content: %s", *input.Text)})
	}

	// 2. Handle Image
	if input.Image != nil && input.Image.Key != "" {
		data, mimeType, err := g.storage.GetObject(ctx, input.Image.Key)
		if err != nil {
			return nil, fmt.Errorf("image download failed: %w", err)
		}
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				MIMEType: mimeType,
				Data:     data,
			},
		})
	}

	// 3. Handle Video
	if input.Video != nil && input.Video.Key != "" {
		fileURI, err := g.handleVideoUpload(ctx, input.Video.Key, input.Video.MimeType)
		if err != nil {
			return nil, fmt.Errorf("video processing failed: %w", err)
		}

		// Cleanup file on Gemini Server
		defer func() {
			_, _ = g.client.Files.Delete(ctx, fileURI, nil)
		}()

		parts = append(parts, &genai.Part{
			FileData: &genai.FileData{
				FileURI:  fileURI,
				MIMEType: input.Video.MimeType,
			},
		})
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("no content provided")
	}

	// 4. Configure Request with Refined Logic
	sysPrompt := `You are a content safety engine. Analyze the input against these rules:
1. Reject: Hate speech, explicit nudity, gore, logo or branding on media, harassment, spam, or intent to mask contact details.
2. Escalate: Ambiguous content requiring human judgment.
3. Accept: Safe content.

OUTPUT LOGIC:
- Treat the input as a single entity. If a violation (like a phone number) appears multiple times or across different fields, summarize it as ONE single finding in the reason field.
- Do not repeat related findings. Be concise and professional.
- Write your reasoning in natural, human-readable language. Refer to content fields using plain English descriptions (e.g., "in the property description" instead of field names).
- For "intent to mask contact details", describe what was found and where it appeared using natural language.

Example reasoning style:
✅ "Phone number found in the property description"
✅ "Explicit language detected in the title and additional details"
❌ "Violation found in extra_description field"

Respond ONLY with valid JSON.`

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: sysPrompt}},
		},
		ResponseMIMEType: "application/json",
		ResponseSchema:   g.buildSchema(),
	}

	// 5. Execute
	resp, err := g.client.Models.GenerateContent(ctx, g.model, []*genai.Content{{Parts: parts}}, config)
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	return g.parseResponse(resp)
}

// ---------------------------------------------------------
// HELPERS
// ---------------------------------------------------------

func (g *GeminiClient) parseResponse(resp *genai.GenerateContentResponse) (*AIModerationResult, error) {
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from AI")
	}

	rawJSON := resp.Candidates[0].Content.Parts[0].Text

	var parsed ModerationResponseSchema
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON: %w", err)
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
	cleanReason := sanitizeReason(parsed.Reason)

	return &AIModerationResult{
		Status:     status,
		Confidence: parsed.Confidence,
		Reason:     cleanReason,
		RawResponse: map[string]any{
			"raw": rawJSON,
		},
	}, nil
}

func (g *GeminiClient) buildSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"status": {
				Type:        genai.TypeString,
				Enum:        []string{"accepted", "rejected", "escalated"},
				Description: "The decision: accepted, rejected, or escalated.",
			},
			"confidence": {
				Type:        genai.TypeNumber,
				Description: "A score between 0.0 and 1.0 indicating certainty.",
			},
			"reason": {
				Type:        genai.TypeString,
				Description: "A concise, non-repetitive explanation for the decision.",
			},
		},
		Required: []string{"status", "confidence", "reason"},
	}
}

// ---------------------------------------------------------
// MEDIA HANDLING
// ---------------------------------------------------------

func (g *GeminiClient) handleVideoUpload(ctx context.Context, key, mimeType string) (string, error) {
	downloadURL, err := g.storage.GenerateSignedDownloadURL(ctx, key, 1*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed url: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "mod-vid-*.mp4")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download req: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download video stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("video download failed status: %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write video to temp file: %w", err)
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek temp file: %w", err)
	}

	uploadRes, err := g.client.Files.Upload(ctx, tmpFile, &genai.UploadFileConfig{
		DisplayName: key,
		MIMEType:    mimeType,
	})
	if err != nil {
		return "", fmt.Errorf("upload to gemini failed: %w", err)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			file, err := g.client.Files.Get(ctx, uploadRes.Name, nil)
			if err != nil {
				return "", fmt.Errorf("failed to get file status: %w", err)
			}

			if file.State == genai.FileStateActive {
				return file.URI, nil
			}
			if file.State == genai.FileStateFailed {
				return "", fmt.Errorf("video processing failed on server side")
			}
		}
	}
}
