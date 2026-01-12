package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"time"

	"github.com/google/uuid"
)

// RunReconciliation executes a full financial reconciliation

func (s *FinanceServiceImpl) RunReconciliation(ctx context.Context) (*domain.ReconciliationReport, error) {
	// Check if a reconciliation is already running
	runningReport, err := s.reconciliationRepo.GetRunningReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check for running reconciliation: %w", err)
	}
	if runningReport != nil {
		return nil, domain.ErrReconciliationRunning
	}

	// Create new report
	report := &domain.ReconciliationReport{
		ID:        uuid.New(),
		Status:    domain.ReconciliationStatusRunning,
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save report with "running" status
	reportSchema := domain.MapReconciliationReportToSchema(report)
	if err := s.reconciliationRepo.CreateReport(ctx, reportSchema); err != nil {
		return nil, fmt.Errorf("failed to create reconciliation report: %w", err)
	}

	// Initialize result
	result := &domain.ReconciliationResult{}

	// Run validation checks
	s.log.Info("starting reconciliation",
		"report_id", report.ID,
	)

	// 1. Validate ledger balance (debit = credit)
	s.log.Info("reconciliation: validating ledger balance")
	ledgerDiscrepancies, err := s.ValidateLedgerBalance(ctx)
	if err != nil {
		s.log.Error("reconciliation failed during ledger validation",
			"report_id", report.ID,
			"error", err,
		)
		return s.failReconciliation(ctx, report, err)
	}
	for _, d := range ledgerDiscrepancies {
		d.ReportID = report.ID
		result.AddDiscrepancy(d)
	}

	// 2. Validate wallet balances
	s.log.Info("reconciliation: validating wallet balances")
	walletDiscrepancies, err := s.ValidateWalletBalance(ctx)
	if err != nil {
		s.log.Error("reconciliation failed during wallet validation",
			"report_id", report.ID,
			"error", err,
		)
		return s.failReconciliation(ctx, report, err)
	}
	for _, d := range walletDiscrepancies {
		d.ReportID = report.ID
		result.AddDiscrepancy(d)
	}

	// TODO: 3. Provider reconciliation (Paystack settlement matching)
	// This will require:
	// - Fetching settlement reports from Paystack API (implement in internal/platform/payment)
	// - Matching our disbursements to Paystack transfers
	// - Detecting missing or duplicate transactions
	s.log.Warn("reconciliation: provider reconciliation not yet implemented")

	// Save all discrepancies
	if len(result.Discrepancies) == 0 {
		s.log.Info("reconciliation passed with no discrepancies",
			"report_id", report.ID,
		)
	} else {
		s.log.Info("reconciliation: found discrepancies",
			"count", len(result.Discrepancies),
		)
	}

	for _, discrepancy := range result.Discrepancies {
		discrepancy.ID = uuid.New()
		discrepancy.CreatedAt = time.Now()
		discrepancySchema := domain.MapDiscrepancyToSchema(&discrepancy)
		if err := s.reconciliationRepo.CreateDiscrepancy(ctx, discrepancySchema); err != nil {
			s.log.Error("failed to save discrepancy",
				"error", err,
			)
			// Continue saving other discrepancies
		}
	}

	// Complete the report
	completedAt := time.Now()
	report.CompletedAt = &completedAt
	report.Status = domain.ReconciliationStatusCompleted
	report.TotalWalletsChecked = result.WalletsChecked
	report.TotalTransactionsChecked = result.TransactionsChecked
	report.DiscrepanciesFound = len(result.Discrepancies)
	report.Summary = result.GetSummary()
	report.Discrepancies = result.Discrepancies
	report.UpdatedAt = time.Now()

	// Update report
	reportSchema = domain.MapReconciliationReportToSchema(report)
	if err := s.reconciliationRepo.UpdateReport(ctx, reportSchema); err != nil {
		s.log.Error("failed to update reconciliation report",
			"report_id", report.ID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to update reconciliation report: %w", err)
	}

	s.log.Info("reconciliation completed",
		"report_id", report.ID,
		"discrepancies", len(result.Discrepancies),
	)

	// Send alert email to admins if discrepancies found
	if result.HasIssues() {
		s.log.Warn("reconciliation found issues - sending admin notifications",
			"discrepancies", len(result.Discrepancies),
		)

		// Check if alerts are enabled in config
		if s.platformConfig.Reconciliation.SendAlerts {
			// Get admin emails via adapter
			adminEmails, err := s.adminProvider.GetAdminEmails(ctx)
			if err != nil {
				s.log.Error("failed to get admin emails for reconciliation alert",
					"error", err,
				)
			} else if len(adminEmails) > 0 {
				// Filter discrepancies by minimum severity
				filteredReport := filterReportBySeverity(report, s.platformConfig.Reconciliation.MinSeverity)

				if len(filteredReport.Discrepancies) > 0 {
					if err := s.notificationSvc.SendReconciliationAlert(ctx, adminEmails, filteredReport); err != nil {
						s.log.Error("failed to send reconciliation alert",
							"error", err,
						)
					} else {
						s.log.Info("reconciliation alerts sent",
							"recipient_count", len(adminEmails),
						)
					}
				}
			}
		}
	}

	return report, nil
}

// ValidateLedgerBalance checks that all transactions have balanced ledger entries
func (s *FinanceServiceImpl) ValidateLedgerBalance(ctx context.Context) ([]domain.Discrepancy, error) {
	var discrepancies []domain.Discrepancy

	// Find imbalanced transactions using repository
	imbalances, err := s.ledgerRepo.FindImbalancedTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger imbalances: %w", err)
	}

	// Create discrepancies for each imbalance
	for _, imbalance := range imbalances {
		discrepancies = append(discrepancies, domain.Discrepancy{
			Type:          domain.DiscrepancyTypeLedgerImbalance,
			Severity:      domain.DiscrepancySeverityCritical,
			TransactionID: &imbalance.TransactionID,
			Description:   fmt.Sprintf("Transaction has imbalanced ledger entries: debit=%d credit=%d", imbalance.TotalDebit, imbalance.TotalCredit),
			ExpectedValue: &imbalance.TotalDebit,
			ActualValue:   &imbalance.TotalCredit,
			Details: map[string]any{
				"transaction_id": imbalance.TransactionID.String(),
				"total_debit":    imbalance.TotalDebit,
				"total_credit":   imbalance.TotalCredit,
			},
		})
	}

	if len(discrepancies) > 0 {
		s.log.Info("ledger validation: found imbalanced transactions",
			"count", len(discrepancies),
		)
	} else {
		s.log.Info("ledger validation: all transactions balanced")
	}

	return discrepancies, nil
}

