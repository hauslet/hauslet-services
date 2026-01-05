package domain

import "errors"

var (
	// Validation errors
	ErrInvalidSessionID       = errors.New("invalid session ID")
	ErrInvalidInteractionType = errors.New("invalid interaction type")
	ErrInvalidEntityType      = errors.New("invalid entity type")
	ErrMissingEntityID        = errors.New("missing entity ID")

	// Service errors
	ErrInteractionNotFound = errors.New("interaction not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInternalError       = errors.New("internal error")
)
