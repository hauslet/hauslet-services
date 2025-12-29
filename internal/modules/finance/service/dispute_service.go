package service

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/modules/finance/repository/schema"
	"time"

	"github.com/google/uuid"
)

// FileDispute creates a new dispute and freezes the associated wallet
func (s *FinanceServiceImpl) FileDispute(
	ctx context.Context,
	bookingID uuid.UUID,
	userID uuid.UUID,
	reason domain.DisputeReason,
	description string,
	amount int64,
	currency string,
) (*domain.Dispute, error) {
	// Determine if user is guest or host for this booking
	filedBy, err := s.bookingPartyQuerier.GetBookingParty(ctx, bookingID, userID)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: user not involved in this booking: %w", err)
	}

	// Get the payment ID for this booking
	paymentID, err := s.bookingPartyQuerier.GetBookingPaymentID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment ID for booking: %w", err)
	}

	// Validate amount
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	// Check if dispute already exists for this booking
	existing, err := s.disputeRepo.GetByBookingID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing dispute: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrDisputeAlreadyExists
	}

	// Get booking escrow wallet
	// Note: booking escrow wallets use OwnerTypeUser with bookingID as ownerID
	wallet, err := s.GetOrCreateWallet(ctx, domain.OwnerTypeUser, bookingID, domain.WalletTypeEscrow, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking wallet: %w", err)
	}

	// Start transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create dispute
	dispute := &domain.Dispute{
		ID:          uuid.New(),
		BookingID:   bookingID,
		WalletID:    wallet.ID,
		PaymentID:   paymentID,
		FiledBy:     filedBy,
		FiledByID:   userID,
		Reason:      reason,
		Status:      domain.DisputeStatusOpen,
		Description: description,
		Amount:      amount,
		Currency:    currency,
		Evidence:    []domain.DisputeEvidence{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save dispute
	disputeSchema := domain.MapDisputeToSchema(dispute)
	if err := s.disputeRepo.WithTx(tx).Create(ctx, disputeSchema); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create dispute: %w", err)
	}

	// Freeze wallet
	if err := s.walletRepo.WithTx(tx).UpdateStatus(ctx, wallet.ID, string(domain.WalletStatusFrozen)); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to freeze wallet: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Logf("INFO dispute filed: dispute_id=%s booking_id=%s filed_by=%s reason=%s amount=%d",
		dispute.ID, bookingID, filedBy, reason, amount)

	return dispute, nil
}

// InvestigateDispute marks a dispute as under investigation
func (s *FinanceServiceImpl) InvestigateDispute(ctx context.Context, disputeID, adminID uuid.UUID) error {
	// Get dispute
	disputeSchema, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}
	if disputeSchema == nil {
		return domain.ErrDisputeNotFound
	}

	dispute := domain.MapDisputeFromSchema(disputeSchema)

	// Mark as investigating
	if err := dispute.MarkInvestigating(); err != nil {
		return err
	}

	// Update in database
	disputeSchema = domain.MapDisputeToSchema(dispute)
	if err := s.disputeRepo.Update(ctx, disputeSchema); err != nil {
		return fmt.Errorf("failed to update dispute: %w", err)
	}

	s.log.Logf("INFO dispute investigating: dispute_id=%s admin_id=%s", disputeID, adminID)

	return nil
}

// ResolveDispute resolves a dispute with refund or release
func (s *FinanceServiceImpl) ResolveDispute(
	ctx context.Context,
	disputeID, adminID uuid.UUID,
	outcome domain.DisputeStatus,
	refundAmount int64,
	reason, notes string,
) error {
	// Validate outcome
	if outcome != domain.DisputeStatusResolvedRefund && outcome != domain.DisputeStatusResolvedRelease {
		return domain.ErrInvalidDisputeResolution
	}

	// Get dispute
	disputeSchema, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}
	if disputeSchema == nil {
		return domain.ErrDisputeNotFound
	}

	dispute := domain.MapDisputeFromSchema(disputeSchema)

	// Validate refund amount
	if outcome == domain.DisputeStatusResolvedRefund && (refundAmount <= 0 || refundAmount > dispute.Amount) {
		return domain.ErrInvalidAmount
	}

	// Start transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Process refund if needed
	var transactionID *uuid.UUID
	if outcome == domain.DisputeStatusResolvedRefund {
		// Record refund transaction via ledger service using stored payment ID
		refundTx, err := s.RecordRefund(ctx, dispute.BookingID, dispute.PaymentID, refundAmount, dispute.Currency)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record refund: %w", err)
		}
		transactionID = &refundTx.ID
	}

	// Resolve dispute
	if err := dispute.Resolve(outcome, adminID, reason, refundAmount, notes); err != nil {
		tx.Rollback()
		return err
	}

	dispute.TransactionID = transactionID

	// Update dispute
	disputeSchema = domain.MapDisputeToSchema(dispute)
	if err := s.disputeRepo.WithTx(tx).Update(ctx, disputeSchema); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update dispute: %w", err)
	}

	// Unfreeze wallet
	if err := s.walletRepo.WithTx(tx).UpdateStatus(ctx, dispute.WalletID, string(domain.WalletStatusActive)); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to unfreeze wallet: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Logf("INFO dispute resolved: dispute_id=%s outcome=%s refund_amount=%d admin_id=%s",
		disputeID, outcome, refundAmount, adminID)

	return nil
}

