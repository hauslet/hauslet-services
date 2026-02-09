package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/platform/kyc"
	"hauslet/internal/platform/ratelimit"
)

// =======================
// Helper Methods
// =======================

func (s *verificationService) checkRateLimits(ctx context.Context, session *domain.VerificationSession, ipAddress *string) error {
	// Get rate limit config from YAML
	rateLimitCfg := s.config.YAML.RateLimit.Verification

	// Parse window durations
	userWindow, _ := time.ParseDuration(rateLimitCfg.User.Window)
	countryWindow, _ := time.ParseDuration(rateLimitCfg.Country.Window)
	ipWindow, _ := time.ParseDuration(rateLimitCfg.IP.Window)
	phoneWindow, _ := time.ParseDuration(rateLimitCfg.Phone.Window)

	// Build rate limit keys
	keys := []ratelimit.LimitKey{
		{
			Type:   ratelimit.KeyTypeUser,
			Value:  session.UserID.String(),
			Limit:  rateLimitCfg.User.Limit,
			Window: userWindow,
		},
		{
			Type:   ratelimit.KeyTypeCountry,
			Value:  session.Country,
			Limit:  rateLimitCfg.Country.Limit,
			Window: countryWindow,
		},
	}

	if ipAddress != nil {
		keys = append(keys, ratelimit.LimitKey{
			Type:   ratelimit.KeyTypeIP,
			Value:  *ipAddress,
			Limit:  rateLimitCfg.IP.Limit,
			Window: ipWindow,
		})
	}

	// Add phone number rate limit based on verification type
	var phoneNumber *string
	switch session.Type {
	case domain.VerificationIdentity:
		if identityData, err := session.GetIdentityData(); err == nil {
			phoneNumber = identityData.ApplicantInfo.PhoneNumber
		}
	case domain.VerificationPhone:
		if phoneData, err := session.GetPhoneData(); err == nil {
			phoneNumber = &phoneData.PhoneNumber
		}
	}

	if phoneNumber != nil && *phoneNumber != "" {
		keys = append(keys, ratelimit.LimitKey{
			Type:   ratelimit.KeyTypePhone,
			Value:  *phoneNumber,
			Limit:  rateLimitCfg.Phone.Limit,
			Window: phoneWindow,
		})
	}

	// Check all rate limits atomically
	results, err := s.rateLimiter.CheckMultiple(ctx, keys...)
	if err != nil {
		s.logger.Error("rate limit check failed", "error", err)
		// Don't block on rate limit errors, just log
		return nil
	}

	// Check if any limit was exceeded
	for _, result := range results {
		if result.IsRateLimited() {
			s.logger.Warn("rate limit exceeded",
				"key_type", result.Key.Type,
				"key_value", result.Key.Value,
				"current", result.Current,
				"limit", result.Limit,
			)
			return domain.ErrRateLimitExceeded
		}
	}

	return nil
}

func (s *verificationService) buildKYCRequest(session *domain.VerificationSession, req SubmitIdentityVerificationRequest) kyc.VerificationRequest {
	// Extract identity data from session
	identityData, err := session.GetIdentityData()
	if err != nil {
		s.logger.Error("failed to get identity data", "error", err)
		// Return empty request if we can't get identity data
		return kyc.VerificationRequest{}
	}

	kycReq := kyc.VerificationRequest{
		UserID:        session.UserID.String(),
		Country:       session.Country,
		DocumentType:  kyc.DocumentType(req.DocumentType.String()),
		SelfieImage:   req.SelfieImage,
		DocumentImage: req.DocumentImage,
		FirstName:     identityData.ApplicantInfo.FirstName,
		LastName:      identityData.ApplicantInfo.LastName,
		DateOfBirth:   &identityData.ApplicantInfo.DateOfBirth,
		Metadata: map[string]string{
			"session_id": session.ID.String(),
			"tier":       session.Tier.String(),
		},
	}

	if req.DocumentNumber != nil {
		kycReq.DocumentNumber = *req.DocumentNumber
	}

	return kycReq
}