// ValidateWalletBalance checks that wallet balances match sum of ledger entries
func (s *FinanceServiceImpl) ValidateWalletBalance(ctx context.Context) ([]domain.Discrepancy, error) {
	var discrepancies []domain.Discrepancy

	// Find wallet balance mismatches using repository
	mismatches, err := s.walletRepo.FindBalanceMismatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet balance mismatches: %w", err)
	}

	// No mismatches found — validation passed
	if len(mismatches) == 0 {
		if s.log != nil {
			s.log.Info("wallet validation passed: no wallet balance mismatches found")
		}
		return nil, nil
	}

	// Create discrepancies for each mismatch
	for _, mismatch := range mismatches {
		severity := domain.DiscrepancySeverityHigh
		if mismatch.ActualBalance == 0 && mismatch.LedgerBalance != 0 {
			// Orphaned wallet with ledger entries but zero balance
			severity = domain.DiscrepancySeverityCritical
		}

		discrepancies = append(discrepancies, domain.Discrepancy{
			Type:     domain.DiscrepancyTypeWalletMismatch,
			Severity: severity,
			WalletID: &mismatch.WalletID,
			Description: fmt.Sprintf(
				"Wallet balance mismatch: actual=%d ledger_sum=%d currency=%s",
				mismatch.ActualBalance,
				mismatch.LedgerBalance,
				mismatch.Currency,
			),
			ExpectedValue: &mismatch.LedgerBalance,
			ActualValue:   &mismatch.ActualBalance,
			Details: map[string]interface{}{
				"wallet_id":      mismatch.WalletID.String(),
				"actual_balance": mismatch.ActualBalance,
				"ledger_balance": mismatch.LedgerBalance,
				"currency":       mismatch.Currency,
				"difference":     mismatch.ActualBalance - mismatch.LedgerBalance,
			},
		})
	}

	if s.log != nil {
		s.log.Info(
			"wallet validation: wallet balance mismatches detected",
			"count", len(discrepancies),
		)
	}

	return discrepancies, nil
}

