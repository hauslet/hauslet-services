package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/property/service"
	"hauslet/internal/queue"
	listingJobs "hauslet/internal/queue/jobs/listing"
)

type EmbeddingGeneratorHandler struct {
	svc     service.PropertyService
	log     *slog.Logger
	subject string
}

func NewEmbeddingGeneratorHandler(svc service.PropertyService, log *slog.Logger, subject string) *EmbeddingGeneratorHandler {
	return &EmbeddingGeneratorHandler{
		svc:     svc,
		log:     log,
		subject: subject,
	}
}

func (h *EmbeddingGeneratorHandler) Handle(ctx context.Context, data []byte) error {
	job, err := h.ParsePayload(data)
	if err != nil {
		h.log.Error("failed to parse job", "error", err)
		return err
	}

	payload, ok := job.(*listingJobs.GenerateEmbeddingsJob)
	if !ok {
		return fmt.Errorf("invalid job type: %T", job)
	}

	h.log.Info("processing embedding generation job", "batch_size", payload.BatchSize)

	count, err := h.svc.GenerateListingEmbeddings(ctx, payload.BatchSize)
	if err != nil {
		h.log.Error("failed to generate embeddings", "error", err)
		return err
	}

	h.log.Info("completed embedding generation job", "processed_count", count)
	return nil
}

func (h *EmbeddingGeneratorHandler) JobType() string {
	return listingJobs.GenerateEmbeddingsJobType
}

func (h *EmbeddingGeneratorHandler) Subject() string {
	return h.subject
}

func (h *EmbeddingGeneratorHandler) ParsePayload(data []byte) (queue.Job, error) {
	var job listingJobs.GenerateEmbeddingsJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}
