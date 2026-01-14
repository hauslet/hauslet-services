package domain

import "errors"

var (
	// Event errors
	ErrEventNotFound      = errors.New("calendar event not found")
	ErrEventAlreadyExists = errors.New("calendar event already exists")
	ErrInvalidEventType   = errors.New("invalid event type")
	ErrInvalidEventStatus = errors.New("invalid event status")
	ErrEventConflict      = errors.New("event conflicts with existing event")

	// Availability errors

	// Configuration errors
	ErrConfigNotFound    = errors.New("calendar configuration not found")
	ErrInvalidTimezone   = errors.New("invalid timezone")
	ErrInvalidBufferTime = errors.New("invalid buffer time")

	// Permission errors
	ErrUnauthorized = errors.New("unauthorized to access this resource")
	ErrAccessDenied = errors.New("access denied")

	// Profile errors
	ErrProfileHooksNotConfigured = errors.New("profile hooks not configured")

	ErrCannotCancelEvent     = errors.New("event cannot be cancelled at this time")
	ErrEventAlreadyCancelled = errors.New("event already cancelled")
	ErrEventAlreadyCompleted = errors.New("event already completed")

	// Recurring event errors
	ErrInvalidRecurrencePattern = errors.New("invalid recurrence pattern")
	ErrRecurringEventNotFound   = errors.New("recurring event pattern not found")

	// Self-interaction errors
	ErrCannotRequestOwnShowing    = errors.New("cannot request showings for their own listings")
	ErrCannotRegisterOwnOpenHouse = errors.New("cannot register for their own open house events")
)