// GetReconciliationReport retrieves a reconciliation report by ID
func (s *FinanceServiceImpl) GetReconciliationReport(ctx context.Context, reportID uuid.UUID) (*domain.ReconciliationReport, error) {
	reportSchema, err := s.reconciliationRepo.GetReportByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reconciliation report: %w", err)
	}
	if reportSchema == nil {
		return nil, domain.ErrReconciliationNotFound
	}

	report := domain.MapReconciliationReportFromSchema(reportSchema)

	// Load discrepancies
	discrepancySchemas, err := s.reconciliationRepo.ListDiscrepanciesByReport(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to load discrepancies: %w", err)
	}

	discrepancies := make([]domain.Discrepancy, len(discrepancySchemas))
	for i, ds := range discrepancySchemas {
		discrepancies[i] = *domain.MapDiscrepancyFromSchema(ds)
	}
	report.Discrepancies = discrepancies

	return report, nil
}

// ListReconciliationReports lists reconciliation reports
func (s *FinanceServiceImpl) ListReconciliationReports(ctx context.Context, limit, offset int) ([]*domain.ReconciliationReport, error) {
	reportSchemas, err := s.reconciliationRepo.ListReports(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list reconciliation reports: %w", err)
	}

	reports := make([]*domain.ReconciliationReport, len(reportSchemas))
	for i, rs := range reportSchemas {
		reports[i] = domain.MapReconciliationReportFromSchema(rs)
	}

	return reports, nil
}

// GetLatestReconciliation retrieves the most recent reconciliation report
func (s *FinanceServiceImpl) GetLatestReconciliation(ctx context.Context) (*domain.ReconciliationReport, error) {
	reportSchema, err := s.reconciliationRepo.GetLatestReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest reconciliation: %w", err)
	}
	if reportSchema == nil {
		return nil, domain.ErrReconciliationNotFound
	}

	report := domain.MapReconciliationReportFromSchema(reportSchema)

	// Load discrepancies
	discrepancySchemas, err := s.reconciliationRepo.ListDiscrepanciesByReport(ctx, report.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load discrepancies: %w", err)
	}

	discrepancies := make([]domain.Discrepancy, len(discrepancySchemas))
	for i, ds := range discrepancySchemas {
		discrepancies[i] = *domain.MapDiscrepancyFromSchema(ds)
	}
	report.Discrepancies = discrepancies

	return report, nil
}

// failReconciliation marks a reconciliation as failed
func (s *FinanceServiceImpl) failReconciliation(ctx context.Context, report *domain.ReconciliationReport, err error) (*domain.ReconciliationReport, error) {
	errorMsg := err.Error()
	completedAt := time.Now()

	report.Status = domain.ReconciliationStatusFailed
	report.CompletedAt = &completedAt
	report.ErrorMessage = &errorMsg
	report.Summary = "Reconciliation failed due to error"
	report.UpdatedAt = time.Now()

	reportSchema := domain.MapReconciliationReportToSchema(report)
	if updateErr := s.reconciliationRepo.UpdateReport(ctx, reportSchema); updateErr != nil {
		s.log.Error("failed to update failed reconciliation report",
			"report_id", report.ID,
			"error", updateErr,
		)
	}

	return report, err
}

// filterReportBySeverity filters discrepancies by minimum severity threshold
func filterReportBySeverity(report *domain.ReconciliationReport, minSeverity string) *domain.ReconciliationReport {
	if minSeverity == "" || minSeverity == "low" {
		return report
	}

	severityOrder := map[domain.DiscrepancySeverity]int{
		domain.DiscrepancySeverityLow:      0,
		domain.DiscrepancySeverityMedium:   1,
		domain.DiscrepancySeverityHigh:     2,
		domain.DiscrepancySeverityCritical: 3,
	}

	minLevel := severityOrder[domain.DiscrepancySeverity(minSeverity)]

	filtered := &domain.ReconciliationReport{
		ID:                       report.ID,
		Status:                   report.Status,
		StartedAt:                report.StartedAt,
		CompletedAt:              report.CompletedAt,
		TotalWalletsChecked:      report.TotalWalletsChecked,
		TotalTransactionsChecked: report.TotalTransactionsChecked,
		Summary:                  report.Summary,
		Discrepancies:            make([]domain.Discrepancy, 0),
	}

	for _, d := range report.Discrepancies {
		if severityOrder[d.Severity] >= minLevel {
			filtered.Discrepancies = append(filtered.Discrepancies, d)
		}
	}

	filtered.DiscrepanciesFound = len(filtered.Discrepancies)
	return filtered
}
