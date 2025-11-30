package domain

import "errors"

var (
	ErrInvalidUserID          = errors.New("invalid userID")
	ErrProfileAlreadyExists   = errors.New("profile already exists")
	ErrProfileNotFound        = errors.New("profile not found")
	ErrMaxPhoneNumbersReached = errors.New("maximum of 2 phone numbers allowed")
	ErrInvalidBadge           = errors.New("invalid badge")
	ErrInvalidRating          = errors.New("invalid rating value")
)
