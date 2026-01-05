package interactions

import (
	"context"
	"encoding/json"
	"hauslet/internal/modules/interactions/domain"
	"hauslet/internal/modules/interactions/repository"
	"hauslet/internal/modules/interactions/repository/schema"
	"hauslet/internal/modules/interactions/service"
	"hauslet/internal/platform/redis"
	"log/slog"
)

// BatchWriterHandler processes interactions from Redis queue and writes to database
type BatchWriterHandler struct {
	redis           redis.RedisClient
	interactionRepo repository.InteractionRepository
	log             *slog.Logger
	subject         string
	batchSize       int
}

// NewBatchWriterHandler creates a new batch writer handler
func NewBatchWriterHandler(
	redis redis.RedisClient,
	interactionRepo repository.InteractionRepository,
	log *slog.Logger,
	subject string,
) *BatchWriterHandler {
	return &BatchWriterHandler{
		redis:           redis,
		interactionRepo: interactionRepo,
		log:             log,
		subject:         subject,
		batchSize:       100,
	}
}

// Handle processes a batch of interactions from Redis
func (h *BatchWriterHandler) Handle(ctx context.Context, payload []byte) error {
	// This handler is triggered by Cloud Scheduler
	// It pops items from Redis and bulk inserts into PostgreSQL

	items := make([]string, 0, h.batchSize)

	// Pop multiple items from queue
	for i := 0; i < h.batchSize; i++ {
		item, err := h.redis.LPop(ctx, service.RedisQueueKey).Result()
		if err != nil || item == "" {
			break // Queue is empty
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		if h.log != nil {
			h.log.Debug("no interactions to process")
		}
		return nil // Nothing to process
	}

	// Deserialize interactions
	interactions := make([]*schema.Interaction, 0, len(items))

	for _, item := range items {
		var domainInteraction domain.Interaction
		if err := json.Unmarshal([]byte(item), &domainInteraction); err != nil {
			if h.log != nil {
				h.log.Warn("failed to unmarshal interaction", "error", err)
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
	if err := h.interactionRepo.BulkInsert(ctx, interactions); err != nil {
		if h.log != nil {
			h.log.Error("failed to bulk insert interactions", "error", err, "count", len(interactions))
		}
		return err
	}

	if h.log != nil {
		h.log.Info("interactions batch processed", "count", len(interactions))
	}

	return nil
}

// JobType returns the job type this handler processes
func (h *BatchWriterHandler) JobType() string {
	return "interactions.batch_writer"
}

// Subject returns the queue subject for this handler
func (h *BatchWriterHandler) Subject() string {
	return h.subject
}
