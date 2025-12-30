package repository

import (
	"context"
	"hauslet/internal/modules/finance/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReconciliationRepositoryImpl implements ReconciliationRepository
type ReconciliationRepositoryImpl struct {
	db *gorm.DB
}

// NewReconciliationRepository creates a new reconciliation repository
func NewReconciliationRepository(db *gorm.DB) ReconciliationRepository {
	return &ReconciliationRepositoryImpl{db: db}
}

// WithTx returns a new repository instance using the provided transaction
func (r *ReconciliationRepositoryImpl) WithTx(tx *gorm.DB) ReconciliationRepository {
	return &ReconciliationRepositoryImpl{db: tx}
}

// CreateReport creates a new reconciliation report
func (r *ReconciliationRepositoryImpl) CreateReport(ctx context.Context, report *schema.ReconciliationReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}

// GetReportByID retrieves a reconciliation report by ID
func (r *ReconciliationRepositoryImpl) GetReportByID(ctx context.Context, id uuid.UUID) (*schema.ReconciliationReport, error) {
	var report schema.ReconciliationReport
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&report).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

// UpdateReport updates an existing reconciliation report
func (r *ReconciliationRepositoryImpl) UpdateReport(ctx context.Context, report *schema.ReconciliationReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}

// ListReports lists reconciliation reports with pagination
func (r *ReconciliationRepositoryImpl) ListReports(ctx context.Context, limit, offset int) ([]*schema.ReconciliationReport, error) {
	var reports []*schema.ReconciliationReport
	err := r.db.WithContext(ctx).
		Order("started_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}

// GetLatestReport retrieves the most recent reconciliation report
func (r *ReconciliationRepositoryImpl) GetLatestReport(ctx context.Context) (*schema.ReconciliationReport, error) {
	var report schema.ReconciliationReport
	err := r.db.WithContext(ctx).
		Order("started_at DESC").
		First(&report).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

// GetRunningReport retrieves a currently running reconciliation report if any
func (r *ReconciliationRepositoryImpl) GetRunningReport(ctx context.Context) (*schema.ReconciliationReport, error) {
	var report schema.ReconciliationReport
	err := r.db.WithContext(ctx).
		Where("status = ?", "running").
		First(&report).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

// CreateDiscrepancy creates a new discrepancy
func (r *ReconciliationRepositoryImpl) CreateDiscrepancy(ctx context.Context, discrepancy *schema.Discrepancy) error {
	return r.db.WithContext(ctx).Create(discrepancy).Error
}

// ListDiscrepanciesByReport lists all discrepancies for a specific report
func (r *ReconciliationRepositoryImpl) ListDiscrepanciesByReport(ctx context.Context, reportID uuid.UUID) ([]*schema.Discrepancy, error) {
	var discrepancies []*schema.Discrepancy
	err := r.db.WithContext(ctx).
		Where("report_id = ?", reportID).
		Order("severity DESC, created_at ASC").
		Find(&discrepancies).Error
	if err != nil {
		return nil, err
	}
	return discrepancies, nil
}

// ListDiscrepanciesBySeverity lists discrepancies for a report filtered by severity
func (r *ReconciliationRepositoryImpl) ListDiscrepanciesBySeverity(ctx context.Context, reportID uuid.UUID, severity string) ([]*schema.Discrepancy, error) {
	var discrepancies []*schema.Discrepancy
	err := r.db.WithContext(ctx).
		Where("report_id = ? AND severity = ?", reportID, severity).
		Order("created_at ASC").
		Find(&discrepancies).Error
	if err != nil {
		return nil, err
	}
	return discrepancies, nil
}
