package leads

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"hauslet/internal/modules/leads/service"
	"hauslet/internal/queue/jobs/leads"

	"github.com/google/uuid"
)

type QualificationHandler struct {
	leadService service.LeadService
	log         *slog.Logger
	subject     string
}

func NewQualificationHandler(leadService service.LeadService, log *slog.Logger, subject string) *QualificationHandler {
	return &QualificationHandler{
		leadService: leadService,
		log:         log,
		subject:     subject,
	}
}

func (h *QualificationHandler) Subject() string {
	return h.subject
}

func (h *QualificationHandler) JobType() string {
	return leads.QualificationJobType
}

func (h *QualificationHandler) Handle(ctx context.Context, payload []byte) error {
	var job leads.QualificationJobPayload
	if err := json.Unmarshal(payload, &job); err != nil {
		return fmt.Errorf("failed to unmarshal qualification job: %w", err)
	}

	h.log.Info("processing lead qualification job", "lead_id", job.LeadID)

	if job.LeadID == uuid.Nil {
		h.log.Warn("skipping lead qualification with invalid ID")
		return nil
	}

	if err := h.leadService.QualifyLead(ctx, job.LeadID); err != nil {
		h.log.Error("failed to qualify lead", "lead_id", job.LeadID, "error", err)
		return err
	}

	return nil
}
