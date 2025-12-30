package finance

import (
	"context"
	"encoding/json"
	financeService "hauslet/internal/modules/finance/service"
	financeJobs "hauslet/internal/queue/jobs/finance"

	"github.com/go-pkgz/lgr"
)

// ReconciliationHandler handles daily financial reconciliation
type ReconciliationHandler struct {
	financeSvc financeService.FinanceService
	log        *lgr.Logger
	subject    string
}

// NewReconciliationHandler creates a new handler for financial reconciliation
func NewReconciliationHandler(
	financeSvc financeService.FinanceService,
	log *lgr.Logger,
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
		h.log.Logf("ERROR failed to unmarshal ReconciliationJob: %v", err)
		return err
	}

	reconciliationTime := job.GetReconciliationTime()
	if h.log != nil {
		h.log.Logf("INFO starting daily financial reconciliation at %s", reconciliationTime)
	}

	// Run full reconciliation
	report, err := h.financeSvc.RunReconciliation(ctx)
	if err != nil {
		h.log.Logf("ERROR reconciliation job failed: %v", err)
		return err
	}

	if h.log != nil {
		h.log.Logf("INFO reconciliation completed: report_id=%s discrepancies=%d",
			report.ID, report.DiscrepanciesFound)

		if report.DiscrepanciesFound > 0 {
			h.log.Logf("WARN reconciliation found %d discrepancies - admin review required",
				report.DiscrepanciesFound)
		}
	}

	return nil
}
