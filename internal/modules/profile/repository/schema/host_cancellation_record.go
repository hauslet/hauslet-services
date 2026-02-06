package schema

import (
	"time"

	"github.com/google/uuid"
)

// HostCancellationRecord tracks host cancellations for penalty calculation.
type HostCancellationRecord struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID        uuid.UUID `gorm:"type:uuid;not null;index:idx_host_cancellations_host_id"`
	BookingID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	CancelledAt   time.Time `gorm:"not null;index:idx_host_cancellations_cancelled_at"`
	PenaltyAmount int64     `gorm:"not null;default:0"` // Minor units
	PenaltyPaid   bool      `gorm:"not null;default:false"`
	WarningSent   bool      `gorm:"not null;default:false"`
	Reason        string    `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
}

func (HostCancellationRecord) TableName() string {
	return "host_cancellation_records"
}
