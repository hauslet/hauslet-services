package domain

import "errors"

var (
	ErrInvalidUserID           = errors.New("invalid userID")
	ErrProfileAlreadyExists    = errors.New("profile already exists")
	ErrProfileNotFound         = errors.New("profile not found")
	ErrMaxPhoneNumbersReached  = errors.New("maximum of 2 phone numbers allowed")
	ErrInvalidBadge            = errors.New("invalid badge")
	ErrInvalidRating           = errors.New("invalid rating value")
	ErrTravelCompanionNotFound = errors.New("travel companion not found")
	ErrInvalidProfileID        = errors.New("invalid profile ID")
	ErrInvalidUserType         = errors.New("invalid user type")
	ErrSupplyRoleRequired      = errors.New("supply user type required")
)
