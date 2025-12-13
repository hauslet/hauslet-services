package domain

import "errors"

var (
	ErrBusinessNotFound           = errors.New("business not found")
	ErrBusinessAlreadyExists      = errors.New("business already exists")
	ErrInvalidBusinessID          = errors.New("invalid business ID")
	ErrInvalidSlug                = errors.New("invalid business slug")
	ErrSlugAlreadyExists          = errors.New("business slug already exists")
	ErrMemberNotFound             = errors.New("business member not found")
	ErrMemberAlreadyExists        = errors.New("user is already a member of this business")
	ErrInsufficientPermissions    = errors.New("insufficient permissions for this operation")
	ErrCannotRemoveOwner          = errors.New("cannot remove the business owner")
	ErrCannotModifyOwnRole        = errors.New("cannot modify your own role")
	ErrMustHaveOneOwner           = errors.New("business must have at least one owner")
	ErrInvitationNotFound         = errors.New("invitation not found")
	ErrInvitationExpired          = errors.New("invitation has expired")
	ErrInvitationAlreadyAccepted  = errors.New("invitation has already been accepted")
	ErrInvitationAlreadyDeclined  = errors.New("invitation has already been declined")
	ErrInvitationRevoked          = errors.New("invitation has been revoked")
	ErrInvalidInvitationToken     = errors.New("invalid invitation token")
	ErrUserAlreadyInvited         = errors.New("user has already been invited")
	ErrInvalidBusinessType        = errors.New("invalid business type")
	ErrInvalidMemberRole          = errors.New("invalid member role")
	ErrInvalidPermissions         = errors.New("invalid permissions")
	ErrBusinessInactive           = errors.New("business is inactive")
)