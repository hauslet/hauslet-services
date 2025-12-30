package review

import (
	"context"
	"encoding/json"

	reviewService "hauslet/internal/modules/review/service"
	reviewJobs "hauslet/internal/queue/jobs/review"

	"github.com/go-pkgz/lgr"
)

// StandoffPublishHandler handles publishing reviews stuck in standoff
type StandoffPublishHandler struct {
	reviewSvc reviewService.ReviewService
	log       *lgr.Logger
	subject   string
}

// NewStandoffPublishHandler creates a new handler for publishing standoff reviews
func NewStandoffPublishHandler(
	reviewSvc reviewService.ReviewService,
	log *lgr.Logger,
	subject string,
) *StandoffPublishHandler {
	return &StandoffPublishHandler{
		reviewSvc: reviewSvc,
		log:       log,
		subject:   subject,
	}
}

// JobType returns the job type identifier
func (h *StandoffPublishHandler) JobType() string {
	return reviewJobs.PublishStandoffsJobType
}

// Subject returns the queue subject this handler listens to
func (h *StandoffPublishHandler) Subject() string {
	return h.subject
}

// Handle processes the standoff publication job
func (h *StandoffPublishHandler) Handle(ctx context.Context, data []byte) error {
	var job reviewJobs.PublishStandoffsJob
	if err := json.Unmarshal(data, &job); err != nil {
		h.log.Logf("ERROR failed to unmarshal PublishStandoffsJob: %v", err)
		return err
	}

	thresholdTime := job.GetThresholdTime()
	h.log.Logf("INFO publishing standoff reviews older than %s", thresholdTime)

	publishedCount, err := h.reviewSvc.PublishExpiredStandoffs(ctx, thresholdTime)
	if err != nil {
		h.log.Logf("ERROR failed to publish expired standoffs: %v", err)
		return err
	}

	h.log.Logf("INFO successfully published %d expired standoff reviews", publishedCount)
	return nil
}
