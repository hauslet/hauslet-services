package domain

import (
	"context"

	"github.com/google/uuid"
)

// LeadHooks defines the contract for interacting with the Leads module
type LeadHooks interface {
	// GetLeadParticipants returns the prospect (inquirer) and the owner/agent IDs
	GetLeadParticipants(ctx context.Context, leadID uuid.UUID) (prospectID, ownerID uuid.UUID, err error)

	// ValidateLeadAccess checks if a specific user is part of this lead
	ValidateLeadAccess(ctx context.Context, leadID, userID uuid.UUID) (bool, error)
}

// BookingHooks defines the contract for interacting with the Booking module
type BookingHooks interface {
	// GetBookingParticipants returns the guest and host IDs
	GetBookingParticipants(ctx context.Context, bookingID uuid.UUID) (guestID, hostID uuid.UUID, err error)

	// GetBookingState returns status to help AI decide if it can answer questions
	GetBookingState(ctx context.Context, bookingID uuid.UUID) (status string, err error)
}

// ProfileHooks defines the contract for user profile data
type ProfileHooks interface {
	// GetProfileData fetches minimal profile metadata for given user IDs
	GetProfileData(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]map[string]string, error)

	// GetUserContact fetches contact info (name, email) for a user
	GetUserContact(ctx context.Context, userID uuid.UUID) (*UserContact, error)
}

// UserContact holds contact information for notifications
type UserContact struct {
	UserID uuid.UUID
	Name   string
	Email  string
}
