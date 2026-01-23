package jobs

import "hauslet/internal/queue"

const GenerateEmbeddingsJobType = "listing.generate_embeddings"

type GenerateEmbeddingsJob struct {
	queue.BaseJob
	BatchSize int `json:"batch_size"`
}

func (j GenerateEmbeddingsJob) Type() string {
	return GenerateEmbeddingsJobType
}

func (j GenerateEmbeddingsJob) Validate() error {
	return nil // No specific validation for now
}
