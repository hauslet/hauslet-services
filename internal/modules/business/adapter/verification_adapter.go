package adapter

import (
	"context"
	"log/slog"
	"time"

	"hauslet/internal/modules/business/service"
	"hauslet/internal/modules/verification/port"

	"github.com/google/uuid"
)

// verificationAdapter implements port.BusinessAdapter
// This adapter allows the verification module to update business verification status.
type verificationAdapter struct {
	businessService service.BusinessService
	logger          *slog.Logger
}

// NewVerificationAdapter creates a new business adapter for verification callbacks.
func NewVerificationAdapter(businessService service.BusinessService, logger *slog.Logger) port.BusinessAdapter {
	return &verificationAdapter{
		businessService: businessService,
		logger:          logger,
	}
}

// MarkBusinessVerified implements port.BusinessAdapter
func (a *verificationAdapter) MarkBusinessVerified(ctx context.Context, businessID uuid.UUID, verifiedAt time.Time) error {
	a.logger.Info("marking business verified",
		"business_id", businessID.String(),
		"verified_at", verifiedAt,
	)

	err := a.businessService.SetVerificationStatus(ctx, businessID, true, &verifiedAt)
	if err != nil {
		a.logger.Error("failed to mark business verified",
			"business_id", businessID.String(),
			"error", err,
		)
		return err
	}

	a.logger.Info("business verification updated",
		"business_id", businessID.String(),
	)
	return nil
}

// Ensure verificationAdapter implements port.BusinessAdapter
var _ port.BusinessAdapter = (*verificationAdapter)(nil)
