package embeddings

import (
	"context"
	"fmt"
	"hauslet/config"

	"google.golang.org/genai"
)

// GeminiProvider implements EmbeddingProvider for Google's Gemini API
type GeminiProvider struct {
	client         *genai.Client
	model          string
	taskType       string
	dimensionality *int32
}

var _ EmbeddingProvider = (*GeminiProvider)(nil)

// GeminiConfig holds configuration for Gemini embedding provider
type GeminiConfig struct {
	APIKey     string
	EmbedModel string // defaults to "gemini-embedding-001"
}

// NewGeminiProvider creates a new Gemini embedding provider
func NewGeminiProvider(ctx context.Context, cfg config.GeminiConfig) (*GeminiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.EmbedModel == "" {
		cfg.EmbedModel = "gemini-embedding-001"
	}

	// Initialize the new client with the API key configuration
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	d := int32(768)
	return &GeminiProvider{
		client:         client,
		model:          cfg.EmbedModel,
		taskType:       "SEMANTIC_SIMILARITY",
		dimensionality: &d,
	}, nil
}

// Embed generates embeddings for a single text input
func (g *GeminiProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Create the content structure required by the new SDK
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: text},
			},
		},
	}

	// Call EmbedContent using the client.Models service

	res, err := g.client.Models.EmbedContent(ctx, g.model, contents, &genai.EmbedContentConfig{
		TaskType:             g.taskType,
		OutputDimensionality: g.dimensionality,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if res == nil || len(res.Embeddings) == 0 || len(res.Embeddings[0].Values) == 0 {
		return nil, fmt.Errorf("received empty embedding response")
	}

	return res.Embeddings[0].Values, nil
}

// EmbedBatch generates embeddings for multiple text inputs
func (g *GeminiProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	// Convert texts to slice of *genai.Content
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		if text == "" {
			return nil, fmt.Errorf("text at index %d is empty", i)
		}
		contents[i] = &genai.Content{
			Parts: []*genai.Part{
				{Text: text},
			},
		}
	}

	// The new SDK handles batching via the same EmbedContent method when multiple contents are passed
	res, err := g.client.Models.EmbedContent(ctx, g.model, contents, &genai.EmbedContentConfig{
		TaskType:             g.taskType,
		OutputDimensionality: g.dimensionality,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate batch embeddings: %w", err)
	}

	if res == nil || len(res.Embeddings) == 0 {
		return nil, fmt.Errorf("received empty batch embedding response")
	}

	if len(res.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings but got %d", len(texts), len(res.Embeddings))
	}

	embeddings := make([][]float32, len(res.Embeddings))
	for i, embedding := range res.Embeddings {
		if embedding == nil || len(embedding.Values) == 0 {
			return nil, fmt.Errorf("received empty embedding at index %d", i)
		}
		embeddings[i] = embedding.Values
	}

	return embeddings, nil
}

// GetDimensions returns the dimension size of the embeddings
func (g *GeminiProvider) GetDimensions() int {
	// gemini-embedding-001 has 768 dimensions
	return 768
}

// GetModelName returns the identifier of the embedding model being used
func (g *GeminiProvider) GetModelName() string {
	return g.model
}
