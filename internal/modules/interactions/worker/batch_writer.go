package worker

import (
	"context"
	"encoding/json"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/modules/interactions/repository"
	"hauslet/internal/modules/interactions/repository/schema"
	"hauslet/internal/modules/interactions/service"
	"hauslet/internal/platform/redis"
	"log/slog"
	"time"
)

// BatchWriter consumes interactions from Redis queue and writes to database
type BatchWriter struct {
	redis           redis.RedisClient
	interactionRepo repository.InteractionRepository
	log             *slog.Logger
	batchSize       int
	pollInterval    time.Duration
}

// NewBatchWriter creates a new batch writer worker
func NewBatchWriter(
	redis redis.RedisClient,
	interactionRepo repository.InteractionRepository,
	log *slog.Logger,
) *BatchWriter {
	return &BatchWriter{
		redis:           redis,
		interactionRepo: interactionRepo,
		log:             log,
		batchSize:       100,
		pollInterval:    1 * time.Second,
	}
}

// Start begins the batch processing loop
func (w *BatchWriter) Start(ctx context.Context) error {
	if w.log != nil {
		w.log.Info("batch writer started", "batch_size", w.batchSize, "poll_interval", w.pollInterval)
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if w.log != nil {
				w.log.Info("batch writer stopping")
			}
			return ctx.Err()

		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				if w.log != nil {
					w.log.Error("failed to process batch", "error", err)
				}
				// Continue processing despite errors
			}
		}
	}
}

// processBatch pulls items from Redis and inserts them into the database
func (w *BatchWriter) processBatch(ctx context.Context) error {
	// Pop items from Redis queue
	items := make([]string, 0, w.batchSize)

	for i := 0; i < w.batchSize; i++ {
		item, err := w.redis.LPop(ctx, service.RedisQueueKey)
		if err != nil || item == "" {
			break // Queue is empty or error occurred
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil // Nothing to process
	}

	// Deserialize interactions
	interactions := make([]*schema.Interaction, 0, len(items))

	for _, item := range items {
		var domainInteraction domain.Interaction
		if err := json.Unmarshal([]byte(item), &domainInteraction); err != nil {
			if w.log != nil {
				w.log.Warn("failed to unmarshal interaction", "error", err)
			}
			continue // Skip invalid items
		}

		schemaInteraction := schema.MapInteractionFromDomain(&domainInteraction)
		interactions = append(interactions, schemaInteraction)
	}

	if len(interactions) == 0 {
		return nil
	}

	// Bulk insert into database
	if err := w.interactionRepo.BulkInsert(ctx, interactions); err != nil {
		if w.log != nil {
			w.log.Error("failed to bulk insert interactions", "error", err, "count", len(interactions))
		}
		return err
	}

	if w.log != nil {
		w.log.Info("batch processed successfully", "count", len(interactions))
	}

	return nil
}
