package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	financeSchema "hauslet/internal/modules/finance/repository/schema"
	"hauslet/internal/platform/payment"
	"time"

	"github.com/google/uuid"
)

// initiateDisbursement calls payment provider to initiate bank transfer
func (s *PayoutServiceImpl) initiateDisbursement(ctx context.Context, disbursement *financeSchema.Disbursement) error {
	if s.log != nil {
		s.log.Info("initiating disbursement", "id", disbursement.ID, "amount", disbursement.Amount)
	}

	// Get wallet to find host ID
	walletSchema, err := s.walletRepo.GetByID(ctx, disbursement.WalletID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	// Get host's payout details (recipient code)
	payoutDetail, err := s.payoutDetailRepo.GetDefaultPayoutDetail(ctx, &walletSchema.OwnerID, nil)
	if err != nil {
		errMsg := fmt.Sprintf("host has no payout details configured: %v", err)
		if s.log != nil {
			s.log.Error("host payout details error", "message", errMsg)
		}
		// Update disbursement status to failed
		disbursement.FailureReason = &errMsg
		disbursement.Status = string(domain.DisbursementStatusFailed)
		nextRetry := time.Now().Add(s.calculateRetryDelay(disbursement.Attempts))
		disbursement.NextRetryAt = &nextRetry
		if updateErr := s.disbursementRepo.UpdateStatus(ctx, disbursement.ID, disbursement.Status, &errMsg); updateErr != nil {
			return fmt.Errorf("failed to update disbursement status: %w (original error: %v)", updateErr, err)
		}
		return fmt.Errorf("host has no payout details configured: %w", err)
	}

	// Ensure payout details are verified
	if !payoutDetail.IsVerified {
		errMsg := "host payout details not verified"
		if s.log != nil {
			s.log.Error(errMsg, "host_id", walletSchema.OwnerID)
		}
		disbursement.FailureReason = &errMsg
		disbursement.Status = string(domain.DisbursementStatusFailed)
		nextRetry := time.Now().Add(s.calculateRetryDelay(disbursement.Attempts))
		disbursement.NextRetryAt = &nextRetry
		if updateErr := s.disbursementRepo.UpdateStatus(ctx, disbursement.ID, disbursement.Status, &errMsg); updateErr != nil {
			return fmt.Errorf("failed to update disbursement status: %w", updateErr)
		}
		return fmt.Errorf("host payout details not verified")
	}

	// Initiate the actual transfer with recipient code
	return s.initiateTransferWithRecipient(ctx, disbursement, payoutDetail.RecipientCode)
}

// InitiateTransferWithRecipient initiates a transfer with a known recipient code
// This is the method that will be called once we have PayoutDetail integration
func (s *PayoutServiceImpl) initiateTransferWithRecipient(
	ctx context.Context,
	disbursement *financeSchema.Disbursement,
	recipientCode string,
) error {
	if s.paymentClient == nil {
		return fmt.Errorf("payment client not configured")
	}

	// Build payout request
	payoutReq := payment.PayoutRequest{
		Amount:        disbursement.Amount,
		Currency:      payment.Currency(disbursement.Currency),
		RecipientCode: recipientCode,
		Reference:     disbursement.ID.String(),
		Narration:     fmt.Sprintf("Booking payout - Transaction %s", disbursement.TransactionID),
		Metadata: map[string]string{
			"disbursement_id": disbursement.ID.String(),
			"transaction_id":  disbursement.TransactionID.String(),
			"wallet_id":       disbursement.WalletID.String(),
		},
	}

	// Call payment provider
	resp, err := s.paymentClient.Transfer(ctx, payoutReq)
	if err != nil {
		// Update disbursement with error
		errMsg := err.Error()
		disbursement.FailureReason = &errMsg
		disbursement.Status = string(domain.DisbursementStatusFailed)

		// Calculate next retry time
		nextRetry := time.Now().Add(s.calculateRetryDelay(disbursement.Attempts))
		disbursement.NextRetryAt = &nextRetry

		if updateErr := s.disbursementRepo.UpdateStatus(ctx, disbursement.ID, disbursement.Status, &errMsg); updateErr != nil {
			return fmt.Errorf("transfer failed and status update failed: %w (original error: %v)", updateErr, err)
		}

		return fmt.Errorf("transfer failed: %w", err)
	}

	// Store provider transfer code
	transferCode := resp.Reference
	if resp.TransferID != "" {
		transferCode = resp.TransferID
	}
	disbursement.TransferCode = &transferCode

	// Update status based on provider response
	statusMsg := resp.Message
	switch resp.Status {
	case payment.TransferSuccess:
		disbursement.Status = string(domain.DisbursementStatusCompleted)
		now := time.Now()
		disbursement.CompletedAt = &now
	case payment.TransferFailed:
		disbursement.Status = string(domain.DisbursementStatusFailed)
		errMsg := resp.Message
		disbursement.FailureReason = &errMsg
		nextRetry := time.Now().Add(s.calculateRetryDelay(disbursement.Attempts))
		disbursement.NextRetryAt = &nextRetry
	default:
		disbursement.Status = string(domain.DisbursementStatusProcessing)
	}

	// Store provider response
	providerResp := fmt.Sprintf("status=%s, transfer_id=%s, message=%s", resp.Status, resp.TransferID, resp.Message)
	disbursement.ProviderResponse = &providerResp

	if err := s.disbursementRepo.UpdateStatus(ctx, disbursement.ID, disbursement.Status, &statusMsg); err != nil {
		return fmt.Errorf("failed to update disbursement status after transfer: %w", err)
	}

	if s.log != nil {
		s.log.Info("transfer initiated", "disbursement_id", disbursement.ID, "transfer_code", transferCode, "status", resp.Status)
	}

	return nil
}

// RetryFailedDisbursements retries all failed disbursements that are due for retry
func (s *PayoutServiceImpl) RetryFailedDisbursements(ctx context.Context) error {
	if s.log != nil {
		s.log.Info(" retrying failed disbursements")
	}

	// Get all pending retries
	pendingRetries, err := s.disbursementRepo.ListPendingRetries(ctx)
	if err != nil {
		return fmt.Errorf("failed to list pending retries: %w", err)
	}

	if s.log != nil {
		s.log.Info(" found disbursements to retry", "count", len(pendingRetries))
	}

	for _, d := range pendingRetries {
		if err := s.retryDisbursement(ctx, d); err != nil {
			if s.log != nil {
				s.log.Error("failed to retry disbursement", "id", d.ID, "error", err)
			}
			// Continue with other retries
			continue
		}
	}

	return nil
}

// retryDisbursement retries a single failed disbursement
func (s *PayoutServiceImpl) retryDisbursement(ctx context.Context, disbursement *financeSchema.Disbursement) error {
	if s.log != nil {
		s.log.Info("retrying disbursement", "id", disbursement.ID, "attempt", disbursement.Attempts+1)
	}

	// Increment attempts
	nextAttempt := disbursement.Attempts + 1
	nextRetry := time.Now().Add(s.calculateRetryDelay(nextAttempt))

	if err := s.disbursementRepo.IncrementAttempts(ctx, disbursement.ID, &nextRetry); err != nil {
		return fmt.Errorf("failed to increment attempts: %w", err)
	}

	// Try to initiate transfer again
	if err := s.initiateDisbursement(ctx, disbursement); err != nil {
		if s.log != nil {
			s.log.Warn("retry failed for disbursement", "id", disbursement.ID, "attempt", nextAttempt, "error", err)
		}
		return err
	}

	if s.log != nil {
		s.log.Info("retry successful for disbursement", "id", disbursement.ID)
	}

	return nil
}

// GetDisbursement retrieves a disbursement by ID
func (s *PayoutServiceImpl) GetDisbursement(ctx context.Context, disbursementID uuid.UUID) (*domain.Disbursement, error) {
	schema, err := s.disbursementRepo.GetByID(ctx, disbursementID)
	if err != nil {
		return nil, err
	}
	return domain.MapDisbursementFromSchema(schema), nil
}

// GetDisbursementByTransferCode retrieves a disbursement by transfer code
func (s *PayoutServiceImpl) GetDisbursementByTransferCode(ctx context.Context, code string) (*domain.Disbursement, error) {
	schema, err := s.disbursementRepo.GetByTransferCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return domain.MapDisbursementFromSchema(schema), nil
}

// UpdateDisbursementStatus updates the status of a disbursement
func (s *PayoutServiceImpl) UpdateDisbursementStatus(ctx context.Context, disbursementID uuid.UUID, status domain.DisbursementStatus, response *string) error {
	// Get disbursement details for logging and notifications
	disbursement, err := s.disbursementRepo.GetByID(ctx, disbursementID)
	if err != nil {
		return fmt.Errorf("failed to get disbursement: %w", err)
	}

	// Log status transition
	if s.log != nil {
		responseMsg := ""
		if response != nil {
			responseMsg = *response
		}
		s.log.Info("[AUDIT] disbursement_status_update",
			"disbursement_id", disbursementID,
			"old_status", disbursement.Status,
			"new_status", status,
			"amount", disbursement.Amount,
			"currency", disbursement.Currency,
			"attempts", disbursement.Attempts,
			"provider", disbursement.Provider,
			"response", responseMsg)
	}

	// Update status
	if err := s.disbursementRepo.UpdateStatus(ctx, disbursementID, string(status), response); err != nil {
		if s.log != nil {
			s.log.Error("[AUDIT] disbursement_status_update_failed",
				"disbursement_id", disbursementID,
				"status", status,
				"error", err)
		}
		return fmt.Errorf("failed to update disbursement status: %w", err)
	}

	// Send notifications based on status
	if s.notificationSvc != nil && s.profileAdapter != nil {
		// Get wallet to determine host ID
		wallet, err := s.walletRepo.GetByID(ctx, disbursement.WalletID)
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get wallet for notification", "error", err)
			}
			return nil // Don't fail the status update if notification fails
		}

		// Get host profile data (email and name)
		hostName, hostEmail, err := s.profileAdapter.GetProfileData(ctx, wallet.OwnerID)
		if err != nil {
			if s.log != nil {
				s.log.Warn("failed to get host profile for notification", "error", err)
			}
			return nil // Don't fail the status update if notification fails
		}

		// Skip notification if no email available
		if hostEmail == "" {
			if s.log != nil {
				s.log.Warn("host has no email configured, skipping notification for disbursement", "id", disbursementID)
			}
			return nil
		}

		switch status {
		case domain.DisbursementStatusCompleted:
			// Get payout details for account information
			payoutDetail, _ := s.payoutDetailRepo.GetDefaultPayoutDetail(ctx, &wallet.OwnerID, nil)
			if payoutDetail != nil {
				// Mask account number for security
				maskedAccount := maskAccountNumber(payoutDetail.AccountNumber)
				s.notificationSvc.SendPayoutSuccess(
					ctx,
					disbursementID,
					hostEmail,
					hostName,
					disbursement.Amount,
					disbursement.Currency,
					payoutDetail.AccountName,
					payoutDetail.BankName,
					maskedAccount,
				)
			}

		case domain.DisbursementStatusFailed:
			failureReason := "Transfer failed"
			if disbursement.FailureReason != nil {
				failureReason = *disbursement.FailureReason
			}

			nextRetry := time.Now().Add(s.calculateRetryDelay(disbursement.Attempts))
			if disbursement.NextRetryAt != nil {
				nextRetry = *disbursement.NextRetryAt
			}

			s.notificationSvc.SendPayoutFailed(
				ctx,
				disbursementID,
				hostEmail,
				hostName,
				disbursement.Amount,
				disbursement.Currency,
				failureReason,
				nextRetry,
			)
		}
	}

	return nil
}

// calculateRetryDelay returns retry delay based on platform config
// Uses constant backoff interval from config (default 15 minutes)
func (s *PayoutServiceImpl) calculateRetryDelay(attempts int) time.Duration {
	// Get retry backoff from config (in minutes)
	backoffMinutes := s.platformConfig.Payouts.RetryBackoffMinutes
	if backoffMinutes <= 0 {
		backoffMinutes = 15 // Default fallback
	}

	// Check max retry attempts from config
	maxAttempts := s.platformConfig.Payouts.MaxRetryAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5 // Default fallback
	}

	// If exceeded max attempts, use a longer delay (24 hours)
	if attempts >= maxAttempts {
		return 24 * time.Hour
	}

	// Use configured constant backoff
	return time.Duration(backoffMinutes) * time.Minute
}
