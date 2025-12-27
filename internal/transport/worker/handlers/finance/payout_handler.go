package finance

import (
	"context"
	"encoding/json"
	financeService "hauslet/internal/modules/finance/service"
	financeJobs "hauslet/internal/queue/jobs/finance"

	"github.com/go-pkgz/lgr"
)

// PayoutProcessHandler handles automated payout processing
type PayoutProcessHandler struct {
	payoutSvc financeService.PayoutService
	log       *lgr.Logger
	subject   string
}

// NewPayoutProcessHandler creates a new handler for processing payouts
func NewPayoutProcessHandler(
	payoutSvc financeService.PayoutService,
	log *lgr.Logger,
	subject string,
) *PayoutProcessHandler {
	return &PayoutProcessHandler{
		payoutSvc: payoutSvc,
		log:       log,
		subject:   subject,
	}
}

// JobType returns the job type identifier
func (h *PayoutProcessHandler) JobType() string {
	return financeJobs.ProcessPayoutsJobType
}

// Subject returns the queue subject this handler listens to
func (h *PayoutProcessHandler) Subject() string {
	return h.subject
}

// Handle processes the payout job
func (h *PayoutProcessHandler) Handle(ctx context.Context, data []byte) error {
	var job financeJobs.ProcessPayoutsJob
	if err := json.Unmarshal(data, &job); err != nil {
		h.log.Logf("ERROR failed to unmarshal ProcessPayoutsJob: %v", err)
		return err
	}

	if h.log != nil {
		h.log.Logf("INFO processing payouts job at %s", job.ProcessTime)
	}

	if err := h.payoutSvc.ProcessDuePayouts(ctx); err != nil {
		h.log.Logf("ERROR failed to process due payouts: %v", err)
		return err
	}

	if h.log != nil {
		h.log.Logf("INFO completed payout processing")
	}

	return nil
}

// DisbursementRetryHandler handles retrying failed disbursements
type DisbursementRetryHandler struct {
	payoutSvc financeService.PayoutService
	log       *lgr.Logger
	subject   string
}

// NewDisbursementRetryHandler creates a new handler for retrying disbursements
func NewDisbursementRetryHandler(
	payoutSvc financeService.PayoutService,
	log *lgr.Logger,
	subject string,
) *DisbursementRetryHandler {
	return &DisbursementRetryHandler{
		payoutSvc: payoutSvc,
		log:       log,
		subject:   subject,
	}
}

// JobType returns the job type identifier
func (h *DisbursementRetryHandler) JobType() string {
	return financeJobs.RetryDisbursementsJobType
}

// Subject returns the queue subject this handler listens to
func (h *DisbursementRetryHandler) Subject() string {
	return h.subject
}

// Handle processes the retry job
func (h *DisbursementRetryHandler) Handle(ctx context.Context, data []byte) error {
	var job financeJobs.RetryDisbursementsJob
	if err := json.Unmarshal(data, &job); err != nil {
		h.log.Logf("ERROR failed to unmarshal RetryDisbursementsJob: %v", err)
		return err
	}

	if h.log != nil {
		h.log.Logf("INFO retrying failed disbursements at %s", job.RetryTime)
	}

	if err := h.payoutSvc.RetryFailedDisbursements(ctx); err != nil {
		h.log.Logf("ERROR failed to retry disbursements: %v", err)
		return err
	}

	if h.log != nil {
		h.log.Logf("INFO completed disbursement retries")
	}

	return nil
}
