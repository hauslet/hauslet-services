package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/promotions/service"
	promotionJob "hauslet/internal/queue/jobs/promotions"
)

// PromotionExpiryHandler expires promotions that have passed their expiry date.
type PromotionExpiryHandler struct {
	promotionSvc service.PromotionService
	log          *slog.Logger
	subject      string
}

// NewPromotionExpiryHandler constructs a promotion expiry handler.
func NewPromotionExpiryHandler(promotionSvc service.PromotionService, log *slog.Logger, subject string) *PromotionExpiryHandler {
	return &PromotionExpiryHandler{
		promotionSvc: promotionSvc,
		log:          log,
		subject:      subject,
	}
}

func (h *PromotionExpiryHandler) JobType() string {
	return promotionJob.PromotionExpiryJobType
}

func (h *PromotionExpiryHandler) Subject() string {
	return h.subject
}

func (h *PromotionExpiryHandler) Handle(ctx context.Context, data []byte) error {
	var job promotionJob.PromotionExpiryJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal promotion expiry job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid promotion expiry job: %w", err)
	}

	checkTime := job.GetCheckTime()
	h.log.Info("processing promotion expiry check", "check_time", checkTime)

	// Expire promotions
	if err := h.promotionSvc.ExpirePromotions(ctx); err != nil {
		return fmt.Errorf("failed to expire promotions: %w", err)
	}

	h.log.Info("promotion expiry check completed")
	return nil
}
