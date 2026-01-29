package aiassist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"hauslet/config"

	"google.golang.org/genai"
)

// GeminiAssistClient implements AssistClient using Google's Gemini.
type GeminiAssistClient struct {
	client *genai.Client
	model  string
}

// Ensure interface compliance
var _ AssistClient = (*GeminiAssistClient)(nil)

// NewGeminiAssistClient creates a new Gemini Client for assistance tasks.
func NewGeminiAssistClient(ctx context.Context, cfg config.GeminiConfig) (*GeminiAssistClient, error) {
	clientCfg := &genai.ClientConfig{}

	switch {
	case cfg.APIKey != "":
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

	modelName := "gemini-2.5-flash"
	if cfg.APIModel != "" {
		modelName = cfg.APIModel
	}

	return &GeminiAssistClient{
		client: client,
		model:  modelName,
	}, nil
}

func (g *GeminiAssistClient) GenerateDescription(ctx context.Context, input DescriptionGenerationInput) (string, error) {
	prompt := g.buildDescriptionPrompt(input)

	resp, err := g.client.Models.GenerateContent(ctx, g.model, []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}, nil)

	if err != nil {
		return "", fmt.Errorf("gemini generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from AI")
	}

	return resp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiAssistClient) buildDescriptionPrompt(input DescriptionGenerationInput) string {
	var sb strings.Builder

	sb.WriteString("Write a captivating real estate listing description for the following property:\n\n")
	sb.WriteString(fmt.Sprintf("- Type: %s\n", input.PropertyType))
	sb.WriteString(fmt.Sprintf("- Location: %s, %s\n", input.City, input.State))
	sb.WriteString(fmt.Sprintf("- Bedrooms: %d\n", input.Bedrooms))
	sb.WriteString(fmt.Sprintf("- Bathrooms: %d\n", input.Bathrooms))

	if len(input.Amenities) > 0 {
		sb.WriteString(fmt.Sprintf("- Key Amenities: %s\n", strings.Join(input.Amenities, ", ")))
	}

	if len(input.Highlights) > 0 {
		sb.WriteString(fmt.Sprintf("- Special Highlights: %s\n", strings.Join(input.Highlights, ", ")))
	}

	tone := "Professional and inviting"
	if input.Tone != "" {
		tone = input.Tone
	}
	sb.WriteString(fmt.Sprintf("\nTone: %s\n", tone))
	sb.WriteString("\nInstructions:\n")
	sb.WriteString("- Create a catchy title (first line) and a detailed description body.\n")
	sb.WriteString("- Highlight the lifestyle and convenience.\n")
	sb.WriteString("- Do NOT include placeholders like [Your Name] or [Phone Number].\n")
	sb.WriteString("- Keep it under 200 words.\n")

	return sb.String()
}

func (g *GeminiAssistClient) QualifyLead(ctx context.Context, input LeadQualificationInput) (*LeadQualificationResult, error) {
	prompt := g.buildQualificationPrompt(input)

	// Configure response schema for JSON output
	jsonSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"score":   {Type: genai.TypeNumber, Description: "Qualification score between 0.0 and 1.0"},
			"reason":  {Type: genai.TypeString, Description: "Explanation for the score"},
			"intent":  {Type: genai.TypeString, Description: "Detected intent of the lead (e.g., inquiry, booking, spam)"},
			"urgency": {Type: genai.TypeString, Description: "Urgency level: high, medium, or low"},
		},
		Required: []string{"score", "reason", "intent", "urgency"},
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   jsonSchema,
	})

	if err != nil {
		return nil, fmt.Errorf("gemini qualification failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from AI")
	}

	// Parse JSON response
	jsonPart := resp.Candidates[0].Content.Parts[0].Text
	var result LeadQualificationResult
	if err := parseJSON(jsonPart, &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &result, nil
}

func (g *GeminiAssistClient) buildQualificationPrompt(input LeadQualificationInput) string {
	var sb strings.Builder
	sb.WriteString("Analyze the following real estate lead and determine its qualification quality.\n")
	sb.WriteString("Consider the intent, completeness, and professionalism of the message.\n\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", input.Name))
	sb.WriteString(fmt.Sprintf("Email: %s\n", input.Email)) // Email domain can be a signal
	sb.WriteString(fmt.Sprintf("Source: %s\n", input.Source))
	sb.WriteString(fmt.Sprintf("Message: %s\n", input.Message))
	sb.WriteString("\nEvaluate:\n")
	sb.WriteString("- Score (0.0-1.0): 1.0 is a perfect, serious lead. 0.0 is complete junk/spam.\n")
	sb.WriteString("- Intent: What does the user want?\n")
	sb.WriteString("- Urgency: Is this time-sensitive?\n")
	return sb.String()
}

// Helper to parse JSON (assuming simple json unmarshal)
func parseJSON(s string, v any) error {
	// Basic cleanup if model adds markdown blocks
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return json.Unmarshal([]byte(s), v)
}
