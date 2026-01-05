package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/verification/domain"
	"hauslet/internal/platform/kyc"
)

// =======================
// Identity Verification (KYC)
// =======================

func (s *verificationService) SubmitIdentityVerification(ctx context.Context, req SubmitIdentityVerificationRequest) (*SubmitVerificationResponse, error) {
	// Get session and authorize
	session, err := s.GetSession(ctx, req.SessionID, req.UserID)
	if err != nil {
		return nil, err
	}

	// Check if session can accept new attempt
	if err := session.CanSubmitAttempt(); err != nil {
		return nil, err
	}

	// Apply multi-dimensional rate limiting
	if err := s.checkRateLimits(ctx, session, req.IPAddress); err != nil {
		return nil, err
	}

	// Estimate which provider will be used (for circuit breaker check)
	// Uses EstimateCost which internally calls factory.GetProvider()
	_, providerName, err := s.kycClient.EstimateCost(session.Country)
	if err != nil {
		return nil, fmt.Errorf("no KYC provider available for country %s: %w", session.Country, err)
	}

	// Check circuit breaker for selected provider
	allowed, err := s.circuitBreaker.AllowRequest(ctx, providerName)
	if err != nil {
		s.logger.Error("circuit breaker check failed", "error", err)
	}
	if !allowed {
		return nil, fmt.Errorf("KYC provider %s is currently unavailable", providerName)
	}

	// Create attempt record
	attempt, err := domain.NewVerificationAttempt(session.ID, providerName)
	if err != nil {
		return nil, err
	}

	// Update session state
	if err := session.StartAttempt(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Submit to KYC provider
	kycReq := s.buildKYCRequest(session, req)
	kycResp, err := s.kycClient.SubmitVerification(ctx, kycReq)

	if err != nil {
		// Record failure in circuit breaker
		_ = s.circuitBreaker.RecordFailure(ctx, providerName)

		// Mark attempt as failed
		attempt.Status = domain.AttemptFailed
		_ = s.repo.CreateAttempt(ctx, attempt)

		s.logger.Error("KYC submission failed",
			"session_id", session.ID,
			"provider", providerName,
			"error", err,
		)

		return nil, fmt.Errorf("verification submission failed: %w", err)
	}

	// Record success in circuit breaker
	_ = s.circuitBreaker.RecordSuccess(ctx, providerName)

	// Update attempt with provider response
	if err := attempt.Submit(kycResp.ProviderRef); err != nil {
		return nil, err
	}

	// Map KYC response to domain result
	result := s.mapKYCResponseToResult(kycResp)

	// If verification completed immediately, process result
	if kycResp.Status == kyc.StatusApproved || kycResp.Status == kyc.StatusRejected {
		if err := attempt.Complete(result); err != nil {
			return nil, err
		}

		// Update session based on result
		if result.Success {
			if err := session.Approve(); err != nil {
				return nil, err
			}

			// Notify profile module of identity verification
			verificationLevel := s.tierToVerificationLevel(session.Tier)
			if err := s.profileAdapter.MarkIdentityVerified(ctx, session.UserID, verificationLevel, time.Now()); err != nil {
				s.logger.Error("failed to notify profile of identity verification",
					"session_id", session.ID,
					"user_id", session.UserID,
					"error", err,
				)
			}
		} else {
			notes := result.RejectionNotes
			if err := session.Reject(*result.RejectionReason, notes); err != nil {
				return nil, err
			}
		}

		if err := s.repo.UpdateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
	}

	// Save attempt
	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		return nil, fmt.Errorf("failed to save attempt: %w", err)
	}

	s.logger.Info("verification submitted",
		"session_id", session.ID,
		"attempt_id", attempt.ID,
		"provider", providerName,
		"provider_ref", kycResp.ProviderRef,
		"status", kycResp.Status,
	)

	return &SubmitVerificationResponse{
		Attempt:      attempt,
		Session:      session,
		ProviderName: providerName,
		Status:       string(kycResp.Status),
		Message:      kycResp.Message,
	}, nil
}
