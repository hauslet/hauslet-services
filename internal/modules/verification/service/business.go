package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/verification/domain"
)

// =======================
// Business Verification
// =======================

func (s *verificationService) SubmitBusinessVerification(ctx context.Context, req SubmitBusinessVerificationRequest) (*SubmitVerificationResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Ensure session is business verification type
	if session.Type != domain.VerificationBusiness {
		return nil, fmt.Errorf("session is not business verification type")
	}

	// Check if session can accept new attempt
	if err := session.CanSubmitAttempt(); err != nil {
		return nil, err
	}

	// For Nigerian businesses, attempt automated CAC lookup first
	businessData, err := session.GetBusinessData()
	if err != nil {
		return nil, err
	}

	if strings.ToUpper(session.Country) == "NG" && businessData.RegistrationNumber != "" {
		cacResult, err := s.attemptCACVerification(ctx, session, businessData)
		if err == nil && cacResult != nil {
			return cacResult, nil
		}
		// If CAC lookup fails, fall through to manual review
		if err != nil {
			s.logger.Warn("CAC automated lookup failed, falling back to manual review",
				"session_id", session.ID,
				"rc_number", businessData.RegistrationNumber,
				"error", err,
			)
		}
	}

	// Build evidence items for manual review
	evidenceItems := []ManualReviewEvidence{
		{
			Type:     domain.EvidenceBusinessRegistration,
			Data:     req.RegistrationDocument,
			MimeType: "application/pdf",
		},
	}

	if req.TaxIDDocument != nil {
		evidenceItems = append(evidenceItems, ManualReviewEvidence{
			Type:     domain.EvidenceBusinessTaxID,
			Data:     *req.TaxIDDocument,
			MimeType: "application/pdf",
		})
	}

	if req.BusinessLicenseDocument != nil {
		evidenceItems = append(evidenceItems, ManualReviewEvidence{
			Type:     domain.EvidenceBusinessLicense,
			Data:     *req.BusinessLicenseDocument,
			MimeType: "application/pdf",
		})
	}

	return s.submitManualReview(ctx, session, req.UserID, evidenceItems, req.IPAddress)
}

// attemptCACVerification performs an automated CAC lookup for Nigerian businesses.
// Returns a completed verification response if the business passes automated checks,
// or nil if manual review is still needed.
func (s *verificationService) attemptCACVerification(
	ctx context.Context,
	session *domain.VerificationSession,
	businessData *domain.BusinessData,
) (*SubmitVerificationResponse, error) {
	// Map business type to Dojah's company_type enum
	companyType := mapBusinessTypeToCACType(businessData.BusinessType)

	// Call Dojah CAC lookup via KYC client
	cacResp, err := s.kycClient.VerifyBusiness(ctx, session.Country, businessData.RegistrationNumber, companyType)
	if err != nil {
		return nil, fmt.Errorf("CAC lookup failed: %w", err)
	}

	if !cacResp.Found {
		// Business not found — reject automatically
		attempt, _ := domain.NewVerificationAttempt(session.ID, "dojah")
		if attempt != nil {
			attempt.Status = domain.AttemptFailed
			_ = s.repo.CreateAttempt(ctx, attempt)
		}

		reason := domain.RejectionOther
		notes := "Business not found in CAC registry"
		_ = session.Reject(reason, &notes)
		_ = s.repo.UpdateSession(ctx, session)

		return &SubmitVerificationResponse{
			Attempt:      attempt,
			Session:      session,
			ProviderName: "dojah",
			Status:       "rejected",
			Message:      "Business not found in CAC registry",
		}, nil
	}

	// Cross-validate business details
	nameMatch := strings.EqualFold(
		strings.TrimSpace(cacResp.CompanyName),
		strings.TrimSpace(businessData.BusinessName),
	)

	isActive := strings.EqualFold(cacResp.Status, "active")

	// Create attempt
	attempt, err := domain.NewVerificationAttempt(session.ID, "dojah")
	if err != nil {
		return nil, err
	}

	if err := session.StartAttempt(); err != nil {
		return nil, err
	}

	if nameMatch && isActive {
		// Automated approval
		result := domain.VerificationResult{
			Success:       true,
			ProviderName:  "dojah",
			ProviderRefID: fmt.Sprintf("CAC-%s", businessData.RegistrationNumber),
			ProcessedAt:   time.Now(),
			RawResponse: map[string]any{
				"company_name":     cacResp.CompanyName,
				"rc_number":        cacResp.RCNumber,
				"status":           cacResp.Status,
				"company_type":     cacResp.CompanyType,
				"address":          cacResp.Address,
				"registered_at":    cacResp.DateOfRegistration,
				"affiliates_count": len(cacResp.Affiliates),
			},
		}

		if err := attempt.Complete(result); err != nil {
			return nil, err
		}
		if err := session.Approve(); err != nil {
			return nil, err
		}
	} else {
		// Data mismatch or inactive — reject with details
		var mismatchDetails []string
		if !nameMatch {
			mismatchDetails = append(mismatchDetails,
				fmt.Sprintf("Name mismatch: submitted '%s', CAC has '%s'",
					businessData.BusinessName, cacResp.CompanyName))
		}
		if !isActive {
			mismatchDetails = append(mismatchDetails,
				fmt.Sprintf("Business status is '%s' (not Active)", cacResp.Status))
		}

		reason := domain.RejectionOther
		notes := strings.Join(mismatchDetails, "; ")

		result := domain.VerificationResult{
			Success:         false,
			ProviderName:    "dojah",
			ProviderRefID:   fmt.Sprintf("CAC-%s", businessData.RegistrationNumber),
			ProcessedAt:     time.Now(),
			RejectionReason: &reason,
			RejectionNotes:  &notes,
		}
		if err := attempt.Complete(result); err != nil {
			return nil, err
		}
		if err := session.Reject(reason, &notes); err != nil {
			return nil, err
		}
	}

	// Persist
	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		return nil, fmt.Errorf("failed to save attempt: %w", err)
	}
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// If approved, notify business module
	if session.Status == domain.SessionApproved {
		s.notifyVerificationSuccess(ctx, session)
	}

	s.logger.Info("CAC automated verification completed",
		"session_id", session.ID,
		"rc_number", businessData.RegistrationNumber,
		"cac_name", cacResp.CompanyName,
		"status", session.Status,
	)

	return &SubmitVerificationResponse{
		Attempt:      attempt,
		Session:      session,
		ProviderName: "dojah",
		Status:       string(session.Status),
		Message:      cacResp.Message,
	}, nil
}

// mapBusinessTypeToCACType maps internal business type to Dojah CAC company_type enum
func mapBusinessTypeToCACType(businessType string) string {
	switch strings.ToLower(businessType) {
	case "sole_proprietorship", "sole_proprietor":
		return "BUSINESS_NAME"
	case "llc", "limited_liability", "limited_company":
		return "COMPANY"
	case "ngo", "nonprofit", "trust":
		return "INCORPORATED_TRUSTEES"
	case "lp", "limited_partnership":
		return "LIMITED_PARTNERSHIP"
	case "llp", "limited_liability_partnership":
		return "LIMITED_LIABILITY_PARTNERSHIP"
	default:
		return "BUSINESS_NAME"
	}
}
