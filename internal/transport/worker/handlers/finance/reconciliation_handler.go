package finance

import (
	"context"
	"encoding/json"
	financeService "hauslet/internal/modules/finance/service"
	financeJobs "hauslet/internal/queue/jobs/finance"
	"log/slog"
)

// ReconciliationHandler handles daily financial reconciliation
type ReconciliationHandler struct {
	financeSvc financeService.FinanceService
	log        *slog.Logger
	subject    string
}

// NewReconciliationHandler creates a new handler for financial reconciliation
func NewReconciliationHandler(
	financeSvc financeService.FinanceService,
	log *slog.Logger,
	subject string,
) *ReconciliationHandler {
	return &ReconciliationHandler{
		financeSvc: financeSvc,
		log:        log,
		subject:    subject,
	}
}

// JobType returns the job type identifier
func (h *ReconciliationHandler) JobType() string {
	return financeJobs.ReconciliationJobType
}

// Subject returns the queue subject this handler listens to
func (h *ReconciliationHandler) Subject() string {
	return h.subject
}

// Handle processes the reconciliation job
func (h *ReconciliationHandler) Handle(ctx context.Context, data []byte) error {
	var job financeJobs.ReconciliationJob
	if err := json.Unmarshal(data, &job); err != nil {
		h.log.Error("failed to unmarshal ReconciliationJob", "error", err)
		return err
	}

	reconciliationTime := job.GetReconciliationTime()
	if h.log != nil {
		h.log.Info("starting daily financial reconciliation", "time", reconciliationTime)
	}

	// Run full reconciliation
	report, err := h.financeSvc.RunReconciliation(ctx)
	if err != nil {
		h.log.Error("reconciliation job failed", "error", err)
		return err
	}

	if h.log != nil {
		h.log.Info("reconciliation completed", "report_id", report.ID, "discrepancies", report.DiscrepanciesFound)

		if report.DiscrepanciesFound > 0 {
			h.log.Warn("reconciliation found discrepancies - admin review required", "count", report.DiscrepanciesFound)
		}
	}

	return nil
}
