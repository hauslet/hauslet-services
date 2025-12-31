package review

import (
	"context"
	"encoding/json"
	"log/slog"

	reviewService "hauslet/internal/modules/review/service"
	reviewJobs "hauslet/internal/queue/jobs/review"
)

// StatsRecalculationHandler handles async stats recalculation
type StatsRecalculationHandler struct {
	reviewSvc reviewService.ReviewService
	log       *slog.Logger
	subject   string
}

// NewStatsRecalculationHandler creates a new handler for stats recalculation
func NewStatsRecalculationHandler(
	reviewSvc reviewService.ReviewService,
	log *slog.Logger,
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
		h.log.Error("failed to unmarshal RecalculateStatsJob", "error", err)
		return err
	}

	h.log.Info("recalculating stats", "target_type", job.TargetType, "target_id", job.TargetID)
	var err error
	switch job.TargetType {
	case "listing":
		err = h.reviewSvc.RecalculateListingStats(ctx, job.TargetID)
	case "host":
		err = h.reviewSvc.RecalculateHostStats(ctx, job.TargetID)
	default:
		h.log.Warn("unknown target type", "target_type", job.TargetType)
		return nil // Don't retry for invalid target types
	}

	if err != nil {
		h.log.Error("failed to recalculate stats", "target_type", job.TargetType, "target_id", job.TargetID, "error", err)
		return err
	}

	h.log.Info("successfully recalculated stats", "target_type", job.TargetType, "target_id", job.TargetID)
	return nil
}
