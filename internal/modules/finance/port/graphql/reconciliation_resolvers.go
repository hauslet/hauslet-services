package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"

	"github.com/google/uuid"
)

// ============================================================================
// Reconciliation Query Resolvers
// ============================================================================

// ReconciliationReport retrieves a reconciliation report by ID (admin only)
func (r *Resolver) ReconciliationReport(ctx context.Context, id string) (*domain.ReconciliationReport, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	reportID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid reconciliation report ID %s: %v", id, err)
		return nil, fmt.Errorf("invalid report ID")
	}

	report, err := r.financeService.GetReconciliationReport(ctx, reportID)
	if err != nil {
		if err == domain.ErrReconciliationNotFound {
			return nil, nil
		}
		r.log.Error("failed to get reconciliation report %s: %v", id, err)
		return nil, err
	}

	return report, nil
}

// ReconciliationReports lists all reconciliation reports (admin only)
func (r *Resolver) ReconciliationReports(
	ctx context.Context,
	limit, offset *int,
) ([]*domain.ReconciliationReport, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	l := 50 // default limit
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0 // default offset
	if offset != nil && *offset > 0 {
		o = *offset
	}

	reports, err := r.financeService.ListReconciliationReports(ctx, l, o)
	if err != nil {
		r.log.Error("failed to list reconciliation reports: %v", err)
		return nil, err
	}

	return reports, nil
}

// LatestReconciliation retrieves the most recent reconciliation report (admin only)
func (r *Resolver) LatestReconciliation(ctx context.Context) (*domain.ReconciliationReport, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	report, err := r.financeService.GetLatestReconciliation(ctx)
	if err != nil {
		if err == domain.ErrReconciliationNotFound {
			return nil, nil
		}
		r.log.Error("failed to get latest reconciliation: %v", err)
		return nil, err
	}

	return report, nil
}

// ReconciliationDiscrepancies lists discrepancies for a report with optional severity filter (admin only)
func (r *Resolver) ReconciliationDiscrepancies(
	ctx context.Context,
	reportID string,
	severity *domain.DiscrepancySeverity,
) ([]*domain.Discrepancy, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	rid, err := uuid.Parse(reportID)
	if err != nil {
		r.log.Error("invalid report ID %s: %v", reportID, err)
		return nil, fmt.Errorf("invalid report ID")
	}

	// First verify the report exists
	report, err := r.financeService.GetReconciliationReport(ctx, rid)
	if err != nil {
		if err == domain.ErrReconciliationNotFound {
			return nil, fmt.Errorf("report not found")
		}
		r.log.Error("failed to get reconciliation report %s: %v", reportID, err)
		return nil, err
	}

	// If severity filter is provided, filter discrepancies by severity
	// Otherwise return all discrepancies from the report
	var discrepancies []*domain.Discrepancy
	if severity != nil {
		for _, d := range report.Discrepancies {
			if d.Severity == *severity {
				discrepancy := d // Create a copy to avoid pointer issues
				discrepancies = append(discrepancies, &discrepancy)
			}
		}
	} else {
		// Return all discrepancies
		for _, d := range report.Discrepancies {
			discrepancy := d // Create a copy to avoid pointer issues
			discrepancies = append(discrepancies, &discrepancy)
		}
	}

	return discrepancies, nil
}
