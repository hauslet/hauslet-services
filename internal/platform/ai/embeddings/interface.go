package embeddings

import "context"

// EmbeddingProvider defines the interface for text embedding services
type EmbeddingProvider interface {
	// Embed generates embeddings for a single text input
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple text inputs
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// GetDimensions returns the dimension size of the embeddings
	GetDimensions() int

	// GetModelName returns the identifier of the embedding model being used
	GetModelName() string
}
