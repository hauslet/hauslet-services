package hooks

import (
	"context"
	businessservice "hauslet/internal/modules/business/service"

	"github.com/google/uuid"
)

// PaymentsBusinessAdapter exposes minimal business data to the payments module.
type PaymentsBusinessAdapter struct {
	svc businessservice.BusinessService
}

// NewPaymentsBusinessAdapter creates the adapter for payments notifications.
func NewPaymentsBusinessAdapter(svc businessservice.BusinessService) *PaymentsBusinessAdapter {
	return &PaymentsBusinessAdapter{svc: svc}
}

// GetBusinessContactEmail returns the billing or primary email for a business.
func (a *PaymentsBusinessAdapter) GetBusinessContactEmail(ctx context.Context, businessID uuid.UUID) (string, error) {
	business, err := a.svc.GetBusiness(ctx, businessID)
	if err != nil {
		return "", err
	}
	if business == nil {
		return "", nil
	}
	if business.BillingEmail != nil && *business.BillingEmail != "" {
		return *business.BillingEmail, nil
	}
	return business.Email, nil
}
