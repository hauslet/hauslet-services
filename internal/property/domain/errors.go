package domain

import "errors"

var (
	ErrInvalidPropertyID     = errors.New("invalid property ID")
	ErrPropertyNotFound      = errors.New("property not found")
	ErrPropertyAlreadyExists = errors.New("property already exists")
	ErrInvalidListingID      = errors.New("invalid listing ID")
	ErrInvalidSlug           = errors.New("invalid slug")
	ErrListingNotFound       = errors.New("listing not found")
	ErrListingAlreadyExists  = errors.New("listing already exists")
	ErrInvalidLocation       = errors.New("invalid location coordinates")
	ErrInvalidOwnerID        = errors.New("invalid owner ID")
	ErrDuplicateSlug         = errors.New("listing slug already exists")
	ErrListingNotPublishable = errors.New("listing cannot be published in current state")
	ErrMediaNotFound         = errors.New("media not found")
	ErrInvalidMediaType      = errors.New("invalid media type")
	ErrNoPrimaryMedia        = errors.New("listing must have at least one primary media")

	// Authorization errors
	ErrForbidden            = errors.New("forbidden: insufficient permissions")
	ErrUnauthorized         = errors.New("unauthorized")

	// Media operation errors
	ErrInvalidMediaInput    = errors.New("invalid media input")
	ErrMediaUploadFailed    = errors.New("media upload failed")
	ErrTooManyMediaItems    = errors.New("too many media items")
)
