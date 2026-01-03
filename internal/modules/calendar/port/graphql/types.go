package graphql

import (
	"time"

	"github.com/google/uuid"
)

type RequestShowingInput struct {
	ListingID uuid.UUID `json:"listingId"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Notes     *string   `json:"notes,omitempty"`
}

type CancelShowingInput struct {
	EventID uuid.UUID `json:"eventId"`
	Reason  *string   `json:"reason,omitempty"`
}

type RescheduleShowingInput struct {
	EventID      uuid.UUID `json:"eventId"`
	NewStartTime time.Time `json:"newStartTime"`
	NewEndTime   time.Time `json:"newEndTime"`
	Reason       *string   `json:"reason,omitempty"`
}

type CreateOpenHouseInput struct {
	ListingID            uuid.UUID  `json:"listingId"`
	StartTime            time.Time  `json:"startTime"`
	EndTime              time.Time  `json:"endTime"`
	Title                string     `json:"title"`
	Description          *string    `json:"description,omitempty"`
	MaxAttendees         int        `json:"maxAttendees"`
	RegistrationDeadline *time.Time `json:"registrationDeadline,omitempty"`
}

type RegisterOpenHouseInput struct {
	EventID uuid.UUID `json:"eventId"`
}
