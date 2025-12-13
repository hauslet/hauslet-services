package notification

import (
	"time"

	"hauslet/internal/modules/business/domain"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	// Invitation notifications
	NotificationTypeInvitation NotificationType = "invitation"

	// Member notifications
	NotificationTypeMemberAdded      NotificationType = "member_added"
	NotificationTypeMemberRemoved    NotificationType = "member_removed"
	NotificationTypeRoleChanged      NotificationType = "role_changed"
	NotificationTypePermissionChanged NotificationType = "permission_changed"

	// Business notifications
	NotificationTypeBusinessCreated NotificationType = "business_created"
	NotificationTypeBusinessUpdated NotificationType = "business_updated"
	NotificationTypeBusinessDeleted NotificationType = "business_deleted"

	// Invitation status notifications
	NotificationTypeInvitationAccepted NotificationType = "invitation_accepted"
	NotificationTypeInvitationDeclined NotificationType = "invitation_declined"
	NotificationTypeInvitationRevoked  NotificationType = "invitation_revoked"
	NotificationTypeInvitationExpired  NotificationType = "invitation_expired"
)

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject     string
	TextBody    string
	HTMLBody    string
	TemplateID  string // For email service provider templates
	Variables   map[string]interface{}
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// InvitationEmailData contains data for invitation emails
type InvitationEmailData struct {
	InvitationID   uuid.UUID
	BusinessID     uuid.UUID
	BusinessName   string
	InviteeEmail   string
	InviterName    string
	Role           domain.MemberRole
	Token          string
	ExpiresAt      time.Time
	AcceptURL      string
	DeclineURL     string
	BusinessLogoURL *string
}

// MemberAddedEmailData contains data for member added notifications
type MemberAddedEmailData struct {
	BusinessID     uuid.UUID
	BusinessName   string
	MemberName     string
	MemberEmail    string
	Role           domain.MemberRole
	AddedBy        string
	BusinessLogoURL *string
}

// MemberRemovedEmailData contains data for member removed notifications
type MemberRemovedEmailData struct {
	BusinessID     uuid.UUID
	BusinessName   string
	MemberName     string
	MemberEmail    string
	RemovedBy      string
	BusinessLogoURL *string
}

// RoleChangedEmailData contains data for role change notifications
type RoleChangedEmailData struct {
	BusinessID     uuid.UUID
	BusinessName   string
	MemberName     string
	MemberEmail    string
	OldRole        domain.MemberRole
	NewRole        domain.MemberRole
	ChangedBy      string
	BusinessLogoURL *string
}

// InvitationAcceptedEmailData contains data for invitation accepted notifications
type InvitationAcceptedEmailData struct {
	BusinessID     uuid.UUID
	BusinessName   string
	AcceptedBy     string
	AcceptedEmail  string
	Role           domain.MemberRole
	BusinessLogoURL *string
}

// InvitationDeclinedEmailData contains data for invitation declined notifications
type InvitationDeclinedEmailData struct {
	BusinessID     uuid.UUID
	BusinessName   string
	DeclinedBy     string
	DeclinedEmail  string
	BusinessLogoURL *string
}

// BusinessCreatedEmailData contains data for business creation notifications
type BusinessCreatedEmailData struct {
	BusinessID      uuid.UUID
	BusinessName    string
	CreatorName     string
	CreatorEmail    string
	BusinessLogoURL *string
	DashboardURL    string
}

// NotificationEvent represents a notification event
type NotificationEvent struct {
	ID        uuid.UUID
	Type      NotificationType
	Data      interface{}
	CreatedAt time.Time
}
