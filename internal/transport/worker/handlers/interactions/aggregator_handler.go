package interactions

import (
	"context"
	"encoding/json"
	"hauslet/internal/modules/interactions/service"
	"log/slog"
	"time"
)

// AggregatorHandler processes hourly aggregation of interactions
type AggregatorHandler struct {
	aggregatorSvc service.AggregatorService
	log           *slog.Logger
	subject       string
}

// AggregatorJob represents the job payload for aggregation
type AggregatorJob struct {
	PeriodType  string    `json:"period_type"`  // "hour" or "day"
	PeriodStart time.Time `json:"period_start"` // Optional: defaults to previous hour/day
}

// NewAggregatorHandler creates a new aggregator handler
func NewAggregatorHandler(
	aggregatorSvc service.AggregatorService,
	log *slog.Logger,
	subject string,
) *AggregatorHandler {
	return &AggregatorHandler{
		aggregatorSvc: aggregatorSvc,
		log:           log,
		subject:       subject,
	}
}

// Handle processes the aggregation job
func (h *AggregatorHandler) Handle(ctx context.Context, payload []byte) error {
	// Parse job payload
	var job AggregatorJob
	if err := json.Unmarshal(payload, &job); err != nil {
		// Default to hourly aggregation of previous hour if payload is invalid
		if h.log != nil {
			h.log.Warn("failed to parse aggregator job, using defaults", "error", err)
		}
		job.PeriodType = "hour"
	}

	// Default period start to previous hour/day if not specified
	if job.PeriodStart.IsZero() {
		now := time.Now().UTC()
		if job.PeriodType == "day" {
			// Previous day (yesterday at 00:00)
			yesterday := now.AddDate(0, 0, -1)
			job.PeriodStart = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
		} else {
			// Previous hour (default)
			job.PeriodStart = now.Truncate(time.Hour).Add(-time.Hour)
		}
	}

	if h.log != nil {
		h.log.Info("starting aggregation job",
			"period_type", job.PeriodType,
			"period_start", job.PeriodStart)
	}

	// Execute aggregation based on period type
	var err error
	switch job.PeriodType {
	case "day":
		err = h.aggregatorSvc.AggregateDaily(ctx, job.PeriodStart)
	case "hour":
		fallthrough
	default:
		err = h.aggregatorSvc.AggregateHourly(ctx, job.PeriodStart)
	}

	if err != nil {
		if h.log != nil {
			h.log.Error("aggregation job failed",
				"error", err,
				"period_type", job.PeriodType,
				"period_start", job.PeriodStart)
		}
		return err
	}

	if h.log != nil {
		h.log.Info("aggregation job completed successfully",
			"period_type", job.PeriodType,
			"period_start", job.PeriodStart)
	}

	return nil
}

// JobType returns the job type this handler processes
func (h *AggregatorHandler) JobType() string {
	return "interactions.aggregator"
}

// Subject returns the queue subject for this handler
func (h *AggregatorHandler) Subject() string {
	return h.subject
}
