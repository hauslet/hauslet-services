package review

import (
	"context"
	"encoding/json"

	reviewService "hauslet/internal/modules/review/service"
	reviewJobs "hauslet/internal/queue/jobs/review"

	"github.com/go-pkgz/lgr"
)

// StatsRecalculationHandler handles async stats recalculation
type StatsRecalculationHandler struct {
	reviewSvc reviewService.ReviewService
	log       *lgr.Logger
	subject   string
}

// NewStatsRecalculationHandler creates a new handler for stats recalculation
func NewStatsRecalculationHandler(
	reviewSvc reviewService.ReviewService,
	log *lgr.Logger,
	subject string,
) *StatsRecalculationHandler {
	return &StatsRecalculationHandler{
		reviewSvc: reviewSvc,
		log:       log,
		subject:   subject,
	}
}

// JobType returns the job type identifier
func (h *StatsRecalculationHandler) JobType() string {
	return reviewJobs.RecalculateStatsJobType
}

// Subject returns the queue subject this handler listens to
func (h *StatsRecalculationHandler) Subject() string {
	return h.subject
}

// Handle processes the stats recalculation job
func (h *StatsRecalculationHandler) Handle(ctx context.Context, data []byte) error {
	var job reviewJobs.RecalculateStatsJob
	if err := json.Unmarshal(data, &job); err != nil {
		h.log.Logf("ERROR failed to unmarshal RecalculateStatsJob: %v", err)
		return err
	}

	h.log.Logf("INFO recalculating %s stats for target %s", job.TargetType, job.TargetID)

	var err error
	switch job.TargetType {
	case "listing":
		err = h.reviewSvc.RecalculateListingStats(ctx, job.TargetID)
	case "host":
		err = h.reviewSvc.RecalculateHostStats(ctx, job.TargetID)
	default:
		h.log.Logf("WARN unknown target type: %s", job.TargetType)
		return nil // Don't retry for invalid target types
	}

	if err != nil {
		h.log.Logf("ERROR failed to recalculate %s stats for %s: %v", job.TargetType, job.TargetID, err)
		return err
	}

	h.log.Logf("INFO successfully recalculated %s stats for %s", job.TargetType, job.TargetID)
	return nil
}
