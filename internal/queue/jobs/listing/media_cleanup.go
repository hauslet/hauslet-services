package jobs

import "fmt"

const ListingMediaCleanupJobType = "listing_media_cleanup"

// ListingMediaCleanupJob represents a cleanup request for stale media.
type ListingMediaCleanupJob struct {
	OlderThanMinutes int  `json:"older_than_minutes,omitempty"`
	Limit            int  `json:"limit,omitempty"`
	DryRun           bool `json:"dry_run,omitempty"`
}

func (j *ListingMediaCleanupJob) Type() string {
	return ListingMediaCleanupJobType
}

func (j *ListingMediaCleanupJob) Validate() error {
	if j.OlderThanMinutes < 0 {
		return fmt.Errorf("older_than_minutes cannot be negative")
	}
	if j.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	return nil
}
