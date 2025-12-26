package domain

import (
	"time"

	"github.com/google/uuid"
)

// Disbursement represents an actual bank transfer to a host
// Tracks the status and retry logic for payouts
type Disbursement struct {
	ID               uuid.UUID
	WalletID         uuid.UUID
	TransactionID    uuid.UUID
	Amount           int64 // Amount in minor currency units
	Currency         string
	Provider         string // Payment provider (e.g., "paystack", "flutterwave")
	TransferCode     *string
	ProviderResponse *string
	Status           DisbursementStatus
	Attempts         int
	NextRetryAt      *time.Time
	CompletedAt      *time.Time
	FailureReason    *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

const (
	// MaxRetryAttempts is the maximum number of retry attempts for a disbursement
	MaxRetryAttempts = 6
)

// MarkProcessing marks the disbursement as being processed
func (d *Disbursement) MarkProcessing() {
	d.Status = DisbursementStatusProcessing
	d.UpdatedAt = time.Now()
}

// MarkCompleted marks the disbursement as successfully completed
func (d *Disbursement) MarkCompleted(transferCode, providerResponse string) {
	d.Status = DisbursementStatusCompleted
	d.TransferCode = &transferCode
	d.ProviderResponse = &providerResponse
	now := time.Now()
	d.CompletedAt = &now
	d.UpdatedAt = now
}

// MarkFailed marks the disbursement as failed and schedules retry
func (d *Disbursement) MarkFailed(failureReason, providerResponse string) error {
	d.Status = DisbursementStatusFailed
	d.FailureReason = &failureReason
	d.ProviderResponse = &providerResponse
	d.Attempts++
	d.UpdatedAt = time.Now()

	// Calculate next retry time with exponential backoff
	if d.Attempts < MaxRetryAttempts {
		nextRetry := d.CalculateNextRetry()
		d.NextRetryAt = &nextRetry
	} else {
		// Max retries exceeded, no more retries
		return ErrMaxRetriesExceeded
	}

	return nil
}

// MarkCancelled marks the disbursement as cancelled
func (d *Disbursement) MarkCancelled(reason string) {
	d.Status = DisbursementStatusCancelled
	d.FailureReason = &reason
	d.UpdatedAt = time.Now()
}

// CalculateNextRetry returns the next retry time using exponential backoff
// Retry intervals: 1min, 5min, 15min, 1hr, 6hr, 24hr
func (d *Disbursement) CalculateNextRetry() time.Time {
	var delay time.Duration

	switch d.Attempts {
	case 1:
		delay = 1 * time.Minute
	case 2:
		delay = 5 * time.Minute
	case 3:
		delay = 15 * time.Minute
	case 4:
		delay = 1 * time.Hour
	case 5:
		delay = 6 * time.Hour
	default:
		delay = 24 * time.Hour
	}

	return time.Now().Add(delay)
}

// CanRetry returns true if the disbursement can be retried
func (d *Disbursement) CanRetry() bool {
	return d.Status == DisbursementStatusFailed &&
		d.Attempts < MaxRetryAttempts &&
		d.NextRetryAt != nil &&
		time.Now().After(*d.NextRetryAt)
}

// IsCompleted returns true if the disbursement completed successfully
func (d *Disbursement) IsCompleted() bool {
	return d.Status == DisbursementStatusCompleted
}

// IsFailed returns true if the disbursement failed
func (d *Disbursement) IsFailed() bool {
	return d.Status == DisbursementStatusFailed
}

// IsPending returns true if the disbursement is pending
func (d *Disbursement) IsPending() bool {
	return d.Status == DisbursementStatusPending
}
