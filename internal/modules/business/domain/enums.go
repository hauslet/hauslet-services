package domain

// BusinessType represents the type of business entity
type BusinessType string

const (
	TypePropertyManagement BusinessType = "property_management"
	TypeRealEstateAgency   BusinessType = "real_estate_agency"
	TypePropertyDeveloper  BusinessType = "property_developer"
	TypeInvestmentFirm     BusinessType = "investment_firm"
	TypeIndividual         BusinessType = "individual"
)

// MemberRole represents the role of a business member
type MemberRole string

const (
	RoleOwner      MemberRole = "owner"
	RoleAdmin      MemberRole = "admin"
	RoleMember     MemberRole = "member"
	RoleViewer     MemberRole = "viewer"
	RoleAccountant MemberRole = "accountant"
)

// InvitationStatus represents the status of a business invitation
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationDeclined InvitationStatus = "declined"
	InvitationExpired  InvitationStatus = "expired"
	InvitationRevoked  InvitationStatus = "revoked"
)