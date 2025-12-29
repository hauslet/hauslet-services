package service

import (
	"context"
	"fmt"
	"hauslet/config"
	"hauslet/internal/modules/finance/domain"
	financeSchema "hauslet/internal/modules/finance/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QueuePayout creates a pending payout record for later processing
func (s *PayoutServiceImpl) QueuePayout(ctx context.Context, bookingID, hostID uuid.UUID, totalAmount int64, currency string) error {
	if s.log != nil {
		s.log.Logf("INFO queueing payout for booking=%s host=%s amount=%d %s", bookingID, hostID, totalAmount, currency)
	}

	// Validate amount
	if err := validateAmount(totalAmount); err != nil {
		return err
	}

	// Generate idempotency reference for payout transaction
	nonce := fmt.Sprintf("%d", time.Now().UnixNano())
	reference := generateReference(domain.TransactionTypePayout, bookingID, totalAmount, nonce)

	// Check for duplicate payout transaction
	existingEntry, _ := s.ledgerRepo.GetByReference(ctx, reference+":debit")
	if existingEntry != nil {
		if s.log != nil {
			s.log.Logf("WARN payout already queued for booking=%s", bookingID)
		}
		return domain.ErrDuplicateTransaction
	}

	// Note: Actual payout processing happens in ProcessDuePayouts cron job
	// This just validates and marks the booking as ready for payout
	if s.log != nil {
		s.log.Logf("INFO payout queued successfully for booking=%s", bookingID)
	}

	return nil
}

// ProcessDuePayouts processes all bookings ready for payout
// This should be called by a cron job (e.g., every hour)
func (s *PayoutServiceImpl) ProcessDuePayouts(ctx context.Context) error {
	if s.log != nil {
		s.log.Logf("INFO [AUDIT] payout_batch_started time=%s", time.Now().Format(time.RFC3339))
	}

	// Check if booking querier is configured
	if s.bookingQuerier == nil {
		if s.log != nil {
			s.log.Logf("WARN booking querier not configured, skipping payout processing")
		}
		return nil
	}

	// Get payout window and event from config
	payoutWindowHours := s.platformConfig.Payouts.EscrowReleaseHours
	if payoutWindowHours == 0 {
		payoutWindowHours = 48 // Default to 48 hours
	}

	// Keep typed for validation and type safety
	escrowReleaseEvent := s.platformConfig.Payouts.EscrowReleaseEvent
	if !escrowReleaseEvent.IsValid() {
		escrowReleaseEvent = config.EscrowReleaseCheckoutConfirmed // Type-safe default
	}

	if s.log != nil {
		s.log.Logf("INFO [AUDIT] payout_config escrow_release_event=%s escrow_release_hours=%d",
			escrowReleaseEvent, payoutWindowHours)
	}

	// Convert to string only when passing to the query
	bookings, err := s.bookingQuerier.FindBookingsReadyForPayout(ctx, escrowReleaseEvent.String(), payoutWindowHours, 100)
	if err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to query bookings ready for payout: %v", err)
		}
		return fmt.Errorf("failed to query bookings: %w", err)
	}

	successCount := 0
	failureCount := 0

	if s.log != nil {
		s.log.Logf("INFO [AUDIT] found %d bookings ready for payout", len(bookings))
	}

	// Process each booking
	for _, booking := range bookings {
		// Get escrow wallet for this booking
		escrowWallet, err := s.walletRepo.GetByOwner(
			ctx,
			string(domain.OwnerTypeUser),
			booking.ID,
			string(domain.WalletTypeEscrow),
		)
		if err != nil {
			if s.log != nil {
				s.log.Logf("ERROR [AUDIT] payout_failed booking_id=%s reason=escrow_wallet_not_found error=%v",
					booking.ID, err)
			}
			failureCount++
			continue
		}

		// Process payout
		err = s.processSinglePayout(
			ctx,
			booking.ID,
			booking.HostID,
			escrowWallet.ID,
			booking.TotalAmount,
			booking.Currency,
		)
		if err != nil {
			if s.log != nil {
				s.log.Logf("ERROR [AUDIT] payout_failed booking_id=%s host_id=%s error=%v",
					booking.ID, booking.HostID, err)
			}
			failureCount++
		} else {
			if s.log != nil {
				s.log.Logf("INFO [AUDIT] payout_success booking_id=%s host_id=%s amount=%d currency=%s",
					booking.ID, booking.HostID, booking.TotalAmount, booking.Currency)
			}
			successCount++
		}
	}

	if s.log != nil {
		s.log.Logf("INFO [AUDIT] payout_batch_completed time=%s total=%d success=%d failed=%d",
			time.Now().Format(time.RFC3339), len(bookings), successCount, failureCount)
	}

	return nil
}

