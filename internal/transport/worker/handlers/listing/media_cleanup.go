package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"hauslet/internal/modules/property/repository"
	"hauslet/internal/platform/storage"
	listingJob "hauslet/internal/queue/jobs/listing"

	"github.com/go-pkgz/lgr"
	"github.com/google/uuid"
)

// ListingMediaCleanupHandler removes stale, unuploaded listing media.
type ListingMediaCleanupHandler struct {
	repo     repository.Repository
	storage  *storage.R2Storage
	log      *lgr.Logger
	subject  string
	defaults cleanupDefaults
}

type cleanupDefaults struct {
	olderThan time.Duration
	limit     int
}

// NewListingMediaCleanupHandler constructs a cleanup handler.
func NewListingMediaCleanupHandler(repo repository.Repository, storage *storage.R2Storage, log *lgr.Logger, subject string) *ListingMediaCleanupHandler {
	return &ListingMediaCleanupHandler{
		repo:    repo,
		storage: storage,
		log:     log,
		subject: subject,
		defaults: cleanupDefaults{
			olderThan: 2 * time.Hour,
			limit:     200,
		},
	}
}

func (h *ListingMediaCleanupHandler) JobType() string {
	return listingJob.ListingMediaCleanupJobType
}

func (h *ListingMediaCleanupHandler) Subject() string {
	return h.subject
}

func (h *ListingMediaCleanupHandler) Handle(ctx context.Context, data []byte) error {
	var job listingJob.ListingMediaCleanupJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal cleanup job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid cleanup job: %w", err)
	}

	olderThan := h.defaults.olderThan
	if job.OlderThanMinutes > 0 {
		olderThan = time.Duration(job.OlderThanMinutes) * time.Minute
	}

	limit := h.defaults.limit
	if job.Limit > 0 {
		limit = job.Limit
	}

	cutoff := time.Now().Add(-olderThan)
	stale, err := h.repo.FindStaleListingMedia(ctx, cutoff, limit)
	if err != nil {
		return err
	}

	if len(stale) == 0 {
		h.log.Logf("INFO cleanup: no stale media older than %s", olderThan)
		return nil
	}

	// Group by listing for scoped delete.
	listingToIDs := make(map[uuid.UUID][]uuid.UUID)
	keySet := make(map[string]struct{})
	for _, m := range stale {
		listingToIDs[m.ListingID] = append(listingToIDs[m.ListingID], m.ID)
		if m.Key != "" {
			keySet[m.Key] = struct{}{}
		}
	}

	if job.DryRun {
		h.log.Logf("INFO cleanup dry-run: found %d stale media across %d listings", len(stale), len(listingToIDs))
		return nil
	}

	for listingID, ids := range listingToIDs {
		if err := h.repo.DeleteListingMedia(ctx, listingID, ids); err != nil {
			return fmt.Errorf("delete listing %s media: %w", listingID, err)
		}
	}

	if h.storage != nil && len(keySet) > 0 {
		for key := range keySet {
			if err := h.storage.DeleteObject(ctx, key); err != nil {
				// Best-effort; log and continue.
				h.log.Logf("ERROR cleanup: failed to delete object %s: %v", key, err)
			}
		}
	}

	h.log.Logf("INFO cleanup: removed %d stale media across %d listings", len(stale), len(listingToIDs))
	return nil
}
