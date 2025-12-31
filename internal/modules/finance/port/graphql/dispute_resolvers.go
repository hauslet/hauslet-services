package graphql

import (
	"context"
	"fmt"
	"hauslet/internal/modules/finance/domain"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// ============================================================================
// Dispute Query Resolvers
// ============================================================================

// Dispute retrieves a dispute by ID (admin or involved party)
func (r *Resolver) Dispute(ctx context.Context, id string) (*domain.Dispute, error) {
	disputeID, err := uuid.Parse(id)
	if err != nil {
		r.log.Error("invalid dispute ID", "id", id, "error", err)
		return nil, fmt.Errorf("invalid dispute ID")
	}

	dispute, err := r.financeService.GetDispute(ctx, disputeID)
	if err != nil {
		if err == domain.ErrDisputeNotFound {
			return nil, nil
		}
		r.log.Error("failed to get dispute", "id", id, "error", err)
		return nil, err
	}

	// Check authorization: admin or involved party
	v := viewer.FromContext(ctx)
	if v == nil {
		return nil, ErrUnauthorized
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if !isAdminRole(v.Role) && userID != dispute.FiledByID {
		r.log.Warn("unauthorized access to dispute", "id", id, "user", userID)
		return nil, fmt.Errorf("unauthorized")
	}

	return dispute, nil
}

// DisputeByBooking retrieves a dispute by booking ID (admin or involved party)
func (r *Resolver) DisputeByBooking(ctx context.Context, bookingID string) (*domain.Dispute, error) {
	bid, err := uuid.Parse(bookingID)
	if err != nil {
		r.log.Error("invalid booking ID", "id", bookingID, "error", err)
		return nil, fmt.Errorf("invalid booking ID")
	}

	dispute, err := r.financeService.GetDisputeByBooking(ctx, bid)
	if err != nil {
		if err == domain.ErrDisputeNotFound {
			return nil, nil
		}
		r.log.Error("failed to get dispute for booking", "id", bookingID, "error", err)
		return nil, err
	}

	// Check authorization: admin or involved party
	v := viewer.FromContext(ctx)
	if v == nil {
		return nil, ErrUnauthorized
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if !isAdminRole(v.Role) && userID != dispute.FiledByID {
		r.log.Warn("unauthorized access to dispute for booking", "id", bookingID, "user", userID)
		return nil, fmt.Errorf("unauthorized")
	}

	return dispute, nil
}

// Disputes lists all disputes with optional status filter (admin only)
func (r *Resolver) Disputes(
	ctx context.Context,
	status *domain.DisputeStatus,
	limit, offset *int,
) ([]*domain.Dispute, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	l := 50 // default limit
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0 // default offset
	if offset != nil && *offset > 0 {
		o = *offset
	}

	disputes, err := r.financeService.ListDisputes(ctx, status, l, o)
	if err != nil {
		r.log.Error("failed to list disputes", "error", err)
		return nil, err
	}

	return disputes, nil
}

// MyDisputes lists disputes filed by the current user
func (r *Resolver) MyDisputes(ctx context.Context, limit, offset *int) ([]*domain.Dispute, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	l := 50 // default limit
	if limit != nil && *limit > 0 {
		l = *limit
	}

	o := 0 // default offset
	if offset != nil && *offset > 0 {
		o = *offset
	}

	disputes, err := r.financeService.ListUserDisputes(ctx, userID, l, o)
	if err != nil {
		r.log.Error("failed to list disputes for user", "user", userID, "error", err)
		return nil, err
	}

	return disputes, nil
}

// ============================================================================
// Dispute Mutation Resolvers
// ============================================================================

// FileDisputeInput represents the input for filing a dispute
type FileDisputeInput struct {
	BookingID   string
	Reason      domain.DisputeReason
	Description string
	Amount      int64
	Currency    string
}

// FileDispute creates a new dispute
func (r *Resolver) FileDispute(ctx context.Context, input FileDisputeInput) (*domain.Dispute, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bookingID, err := uuid.Parse(input.BookingID)
	if err != nil {
		r.log.Error("invalid booking ID", "id", input.BookingID, "error", err)
		return nil, fmt.Errorf("invalid booking ID")
	}

	// Service handles authorization and determines party internally
	dispute, err := r.financeService.FileDispute(
		ctx,
		bookingID,
		userID,
		input.Reason,
		input.Description,
		input.Amount,
		input.Currency,
	)
	if err != nil {
		r.log.Error("failed to file dispute", "error", err)
		return nil, err
	}

	r.log.Info(" dispute filed", "id", dispute.ID, "booking_id", bookingID, "user_id", userID)

	return dispute, nil
}

// InvestigateDispute marks a dispute as under investigation (admin only)
func (r *Resolver) InvestigateDispute(ctx context.Context, disputeID string) (*domain.Dispute, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	adminID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	did, err := uuid.Parse(disputeID)
	if err != nil {
		r.log.Error("invalid dispute ID", "id", disputeID, "error", err)
		return nil, fmt.Errorf("invalid dispute ID")
	}

	if err := r.financeService.InvestigateDispute(ctx, did, adminID); err != nil {
		r.log.Error("failed to investigate dispute", "id", disputeID, "error", err)
		return nil, err
	}

	// Return updated dispute
	dispute, err := r.financeService.GetDispute(ctx, did)
	if err != nil {
		r.log.Error("failed to get dispute after investigation", "error", err)
		return nil, err
	}

	return dispute, nil
}

// ResolveDisputeInput represents the input for resolving a dispute
type ResolveDisputeInput struct {
	DisputeID    string
	Outcome      domain.DisputeStatus
	RefundAmount int64
	Reason       string
	Notes        string
}

// ResolveDispute resolves a dispute (admin only)
func (r *Resolver) ResolveDispute(ctx context.Context, input ResolveDisputeInput) (*domain.Dispute, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	adminID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	did, err := uuid.Parse(input.DisputeID)
	if err != nil {
		r.log.Error("invalid dispute ID", "id", input.DisputeID, "error", err)
		return nil, fmt.Errorf("invalid dispute ID")
	}

	if err := r.financeService.ResolveDispute(
		ctx,
		did,
		adminID,
		input.Outcome,
		input.RefundAmount,
		input.Reason,
		input.Notes,
	); err != nil {
		r.log.Error("failed to resolve dispute", "id", input.DisputeID, "error", err)
		return nil, err
	}

	// Return updated dispute
	dispute, err := r.financeService.GetDispute(ctx, did)
	if err != nil {
		r.log.Error("failed to get dispute after resolution", "error", err)
		return nil, err
	}

	r.log.Info(" dispute resolved", "id", dispute.ID, "outcome", input.Outcome, "admin_id", adminID)

	return dispute, nil
}

// CancelDispute cancels/withdraws a dispute (disputing party only)
func (r *Resolver) CancelDispute(ctx context.Context, disputeID string) (*domain.Dispute, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	did, err := uuid.Parse(disputeID)
	if err != nil {
		r.log.Error("invalid dispute ID", "id", disputeID, "error", err)
		return nil, fmt.Errorf("invalid dispute ID")
	}

	// Get dispute to check ownership
	dispute, err := r.financeService.GetDispute(ctx, did)
	if err != nil {
		r.log.Error("failed to get dispute", "id", disputeID, "error", err)
		return nil, err
	}

	// Only the person who filed the dispute can cancel it (unless admin)
	v := viewer.FromContext(ctx)
	if !isAdminRole(v.Role) && dispute.FiledByID != userID {
		r.log.Warn("unauthorized cancellation attempt for dispute", "id", disputeID, "user_id", userID)
		return nil, fmt.Errorf("unauthorized: only the disputing party can cancel")
	}

	if err := r.financeService.CancelDispute(ctx, did, userID); err != nil {
		r.log.Error("failed to cancel dispute", "id", disputeID, "error", err)
		return nil, err
	}

	// Return updated dispute
	dispute, err = r.financeService.GetDispute(ctx, did)
	if err != nil {
		r.log.Error("failed to get dispute after cancellation", "error", err)
		return nil, err
	}

	r.log.Info(" dispute cancelled", "id", dispute.ID, "user_id", userID)
	return dispute, nil
}

// AddDisputeEvidenceInput represents the input for adding evidence to a dispute
type AddDisputeEvidenceInput struct {
	DisputeID   string
	Type        string
	URL         string
	Description string
}

// AddDisputeEvidence adds evidence to a dispute (involved parties only)
func (r *Resolver) AddDisputeEvidence(ctx context.Context, input AddDisputeEvidenceInput) (*domain.Dispute, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	did, err := uuid.Parse(input.DisputeID)
	if err != nil {
		r.log.Error("invalid dispute ID %s: %v", input.DisputeID, err)
		return nil, fmt.Errorf("invalid dispute ID")
	}

	// Service handles authorization internally
	if err := r.financeService.AddEvidence(
		ctx,
		did,
		userID,
		input.Type,
		input.URL,
		input.Description,
	); err != nil {
		r.log.Error("failed to add evidence to dispute %s: %v", input.DisputeID, err)
		return nil, err
	}

	// Return updated dispute
	dispute, err := r.financeService.GetDispute(ctx, did)
	if err != nil {
		r.log.Error("failed to get dispute after adding evidence", "error", err)
		return nil, err
	}

	r.log.Info(" evidence added to dispute", "id", dispute.ID, "type", input.Type, "user_id", userID)
	return dispute, nil
}
