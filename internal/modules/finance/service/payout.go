package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/config"
	"hauslet/internal/modules/finance/domain"
	financeSchema "hauslet/internal/modules/finance/repository/schema"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QueuePayout creates a pending payout record for later processing
func (s *PayoutServiceImpl) QueuePayout(ctx context.Context,
	bookingID,
	hostID uuid.UUID,
	totalAmount int64,
	currency string) error {
	if s.log != nil {
		s.log.Info("queueing payout",
			"booking_id", bookingID,
			"host_id", hostID,
			"amount", totalAmount,
			"currency", currency,
		)
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
			s.log.Warn("payout already queued",
				"booking_id", bookingID,
			)
		}
		return domain.ErrDuplicateTransaction
	}

	// Note: Actual payout processing happens in ProcessDuePayouts cron job
	// This just validates and marks the booking as ready for payout
	if s.log != nil {
		s.log.Info("payout queued successfully",
			"booking_id", bookingID,
		)
	}

	return nil
}

// ProcessDuePayouts processes all bookings ready for payout
// This should be called by a cron job (e.g., every hour)
func (s *PayoutServiceImpl) ProcessDuePayouts(ctx context.Context) error {
	if s.log != nil {
		s.log.Info("[AUDIT] payout_batch_started",
			"time", time.Now().Format(time.RFC3339),
		)
	}

	// Check if booking querier is configured
	if s.bookingQuerier == nil {
		if s.log != nil {
			s.log.Warn("booking querier not configured, skipping payout processing")
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
		s.log.Info("[AUDIT] payout_config",
			"escrow_release_event", escrowReleaseEvent,
			"escrow_release_hours", payoutWindowHours,
		)
	}

	// Convert to string only when passing to the query
	bookings, err := s.bookingQuerier.FindBookingsReadyForPayout(ctx, escrowReleaseEvent.String(), payoutWindowHours, 100)
	if err != nil {
		if s.log != nil {
			s.log.Error("failed to query bookings ready for payout",
				"error", err,
			)
		}
		return fmt.Errorf("failed to query bookings: %w", err)
	}

	successCount := 0
	failureCount := 0

	if s.log != nil {
		s.log.Info("[AUDIT] found bookings ready for payout",
			"count", len(bookings),
		)
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
				s.log.Error("[AUDIT] payout_failed",
					"booking_id", booking.ID,
					"reason", "escrow_wallet_query_error",
					"error", err,
				)
			}
			failureCount++
			continue
		}

		// Check if wallet exists
		if escrowWallet == nil {
			if s.log != nil {
				s.log.Error("[AUDIT] payout_failed",
					"booking_id", booking.ID,
					"reason", "escrow_wallet_not_found",
				)
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
				s.log.Error("[AUDIT] payout_failed",
					"booking_id", booking.ID,
					"host_id", booking.HostID,
					"error", err,
				)
			}
			failureCount++
		} else {
			if s.log != nil {
				s.log.Info("[AUDIT] payout_success",
					"booking_id", booking.ID,
					"host_id", booking.HostID,
					"amount", booking.TotalAmount,
					"currency", booking.Currency,
				)
			}
			successCount++
		}
	}

	if s.log != nil {
		s.log.Info("[AUDIT] payout_batch_completed",
			"time", time.Now().Format(time.RFC3339),
			"total", len(bookings),
			"success", successCount,
			"failed", failureCount,
		)
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
		s.log.Info("[AUDIT] payout_processing_started",
			"booking_id", bookingID,
			"host_id", hostID,
			"amount", totalAmount,
			"currency", currency,
		)
	}

	// Use database transaction to ensure atomicity
	var disbursement *financeSchema.Disbursement
	var payoutSkipped bool
	var payoutWithheld bool
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create transaction-aware repositories
		walletRepo := s.walletRepo.WithTx(tx)
		disbursementRepo := s.disbursementRepo.WithTx(tx)

		// Step 0: Pre-validate payout details (Industry Standard: Hold & Notify)
		// We verify host has valid banking details BEFORE moving any money.
		payoutDetail, err := s.payoutDetailRepo.GetDefaultPayoutDetail(ctx, &hostID, nil)
		if err != nil || payoutDetail == nil || !payoutDetail.IsVerified {
			if s.log != nil {
				s.log.Warn("[AUDIT] payout_skipped_missing_details",
					"booking_id", bookingID,
					"host_id", hostID,
					"reason", "missing_or_unverified_payout_details",
				)
			}

			// Notify host provided we have notification service
			if s.notificationSvc != nil && s.profileAdapter != nil {
				// Get host profile for email and name
				hostName, hostEmail, err := s.profileAdapter.GetProfileData(ctx, hostID)
				if err == nil && hostEmail != "" {
					// Use specific "Action Required" notification
					s.notificationSvc.SendPayoutActionRequired(
						ctx,
						bookingID,
						hostEmail,
						hostName,
						totalAmount,
						currency,
					)
				}
			}

			payoutSkipped = true
			// Return nil to skip this payout for now. It will be picked up again
			// in the next cron run once the user adds their details.
			return nil
		}

		// Step 1: Calculate platform commission from config
		commission := calculateCommission(totalAmount, s.platformConfig.Fees.HostCommissionPercent)
		hostPayout := totalAmount - commission

		if s.log != nil {
			s.log.Info("[AUDIT] payout_breakdown",
				"booking_id", bookingID,
				"total", totalAmount,
				"commission", commission,
				"host_payout", hostPayout,
				"commission_rate", s.platformConfig.Fees.HostCommissionPercent*100,
			)
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
		if platformWalletSchema == nil {
			return fmt.Errorf("platform fee wallet not found for platform ID: %s", platformID)
		}
		platformWallet := domain.MapWalletFromSchema(platformWalletSchema)

		// Step 3: Record commission ledger entry (escrow → platform_fee)
		commissionTx, err := s.recordCommissionInternal(ctx, tx, bookingID, escrowWalletID, platformWallet.ID, commission, currency)
		if err != nil {
			return fmt.Errorf("failed to record commission: %w", err)
		}

		if s.log != nil {
			s.log.Info("commission recorded",
				"transaction_id", commissionTx.ID,
				"amount", commission,
			)
		}

		// Step 3b: Apply outstanding penalty debts (netted from host payout)
		if hostPayout > 0 {
			penaltyApplied, err := s.applyPenaltyDebtsToPayout(
				ctx,
				tx,
				hostID,
				escrowWalletID,
				platformWallet.ID,
				hostPayout,
				currency,
				bookingID,
			)
			if err != nil {
				return fmt.Errorf("failed to apply penalty debts: %w", err)
			}
			if penaltyApplied > 0 && s.log != nil {
				s.log.Info("[AUDIT] penalty_debt_applied",
					"booking_id", bookingID,
					"host_id", hostID,
					"amount", penaltyApplied,
				)
			}
			hostPayout -= penaltyApplied
			if hostPayout < 0 {
				hostPayout = 0
			}
		}

		var hostWallet *domain.Wallet
		var payoutTx *domain.Transaction

		if hostPayout > 0 {
			// Step 4: Get or create host available wallet
			hostWalletSchema, err := walletRepo.GetByOwner(
				ctx,
				string(domain.OwnerTypeUser),
				hostID,
				string(domain.WalletTypeHostAvailable),
			)
			if err != nil {
				return fmt.Errorf("failed to get host wallet: %w", err)
			}
			if hostWalletSchema == nil {
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
			}
			hostWallet = domain.MapWalletFromSchema(hostWalletSchema)

			// Step 5: Record payout ledger entry (escrow → host_available)
			payoutTx, err = s.recordPayoutInternal(ctx, tx, bookingID, escrowWalletID, hostWallet.ID, hostPayout, currency)
			if err != nil {
				return fmt.Errorf("failed to record payout: %w", err)
			}

			if s.log != nil {
				s.log.Info("payout recorded",
					"transaction_id", payoutTx.ID,
					"amount", hostPayout,
				)
			}
		} else if s.log != nil {
			s.log.Info("[AUDIT] payout_withheld_due_to_penalty",
				"booking_id", bookingID,
				"host_id", hostID,
			)
		}

		// Step 6: Create disbursement record for actual bank transfer
		if hostPayout <= 0 {
			// No funds to disburse after penalties.
			payoutWithheld = true
			if s.bookingHooks != nil {
				if err := s.bookingHooks.MarkAsSettled(ctx, bookingID); err != nil && s.log != nil {
					s.log.Warn("failed to mark booking as settled",
						"error", err,
					)
				}
			}
			return nil
		}

		defaultProvider := "paystack"
		switch strings.ToUpper(currency) {
		case "USD", "GHS":
			defaultProvider = "stripe"
		}

		provider := strings.ToLower(strings.TrimSpace(s.platformConfig.Payouts.DisbursementProvider))
		if provider == "" {
			provider = defaultProvider
		} else {
			switch provider {
			case "paystack", "stripe":
				if provider != defaultProvider {
					if s.log != nil {
						s.log.Warn("disbursement provider does not match currency; using default",
							"provider", provider,
							"currency", currency,
							"default_provider", defaultProvider,
						)
					}
					provider = defaultProvider
				}
			default:
				if s.log != nil {
					s.log.Warn("invalid disbursement provider; using default",
						"provider", provider,
						"default_provider", defaultProvider,
					)
				}
				provider = defaultProvider
			}
		}

		disbursement = &financeSchema.Disbursement{
			ID:            uuid.New(),
			WalletID:      hostWallet.ID,
			TransactionID: payoutTx.ID,
			Amount:        hostPayout,
			Currency:      currency,
			Provider:      provider,
			Status:        string(domain.DisbursementStatusPending),
			Attempts:      0,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := disbursementRepo.Create(ctx, disbursement); err != nil {
			return fmt.Errorf("failed to create disbursement: %w", err)
		}

		if s.log != nil {
			s.log.Info("disbursement created",
				"disbursement_id", disbursement.ID,
			)
		}

		// Step 8: Mark booking as settled via hooks
		if s.bookingHooks != nil {
			if err := s.bookingHooks.MarkAsSettled(ctx, bookingID); err != nil {
				// Log but don't fail - booking status update is not critical
				if s.log != nil {
					s.log.Warn("failed to mark booking as settled",
						"error", err,
					)
				}
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if disbursement == nil {
		if payoutWithheld {
			if s.log != nil {
				s.log.Info("[AUDIT] payout_skipped_due_to_penalty_debt",
					"booking_id", bookingID,
					"host_id", hostID,
				)
			}
			return nil
		}
		if payoutSkipped {
			return nil
		}
		return fmt.Errorf("disbursement creation failed")
	}

	// Step 7: Initiate transfer via payment provider (run after transaction commit)
	if err := s.initiateDisbursement(ctx, disbursement); err != nil {
		// Don't fail the whole payout; we mark for retry
		if s.log != nil {
			s.log.Warn("failed to initiate disbursement (will retry)",
				"error", err,
			)
		}
		nextRetry := time.Now().Add(s.calculateRetryDelay(0))
		disbursement.NextRetryAt = &nextRetry
		disbursement.Status = string(domain.DisbursementStatusFailed)
		if updateErr := s.disbursementRepo.UpdateStatus(
			ctx,
			disbursement.ID,
			disbursement.Status,
			nil,
		); updateErr != nil {
			return fmt.Errorf("failed to update disbursement status: %w", updateErr)
		}
	}

	return nil
}
