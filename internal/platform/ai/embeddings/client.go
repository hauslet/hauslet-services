package embeddings

import "context"

type Client struct {
	provider EmbeddingProvider
}

// New creates a new Embeddings client with the given provider
func New(provider EmbeddingProvider) *Client {
	return &Client{provider: provider}
}

// Provider returns the name of the embedding provider
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	return c.provider.Embed(ctx, text)
}

// EmbedBatch generates embeddings for multiple text inputs
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return c.provider.EmbedBatch(ctx, texts)
}

// GetDimensions returns the dimension size of the embeddings
func (c *Client) GetDimensions() int {
	return c.provider.GetDimensions()
}

// GetModelName returns the name of the embedding model
func (c *Client) GetModelName() string {
	return c.provider.GetModelName()
}
