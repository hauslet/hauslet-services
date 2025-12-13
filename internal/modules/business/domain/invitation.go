package domain

import (
	"time"

	"github.com/google/uuid"
)

// BusinessInvitation represents an invitation to join a business
type BusinessInvitation struct {
	ID         uuid.UUID
	BusinessID uuid.UUID

	// Invitee
	Email  string
	UserID *uuid.UUID // Set if user exists in the system

	// Invitation details
	Role        MemberRole
	Permissions MemberPermissions

	// Lifecycle
	InvitedBy  uuid.UUID
	Token      string
	Status     InvitationStatus
	ExpiresAt  time.Time
	AcceptedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsExpired checks if the invitation has expired
func (i *BusinessInvitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsPending checks if the invitation is still pending
func (i *BusinessInvitation) IsPending() bool {
	return i.Status == InvitationPending && !i.IsExpired()
}

// CanBeAccepted checks if the invitation can be accepted
func (i *BusinessInvitation) CanBeAccepted() bool {
	return i.Status == InvitationPending && !i.IsExpired()
}

// CanBeRevoked checks if the invitation can be revoked
func (i *BusinessInvitation) CanBeRevoked() bool {
	return i.Status == InvitationPending
}

// Accept marks the invitation as accepted
func (i *BusinessInvitation) Accept() error {
	if !i.CanBeAccepted() {
		if i.IsExpired() {
			return ErrInvitationExpired
		}
		return ErrInvitationAlreadyAccepted
	}

	now := time.Now()
	i.Status = InvitationAccepted
	i.AcceptedAt = &now
	return nil
}

// Decline marks the invitation as declined
func (i *BusinessInvitation) Decline() error {
	if i.Status != InvitationPending {
		return ErrInvitationAlreadyDeclined
	}

	i.Status = InvitationDeclined
	return nil
}

// Revoke marks the invitation as revoked
func (i *BusinessInvitation) Revoke() error {
	if !i.CanBeRevoked() {
		return ErrInvitationAlreadyAccepted
	}

	i.Status = InvitationRevoked
	return nil
}