// CancelDispute cancels/withdraws a dispute
func (s *FinanceServiceImpl) CancelDispute(ctx context.Context, disputeID, cancelledByID uuid.UUID) error {
	// Get dispute
	disputeSchema, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}
	if disputeSchema == nil {
		return domain.ErrDisputeNotFound
	}

	dispute := domain.MapDisputeFromSchema(disputeSchema)

	// Start transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Cancel dispute
	if err := dispute.Cancel(cancelledByID); err != nil {
		tx.Rollback()
		return err
	}

	// Update dispute
	disputeSchema = domain.MapDisputeToSchema(dispute)
	if err := s.disputeRepo.WithTx(tx).Update(ctx, disputeSchema); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update dispute: %w", err)
	}

	// Unfreeze wallet
	if err := s.walletRepo.WithTx(tx).UpdateStatus(ctx, dispute.WalletID, string(domain.WalletStatusActive)); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to unfreeze wallet: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Logf("INFO dispute cancelled: dispute_id=%s cancelled_by=%s", disputeID, cancelledByID)

	return nil
}

// AddEvidence adds evidence to a dispute
func (s *FinanceServiceImpl) AddEvidence(
	ctx context.Context,
	disputeID uuid.UUID,
	userID uuid.UUID,
	evidenceType, url, description string,
) error {
	// Get dispute
	disputeSchema, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return fmt.Errorf("failed to get dispute: %w", err)
	}
	if disputeSchema == nil {
		return domain.ErrDisputeNotFound
	}

	dispute := domain.MapDisputeFromSchema(disputeSchema)

	// Check if user is involved in the booking (either guest or host)
	_, err = s.bookingPartyQuerier.GetBookingParty(ctx, dispute.BookingID, userID)
	if err != nil {
		return fmt.Errorf("unauthorized: only involved parties can add evidence: %w", err)
	}

	// Add evidence
	if err := dispute.AddEvidence(evidenceType, url, description, userID); err != nil {
		return err
	}

	// Update dispute
	disputeSchema = domain.MapDisputeToSchema(dispute)
	if err := s.disputeRepo.Update(ctx, disputeSchema); err != nil {
		return fmt.Errorf("failed to update dispute: %w", err)
	}

	s.log.Logf("INFO evidence added: dispute_id=%s type=%s uploaded_by=%s", disputeID, evidenceType, userID)

	return nil
}

// GetDispute retrieves a dispute by ID
func (s *FinanceServiceImpl) GetDispute(ctx context.Context, disputeID uuid.UUID) (*domain.Dispute, error) {
	disputeSchema, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute: %w", err)
	}
	if disputeSchema == nil {
		return nil, domain.ErrDisputeNotFound
	}

	return domain.MapDisputeFromSchema(disputeSchema), nil
}

// GetDisputeByBooking retrieves a dispute by booking ID
func (s *FinanceServiceImpl) GetDisputeByBooking(ctx context.Context, bookingID uuid.UUID) (*domain.Dispute, error) {
	disputeSchema, err := s.disputeRepo.GetByBookingID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute: %w", err)
	}
	if disputeSchema == nil {
		return nil, domain.ErrDisputeNotFound
	}

	return domain.MapDisputeFromSchema(disputeSchema), nil
}

// ListDisputes lists disputes with optional status filter
func (s *FinanceServiceImpl) ListDisputes(
	ctx context.Context,
	status *domain.DisputeStatus,
	limit, offset int,
) ([]*domain.Dispute, error) {
	var disputeSchemas []*schema.Dispute
	var err error

	if status != nil {
		disputeSchemas, err = s.disputeRepo.ListByStatus(ctx, string(*status), limit, offset)
	} else {
		disputeSchemas, err = s.disputeRepo.ListAll(ctx, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list disputes: %w", err)
	}

	disputes := make([]*domain.Dispute, len(disputeSchemas))
	for i, ds := range disputeSchemas {
		disputes[i] = domain.MapDisputeFromSchema(ds)
	}

	return disputes, nil
}

// ListUserDisputes lists disputes filed by a specific user
func (s *FinanceServiceImpl) ListUserDisputes(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*domain.Dispute, error) {
	disputeSchemas, err := s.disputeRepo.ListByFiledBy(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list user disputes: %w", err)
	}

	disputes := make([]*domain.Dispute, len(disputeSchemas))
	for i, ds := range disputeSchemas {
		disputes[i] = domain.MapDisputeFromSchema(ds)
	}

	return disputes, nil
}