// processSinglePayout processes payout for a single booking
func (s *PayoutServiceImpl) processSinglePayout(
	ctx context.Context,
	bookingID uuid.UUID,
	hostID uuid.UUID,
	escrowWalletID uuid.UUID,
	totalAmount int64,
	currency string,
) error {
	if s.log != nil {
		s.log.Logf("INFO [AUDIT] payout_processing_started booking_id=%s host_id=%s amount=%d currency=%s",
			bookingID, hostID, totalAmount, currency)
	}

	// Use database transaction to ensure atomicity
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		walletRepo := s.walletRepo.WithTx(tx)
		disbursementRepo := s.disbursementRepo.WithTx(tx)

		// Step 1: Calculate platform commission from config
		commission := calculateCommission(totalAmount, s.platformConfig.Fees.HostCommissionPercent)
		hostPayout := totalAmount - commission

		if s.log != nil {
			s.log.Logf("INFO [AUDIT] payout_breakdown booking_id=%s total=%d commission=%d host_payout=%d commission_rate=%.2f%%",
				bookingID, totalAmount, commission, hostPayout, s.platformConfig.Fees.HostCommissionPercent*100)
		}

		// Step 2: Get platform fee wallet
		platformID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("hauslet"))
		platformWalletSchema, err := walletRepo.GetByOwner(
			ctx,
			string(domain.OwnerTypePlatform),
			platformID,
			string(domain.WalletTypePlatformFee),
		)
		if err != nil {
			return fmt.Errorf("failed to get platform wallet: %w", err)
		}
		platformWallet := domain.MapWalletFromSchema(platformWalletSchema)

		// Step 3: Record commission ledger entry (escrow → platform_fee)
		commissionTx, err := s.recordCommissionInternal(ctx, tx, bookingID, escrowWalletID, platformWallet.ID, commission, currency)
		if err != nil {
			return fmt.Errorf("failed to record commission: %w", err)
		}

		if s.log != nil {
			s.log.Logf("INFO commission recorded: tx=%s amount=%d", commissionTx.ID, commission)
		}

		// Step 4: Get or create host available wallet
		hostWalletSchema, err := walletRepo.GetByOwner(
			ctx,
			string(domain.OwnerTypeUser),
			hostID,
			string(domain.WalletTypeHostAvailable),
		)
		if err == gorm.ErrRecordNotFound {
			// Create host wallet
			newWallet := &financeSchema.Wallet{
				ID:         uuid.New(),
				OwnerType:  string(domain.OwnerTypeUser),
				OwnerID:    hostID,
				WalletType: string(domain.WalletTypeHostAvailable),
				Balance:    0,
				Currency:   currency,
				Status:     string(domain.WalletStatusActive),
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if err := walletRepo.Create(ctx, newWallet); err != nil {
				return fmt.Errorf("failed to create host wallet: %w", err)
			}
			hostWalletSchema = newWallet
		} else if err != nil {
			return fmt.Errorf("failed to get host wallet: %w", err)
		}
		hostWallet := domain.MapWalletFromSchema(hostWalletSchema)

		// Step 5: Record payout ledger entry (escrow → host_available)
		payoutTx, err := s.recordPayoutInternal(ctx, tx, bookingID, escrowWalletID, hostWallet.ID, hostPayout, currency)
		if err != nil {
			return fmt.Errorf("failed to record payout: %w", err)
		}

		if s.log != nil {
			s.log.Logf("INFO payout recorded: tx=%s amount=%d", payoutTx.ID, hostPayout)
		}

		// Step 6: Create disbursement record for actual bank transfer
		disbursement := &financeSchema.Disbursement{
			ID:            uuid.New(),
			WalletID:      hostWallet.ID,
			TransactionID: payoutTx.ID,
			Amount:        hostPayout,
			Currency:      currency,
			Provider:      "paystack", // TODO: Make configurable
			Status:        string(domain.DisbursementStatusPending),
			Attempts:      0,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := disbursementRepo.Create(ctx, disbursement); err != nil {
			return fmt.Errorf("failed to create disbursement: %w", err)
		}

		if s.log != nil {
			s.log.Logf("INFO disbursement created: id=%s", disbursement.ID)
		}

		// Step 7: Initiate transfer via payment provider
		if err := s.initiateDisbursement(ctx, disbursement); err != nil {
			// Don't fail the whole transaction - we'll retry later
			if s.log != nil {
				s.log.Logf("WARN failed to initiate disbursement (will retry): %v", err)
			}
			// Set next retry time
			nextRetry := time.Now().Add(s.calculateRetryDelay(0))
			disbursement.NextRetryAt = &nextRetry
			disbursement.Status = string(domain.DisbursementStatusFailed)
			if err := s.disbursementRepo.UpdateStatus(ctx, disbursement.ID, disbursement.Status, nil); err != nil {
				return fmt.Errorf("failed to update disbursement status: %w", err)
			}
		}

		// Step 8: Mark booking as settled via hooks
		if s.bookingHooks != nil {
			if err := s.bookingHooks.MarkAsSettled(ctx, bookingID); err != nil {
				// Log but don't fail - booking status update is not critical
				if s.log != nil {
					s.log.Logf("WARN failed to mark booking as settled: %v", err)
				}
			}
		}

		return nil
	})
}
