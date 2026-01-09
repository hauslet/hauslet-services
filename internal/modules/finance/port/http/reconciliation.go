package http

import (
	"hauslet/internal/modules/finance/domain"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ListReconciliationReports lists all reconciliation reports
func (h *HTTPHandler) ListReconciliationReports(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	limitParam := r.URL.Query().Get("limit")
	offsetParam := r.URL.Query().Get("offset")

	// Parse pagination
	limit := 50
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = min(l, 50)
		}
	}

	offset := 0
	if offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o > 0 {
			offset = o
		}
	}

	// Get reports from service
	reports, err := h.financeService.ListReconciliationReports(r.Context(), limit, offset)
	if err != nil {
		h.log.Error("failed to list reconciliation reports", "error", err)
		h.sendError(w, "Failed to retrieve reconciliation reports", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, reports, http.StatusOK)
}

// GetLatestReconciliation retrieves the most recent reconciliation report
func (h *HTTPHandler) GetLatestReconciliation(w http.ResponseWriter, r *http.Request) {
	report, err := h.financeService.GetLatestReconciliation(r.Context())
	if err != nil {
		if err == domain.ErrReconciliationNotFound {
			h.sendError(w, "No reconciliation reports found", http.StatusNotFound, "")
			return
		}
		h.log.Error("failed to get latest reconciliation", "error", err)
		h.sendError(w, "Failed to retrieve latest reconciliation", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, report, http.StatusOK)
}

// GetReconciliationReport retrieves a reconciliation report by ID
func (h *HTTPHandler) GetReconciliationReport(w http.ResponseWriter, r *http.Request) {
	// Extract report ID from URL
	reportIDStr := chi.URLParam(r, "id")
	if reportIDStr == "" {
		h.sendError(w, "Report ID is required", http.StatusBadRequest, "id")
		return
	}

	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		h.log.Error("invalid report ID", "report_id", reportIDStr, "error", err)
		h.sendError(w, "Invalid report ID", http.StatusBadRequest, "id")
		return
	}

	// Get report from service
	report, err := h.financeService.GetReconciliationReport(r.Context(), reportID)
	if err != nil {
		if err == domain.ErrReconciliationNotFound {
			h.sendError(w, "Reconciliation report not found", http.StatusNotFound, "")
			return
		}
		h.log.Error("failed to get reconciliation report", "report_id", reportID, "error", err)
		h.sendError(w, "Failed to retrieve reconciliation report", http.StatusInternalServerError, "")
		return
	}

	h.sendSuccess(w, report, http.StatusOK)
}

// GetReconciliationDiscrepancies lists discrepancies for a reconciliation report
func (h *HTTPHandler) GetReconciliationDiscrepancies(w http.ResponseWriter, r *http.Request) {
	// Extract report ID from URL
	reportIDStr := chi.URLParam(r, "id")
	if reportIDStr == "" {
		h.sendError(w, "Report ID is required", http.StatusBadRequest, "id")
		return
	}

	reportID, err := uuid.Parse(reportIDStr)
	if err != nil {
		h.log.Error("invalid report ID", "report_id", reportIDStr, "error", err)
		h.sendError(w, "Invalid report ID", http.StatusBadRequest, "id")
		return
	}

	// Extract severity filter from query params
	severityParam := r.URL.Query().Get("severity")
	var severity *domain.DiscrepancySeverity
	if severityParam != "" {
		s := domain.DiscrepancySeverity(severityParam)
		severity = &s
	}

	// Get report to access discrepancies
	report, err := h.financeService.GetReconciliationReport(r.Context(), reportID)
	if err != nil {
		if err == domain.ErrReconciliationNotFound {
			h.sendError(w, "Reconciliation report not found", http.StatusNotFound, "")
			return
		}
		h.log.Error("failed to get reconciliation report", "report_id", reportID, "error", err)
		h.sendError(w, "Failed to retrieve reconciliation report", http.StatusInternalServerError, "")
		return
	}

	// Filter discrepancies by severity if provided
	var discrepancies []domain.Discrepancy
	if severity != nil {
		for _, d := range report.Discrepancies {
			if d.Severity == *severity {
				discrepancies = append(discrepancies, d)
			}
		}
	} else {
		discrepancies = report.Discrepancies
	}

	h.sendSuccess(w, discrepancies, http.StatusOK)
}
