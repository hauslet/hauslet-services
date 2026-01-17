package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/messaging/service"
	messagingJob "hauslet/internal/queue/jobs/messaging"
)

// ConversationCleanupHandler archives stale conversations and deletes old archived ones.
type ConversationCleanupHandler struct {
	messagingSvc service.MessagingService
	log          *slog.Logger
	subject      string
}

// NewConversationCleanupHandler constructs a conversation cleanup handler.
func NewConversationCleanupHandler(messagingSvc service.MessagingService, log *slog.Logger, subject string) *ConversationCleanupHandler {
	return &ConversationCleanupHandler{
		messagingSvc: messagingSvc,
		log:          log,
		subject:      subject,
	}
}

func (h *ConversationCleanupHandler) JobType() string {
	return messagingJob.ConversationCleanupJobType
}

func (h *ConversationCleanupHandler) Subject() string {
	return h.subject
}

func (h *ConversationCleanupHandler) Handle(ctx context.Context, data []byte) error {
	var job messagingJob.ConversationCleanupJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal conversation cleanup job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid conversation cleanup job: %w", err)
	}

	checkTime := job.GetCheckTime()
	h.log.Info("processing conversation cleanup", "check_time", checkTime)

	// Archive stale conversations
	archivedIDs, err := h.messagingSvc.ArchiveStaleConversations(ctx)
	if err != nil {
		return fmt.Errorf("failed to archive stale conversations: %w", err)
	}

	h.log.Info("archived stale conversations", "count", len(archivedIDs))

	// Delete old archived conversations
	deletedIDs, err := h.messagingSvc.DeleteOldArchivedConversations(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete old archived conversations: %w", err)
	}

	h.log.Info("deleted old archived conversations", "count", len(deletedIDs))
	h.log.Info("conversation cleanup completed", "archived", len(archivedIDs), "deleted", len(deletedIDs))

	return nil
}
