package finance

import (
	"context"
	"encoding/json"
	financeService "hauslet/internal/modules/finance/service"
	financeJobs "hauslet/internal/queue/jobs/finance"
	"log/slog"
)

// PayoutProcessHandler handles automated payout processing
type PayoutProcessHandler struct {
	payoutSvc financeService.PayoutService
	log       *slog.Logger
	subject   string
}

// NewPayoutProcessHandler creates a new handler for processing payouts
func NewPayoutProcessHandler(
	payoutSvc financeService.PayoutService,
	log *slog.Logger,
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
		h.log.Error("failed to unmarshal ProcessPayoutsJob", "error", err)
		return err
	}

	processTime := job.GetProcessTime()
	if h.log != nil {
		h.log.Info("processing payouts job", "time", processTime)
	}

	if err := h.payoutSvc.ProcessDuePayouts(ctx); err != nil {
		h.log.Error("failed to process due payouts", "error", err)
		return err
	}

	if h.log != nil {
		h.log.Info("completed payout processing")
	}

	return nil
}

// DisbursementRetryHandler handles retrying failed disbursements
type DisbursementRetryHandler struct {
	payoutSvc financeService.PayoutService
	log       *slog.Logger
	subject   string
}

// NewDisbursementRetryHandler creates a new handler for retrying disbursements
func NewDisbursementRetryHandler(
	payoutSvc financeService.PayoutService,
	log *slog.Logger,
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
		h.log.Error("failed to unmarshal RetryDisbursementsJob", "error", err)
		return err
	}

	retryTime := job.GetRetryTime()
	if h.log != nil {
		h.log.Info("retrying failed disbursements", "time", retryTime)
	}

	if err := h.payoutSvc.RetryFailedDisbursements(ctx); err != nil {
		h.log.Error("failed to retry disbursements", "error", err)
		return err
	}

	if h.log != nil {
		h.log.Info("completed disbursement retries")
	}

	return nil
}
