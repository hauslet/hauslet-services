package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	defaultArchiveAfterDays = 30
	defaultDeleteAfterDays  = 60
	cleanupBatchLimit       = 100
)

// ArchiveStaleConversations archives conversations that have been inactive for the configured period.
func (s *messagingServiceImpl) ArchiveStaleConversations(ctx context.Context) ([]uuid.UUID, error) {
	archiveAfterDays := defaultArchiveAfterDays
	if s.platformConfig != nil && s.platformConfig.Messaging.ArchiveAfterDays > 0 {
		archiveAfterDays = s.platformConfig.Messaging.ArchiveAfterDays
	}

	cutoff := time.Now().AddDate(0, 0, -archiveAfterDays)

	staleConversations, err := s.convRepo.FindStaleConversations(ctx, cutoff, cleanupBatchLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to find stale conversations: %w", err)
	}

	if len(staleConversations) == 0 {
		if s.log != nil {
			s.log.Info("no stale conversations found for archival")
		}
		return []uuid.UUID{}, nil
	}

	archivedIDs := make([]uuid.UUID, 0, len(staleConversations))

	for _, conv := range staleConversations {
		conv.Status = "archived"
		conv.UpdatedAt = time.Now()

		if err := s.convRepo.Update(ctx, conv); err != nil {
			if s.log != nil {
				s.log.Error("failed to archive conversation", "conversation_id", conv.ID, "error", err)
			}
			continue
		}

		archivedIDs = append(archivedIDs, conv.ID)
		if s.log != nil {
			s.log.Info("archived stale conversation", "conversation_id", conv.ID)
		}
	}

	if s.log != nil {
		s.log.Info("archived stale conversations", "count", len(archivedIDs))
	}

	return archivedIDs, nil
}

// DeleteOldArchivedConversations soft-deletes archived conversations older than the configured period.
func (s *messagingServiceImpl) DeleteOldArchivedConversations(ctx context.Context) ([]uuid.UUID, error) {
	deleteAfterDays := defaultDeleteAfterDays
	if s.platformConfig != nil && s.platformConfig.Messaging.DeleteAfterDays > 0 {
		deleteAfterDays = s.platformConfig.Messaging.DeleteAfterDays
	}

	cutoff := time.Now().AddDate(0, 0, -deleteAfterDays)

	archivedConversations, err := s.convRepo.FindArchivedForDeletion(ctx, cutoff, cleanupBatchLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to find archived conversations for deletion: %w", err)
	}

	if len(archivedConversations) == 0 {
		if s.log != nil {
			s.log.Info("no archived conversations found for deletion")
		}
		return []uuid.UUID{}, nil
	}

	deletedIDs := make([]uuid.UUID, 0, len(archivedConversations))

	for _, conv := range archivedConversations {
		if err := s.convRepo.SoftDelete(ctx, conv.ID); err != nil {
			if s.log != nil {
				s.log.Error("failed to delete archived conversation", "conversation_id", conv.ID, "error", err)
			}
			continue
		}

		deletedIDs = append(deletedIDs, conv.ID)
		if s.log != nil {
			s.log.Info("deleted archived conversation", "conversation_id", conv.ID)
		}
	}

	if s.log != nil {
		s.log.Info("deleted old archived conversations", "count", len(deletedIDs))
	}

	return deletedIDs, nil
}
