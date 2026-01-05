package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/verification/domain"
)

// =======================
// Webhook Processing
// =======================

func (s *verificationService) ProcessWebhook(ctx context.Context, req ProcessWebhookRequest) error {
	// Verify webhook signature
	valid, err := s.kycClient.VerifyWebhookSignature(req.ProviderName, req.Payload, req.Signature)
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}
	if !valid {
		s.logger.Warn("invalid webhook signature",
			"provider", req.ProviderName,
		)
		return fmt.Errorf("invalid webhook signature")
	}

	// Parse webhook event
	event, err := s.kycClient.ParseWebhook(ctx, req.ProviderName, req.Payload, req.Headers)
	if err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Find attempt by provider reference
	attempt, err := s.repo.GetAttemptByProviderSessionID(ctx, event.ProviderRef)
	if err != nil {
		s.logger.Warn("attempt not found for webhook",
			"provider_ref", event.ProviderRef,
			"provider", req.ProviderName,
		)
		return domain.ErrAttemptNotFound
	}

	// Mark webhook received
	attempt.MarkWebhookReceived()

	// Map webhook event to verification result
	result := s.mapWebhookEventToResult(event)

	// Complete attempt
	if err := attempt.Complete(result); err != nil {
		return err
	}
	if err := s.repo.UpdateAttempt(ctx, attempt); err != nil {
		return fmt.Errorf("failed to update attempt: %w", err)
	}

	// Update session
	session, err := s.repo.GetSessionByID(ctx, attempt.SessionID)
	if err != nil {
		return err
	}

	if result.Success {
		if err := session.Approve(); err != nil {
			return err
		}

		// Notify relevant module based on verification type
		s.notifyVerificationSuccess(ctx, session)
	} else {
		notes := result.RejectionNotes
		if err := session.Reject(*result.RejectionReason, notes); err != nil {
			return err
		}
	}

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	s.logger.Info("webhook processed",
		"provider", req.ProviderName,
		"provider_ref", event.ProviderRef,
		"session_id", session.ID,
		"status", event.Status,
	)

	return nil
}

// notifyVerificationSuccess sends notifications to relevant modules based on verification type
func (s *verificationService) notifyVerificationSuccess(ctx context.Context, session *domain.VerificationSession) {
	now := time.Now()

	switch session.Type {
	case domain.VerificationIdentity:
		verificationLevel := s.tierToVerificationLevel(session.Tier)
		if err := s.profileAdapter.MarkIdentityVerified(ctx, session.UserID, verificationLevel, now); err != nil {
			s.logger.Error("failed to notify profile of identity verification via webhook",
				"session_id", session.ID,
				"user_id", session.UserID,
				"error", err,
			)
		}

	case domain.VerificationPhone:
		if phoneData := session.Data.Phone; phoneData != nil {
			if err := s.profileAdapter.MarkPhoneVerified(ctx, session.UserID, phoneData.PhoneNumber, now); err != nil {
				s.logger.Error("failed to notify profile of phone verification via webhook",
					"session_id", session.ID,
					"user_id", session.UserID,
					"error", err,
				)
			}
		}

	case domain.VerificationAddress:
		if addressData := session.Data.Address; addressData != nil {
			if err := s.profileAdapter.MarkAddressVerified(ctx, session.UserID, addressData.FullAddress, now); err != nil {
				s.logger.Error("failed to notify profile of address verification via webhook",
					"session_id", session.ID,
					"user_id", session.UserID,
					"error", err,
				)
			}
		}

	case domain.VerificationBusiness:
		if businessData := session.Data.Business; businessData != nil && businessData.OwnerUserID != nil {
			// For business verification, we need a businessID not userID
			// The business module adapter expects a business entity ID
			// This would typically come from the business data or be looked up
			s.logger.Info("business verification approved via webhook - manual business update may be required",
				"session_id", session.ID,
				"business_name", businessData.BusinessName,
			)
			// Note: Business verification completion requires the business entity ID
			// which should be associated with the session or looked up by registration number
		}
	}
}
