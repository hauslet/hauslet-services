package domain

import (
	"time"

	"github.com/google/uuid"
)

// Dispute represents a financial dispute for a booking
type Dispute struct {
	ID          uuid.UUID
	BookingID   uuid.UUID
	WalletID    uuid.UUID // The escrow wallet that was frozen
	PaymentID   uuid.UUID // The payment associated with this booking
	FiledBy     DisputeParty
	FiledByID   uuid.UUID // User ID of who filed the dispute
	Reason      DisputeReason
	Status      DisputeStatus
	Description string
	Amount      int64  // Amount in dispute (in minor currency units)
	Currency    string

	// Evidence and resolution
	Evidence       []DisputeEvidence
	AdminNotes     *string
	Resolution     *DisputeResolution
	ResolvedByID   *uuid.UUID // Admin user ID who resolved
	ResolvedAt     *time.Time
	RefundAmount   *int64 // If resolved with refund, amount refunded
	TransactionID  *uuid.UUID // Transaction created for resolution

	CreatedAt time.Time
	UpdatedAt time.Time
}

// DisputeEvidence represents evidence submitted for a dispute
type DisputeEvidence struct {
	Type        string    // photo, document, message
	URL         string    // S3 URL to the evidence file
	Description string
	UploadedBy  uuid.UUID // User ID
	UploadedAt  time.Time
}

// DisputeResolution contains the resolution details
type DisputeResolution struct {
	Outcome     DisputeStatus // resolved_refund or resolved_release
	Reason      string        // Admin's reasoning
	RefundAmount int64        // Amount to refund (if applicable)
	Notes       string        // Additional notes
	ResolvedAt  time.Time
}

// CanBeUpdated checks if dispute can still be modified
func (d *Dispute) CanBeUpdated() bool {
	return d.Status == DisputeStatusOpen || d.Status == DisputeStatusInvestigating
}

// IsResolved checks if dispute has been resolved
func (d *Dispute) IsResolved() bool {
	return d.Status == DisputeStatusResolvedRefund ||
		d.Status == DisputeStatusResolvedRelease ||
		d.Status == DisputeStatusCancelled
}

// MarkInvestigating updates status to investigating
func (d *Dispute) MarkInvestigating() error {
	if d.Status != DisputeStatusOpen {
		return ErrDisputeAlreadyInProgress
	}
	d.Status = DisputeStatusInvestigating
	d.UpdatedAt = time.Now()
	return nil
}

// Resolve resolves the dispute with a specific outcome
func (d *Dispute) Resolve(outcome DisputeStatus, resolvedByID uuid.UUID, reason string, refundAmount int64, notes string) error {
	if !d.CanBeUpdated() {
		return ErrDisputeAlreadyResolved
	}

	if outcome != DisputeStatusResolvedRefund && outcome != DisputeStatusResolvedRelease {
		return ErrInvalidDisputeResolution
	}

	now := time.Now()
	d.Status = outcome
	d.ResolvedByID = &resolvedByID
	d.ResolvedAt = &now
	d.UpdatedAt = now

	if outcome == DisputeStatusResolvedRefund {
		d.RefundAmount = &refundAmount
	}

	d.Resolution = &DisputeResolution{
		Outcome:      outcome,
		Reason:       reason,
		RefundAmount: refundAmount,
		Notes:        notes,
		ResolvedAt:   now,
	}

	return nil
}

// Cancel cancels/withdraws the dispute
func (d *Dispute) Cancel(cancelledByID uuid.UUID) error {
	if !d.CanBeUpdated() {
		return ErrDisputeAlreadyResolved
	}

	now := time.Now()
	d.Status = DisputeStatusCancelled
	d.ResolvedByID = &cancelledByID
	d.ResolvedAt = &now
	d.UpdatedAt = now

	return nil
}

// AddEvidence adds evidence to the dispute
func (d *Dispute) AddEvidence(evidenceType, url, description string, uploadedBy uuid.UUID) error {
	if !d.CanBeUpdated() {
		return ErrDisputeAlreadyResolved
	}

	if d.Evidence == nil {
		d.Evidence = []DisputeEvidence{}
	}

	d.Evidence = append(d.Evidence, DisputeEvidence{
		Type:        evidenceType,
		URL:         url,
		Description: description,
		UploadedBy:  uploadedBy,
		UploadedAt:  time.Now(),
	})

	d.UpdatedAt = time.Now()
	return nil
}