func (s *verificationService) mapKYCResponseToResult(resp *kyc.VerificationResponse) domain.VerificationResult {
	result := domain.VerificationResult{
		Success:       resp.Success && resp.Status == kyc.StatusApproved,
		ProviderName:  resp.Provider,
		ProviderRefID: resp.ProviderRef,
		ProcessedAt:   time.Now(),
		RawResponse:   make(map[string]any),
	}

	if resp.QualityScore != nil {
		score := *resp.QualityScore * 100 // Convert 0-1 to 0-100
		result.Score = &score
	}

	if resp.FailureCode != nil {
		reason := s.mapFailureCodeToRejectionReason(*resp.FailureCode)
		result.RejectionReason = &reason
	}

	if resp.FailureReason != nil {
		result.RejectionNotes = resp.FailureReason
	}

	// Store extracted data in raw response
	for k, v := range resp.ExtractedData {
		result.RawResponse[k] = v
	}
	result.RawResponse["status"] = string(resp.Status)
	result.RawResponse["message"] = resp.Message

	return result
}

func (s *verificationService) mapWebhookEventToResult(event *kyc.WebhookEvent) domain.VerificationResult {
	result := domain.VerificationResult{
		Success:       event.Status == kyc.StatusApproved,
		ProviderName:  event.Provider,
		ProviderRefID: event.ProviderRef,
		ProcessedAt:   event.ReceivedAt,
		RawResponse:   make(map[string]any),
	}

	if event.FailureCode != nil {
		reason := s.mapFailureCodeToRejectionReason(*event.FailureCode)
		result.RejectionReason = &reason
	}

	if event.FailureReason != nil {
		result.RejectionNotes = event.FailureReason
	}

	// Store extracted data
	for k, v := range event.ExtractedData {
		result.RawResponse[k] = v
	}
	result.RawResponse["status"] = string(event.Status)

	return result
}

func (s *verificationService) mapFailureCodeToRejectionReason(code kyc.FailureCode) domain.RejectionReason {
	switch code {
	case kyc.FailureDocExpired:
		return domain.RejectionDocumentExpired
	case kyc.FailureInvalidDocument:
		return domain.RejectionDocumentInvalid
	case kyc.FailureImageBlurry, kyc.FailurePoorQuality:
		return domain.RejectionDocumentUnreadable
	case kyc.FailureFaceMismatch, kyc.FailureNoFaceDetected:
		return domain.RejectionPhotoMismatch
	case kyc.FailureUnderage:
		return domain.RejectionUnderAge
	case kyc.FailureSuspectedFraud:
		return domain.RejectionDuplicateAccount
	case kyc.FailureProviderError, kyc.FailureProviderTimeout:
		return domain.RejectionProviderError
	default:
		return domain.RejectionOther
	}
}

func (s *verificationService) mapAddressDocTypeToEvidence(docType string) domain.EvidenceType {
	switch docType {
	case "utility_bill":
		return domain.EvidenceAddressUtilityBill
	case "bank_statement":
		return domain.EvidenceAddressBankStatement
	case "lease":
		return domain.EvidenceAddressLease
	default:
		return domain.EvidenceAddressOther
	}
}

func stringPtr(s string) *string {
	return &s
}

// =======================
// Tier to Level Mapping
// =======================

// tierToVerificationLevel maps verification tier to profile verification level
func (s *verificationService) tierToVerificationLevel(tier domain.VerificationTier) string {
	switch tier {
	case domain.TierBasic:
		return "basic"
	case domain.TierStandard:
		return "identity"
	case domain.TierEnhanced:
		return "trusted"
	default:
		return "basic"
	}
}

// =======================
// OTP Helper Functions
// =======================

// generateSecureOTP generates a cryptographically secure 6-digit OTP
func (s *verificationService) generateSecureOTP() (string, error) {
	max := big.NewInt(1000000) // 10^6 for 6 digits
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	// Format with leading zeros if necessary
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// buildOTPKey constructs the Redis key for OTP storage
func (s *verificationService) buildOTPKey(sessionID string) string {
	return otpKeyPrefix + sessionID
}
