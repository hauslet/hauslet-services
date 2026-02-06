package finance

import (
	"context"
	"encoding/json"
	financeService "hauslet/internal/modules/finance/service"
	financeJobs "hauslet/internal/queue/jobs/finance"
	"log/slog"
)

// PenaltyDebtCollectionHandler handles unpaid penalty collection.
type PenaltyDebtCollectionHandler struct {
	financeSvc financeService.FinanceService
	log        *slog.Logger
	subject    string
}

// NewPenaltyDebtCollectionHandler creates a new handler for penalty debt collection.
func NewPenaltyDebtCollectionHandler(
	financeSvc financeService.FinanceService,
	log *slog.Logger,
	subject string,
) *PenaltyDebtCollectionHandler {
	return &PenaltyDebtCollectionHandler{
		financeSvc: financeSvc,
		log:        log,
		subject:    subject,
	}
}

// JobType returns the job type identifier.
func (h *PenaltyDebtCollectionHandler) JobType() string {
	return financeJobs.PenaltyDebtCollectionJobType
}

// Subject returns the queue subject this handler listens to.
func (h *PenaltyDebtCollectionHandler) Subject() string {
	return h.subject
}

// Handle processes the penalty debt collection job.
func (h *PenaltyDebtCollectionHandler) Handle(ctx context.Context, data []byte) error {
	var job financeJobs.PenaltyDebtCollectionJob
	if err := json.Unmarshal(data, &job); err != nil {
		if h.log != nil {
			h.log.Error("failed to unmarshal PenaltyDebtCollectionJob", "error", err)
		}
		return err
	}

	checkTime := job.GetCheckTime()
	if h.log != nil {
		h.log.Info("processing penalty debt collection job", "time", checkTime)
	}

	report, err := h.financeSvc.CollectOutstandingPenaltyDebts(ctx)
	if err != nil {
		if h.log != nil {
			h.log.Error("penalty debt collection failed", "error", err)
		}
		return err
	}

	if h.log != nil {
		h.log.Info("penalty debt collection completed",
			"processed", report.DebtsProcessed,
			"settled", report.DebtsSettled,
			"amount_collected", report.AmountCollected,
		)
	}

	return nil
}
