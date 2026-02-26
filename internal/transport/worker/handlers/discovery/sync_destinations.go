package discovery

import (
	"context"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/discovery/service"
)

// SyncDestinationsHandler handles sync_destinations messages to push properties into Redis for autocomplete.
type SyncDestinationsHandler struct {
	repo         service.PropertyDiscoveryHooks
	discoverySvc service.DiscoveryService
	log          *slog.Logger
	subject      string
}

// NewSyncDestinationsHandler creates a new handler.
func NewSyncDestinationsHandler(
	repo service.PropertyDiscoveryHooks,
	discoverySvc service.DiscoveryService,
	log *slog.Logger,
	subject string,
) *SyncDestinationsHandler {
	return &SyncDestinationsHandler{
		repo:         repo,
		discoverySvc: discoverySvc,
		log:          log.With("component", "SyncDestinationsHandler"),
		subject:      subject,
	}
}

// Subject returns the queue subject this handler listens to
func (h *SyncDestinationsHandler) Subject() string {
	return h.subject
}

// JobType returns the job type.
func (h *SyncDestinationsHandler) JobType() string {
	return "sync_destinations"
}

// Handle processes the queue message
func (h *SyncDestinationsHandler) Handle(ctx context.Context, data []byte) error {
	h.log.Info("Starting destinations sync into Redis")

	err := h.discoverySvc.SyncDestinationsToRedis(ctx)
	if err != nil {
		h.log.Error("Failed to sync destinations", "error", err)
		return fmt.Errorf("sync failed: %w", err)
	}

	h.log.Info("Successfully synced destinations to Redis")
	return nil
}
