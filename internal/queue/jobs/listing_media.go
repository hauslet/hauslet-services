package jobs

import (
	"fmt"

	"github.com/google/uuid"
)

const ListingMediaThumbnailJobType = "listing_media_thumbnail"

// ListingMediaThumbnailJob represents a request to generate thumbnails for finalized listing media.
type ListingMediaThumbnailJob struct {
	ListingID uuid.UUID `json:"listing_id"`
	MediaKeys []string  `json:"media_keys"`
}

// Type returns the job type identifier.
func (j *ListingMediaThumbnailJob) Type() string {
	return ListingMediaThumbnailJobType
}

// Validate ensures the job payload is valid.
func (j *ListingMediaThumbnailJob) Validate() error {
	if j.ListingID == uuid.Nil {
		return fmt.Errorf("listing_id is required")
	}
	if len(j.MediaKeys) == 0 {
		return fmt.Errorf("media_keys is required")
	}
	return nil
}
