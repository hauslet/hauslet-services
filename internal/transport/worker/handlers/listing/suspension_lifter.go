package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	profileservice "hauslet/internal/modules/profile/service"
	listingjob "hauslet/internal/queue/jobs/listing"
)

// ListingSuspensionLifterHandler lifts expired listing suspensions.
type ListingSuspensionLifterHandler struct {
	hostPenaltySvc profileservice.HostPenaltyService
	notifier       profileservice.SuspensionNotifier
	log            *slog.Logger
	subject        string
}

// NewListingSuspensionLifterHandler constructs a suspension lifter handler.
func NewListingSuspensionLifterHandler(
	hostPenaltySvc profileservice.HostPenaltyService,
	notifier profileservice.SuspensionNotifier,
	log *slog.Logger,
	subject string,
) *ListingSuspensionLifterHandler {
	return &ListingSuspensionLifterHandler{
		hostPenaltySvc: hostPenaltySvc,
		notifier:       notifier,
		log:            log,
		subject:        subject,
	}
}

func (h *ListingSuspensionLifterHandler) JobType() string {
	return listingjob.ListingSuspensionLifterJobType
}

func (h *ListingSuspensionLifterHandler) Subject() string {
	return h.subject
}

func (h *ListingSuspensionLifterHandler) Handle(ctx context.Context, data []byte) error {
	var job listingjob.ListingSuspensionLifterJob
	if err := json.Unmarshal(data, &job); err != nil {
		return fmt.Errorf("unmarshal listing suspension lifter job: %w", err)
	}
	if err := job.Validate(); err != nil {
		return fmt.Errorf("invalid listing suspension lifter job: %w", err)
	}

	h.log.Info("processing listing suspension lifter job", "check_time", job.GetCheckTime())

	// Send notifications for suspensions ending soon (before lifting)
	if h.notifier != nil {
		notifiedCount, err := h.hostPenaltySvc.NotifySuspensionEndingSoon(ctx, h.notifier)
		if err != nil {
			h.log.Warn("failed to send suspension ending notifications", "error", err)
		} else if notifiedCount > 0 {
			h.log.Info("sent suspension ending notifications", "count", notifiedCount)
		}
	}

	// Lift expired suspensions
	if err := h.hostPenaltySvc.CheckAndLiftSuspensions(ctx); err != nil {
		return fmt.Errorf("failed to lift listing suspensions: %w", err)
	}

	h.log.Info("listing suspension lifter job completed")
	return nil
}
