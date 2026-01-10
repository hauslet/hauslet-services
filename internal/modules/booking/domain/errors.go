package domain

import "errors"

var (
	ErrBookingNotFound      = errors.New("booking not found")
	ErrDatesUnavailable     = errors.New("dates not available")
	ErrMinimumStayNotMet    = errors.New("booking does not meet minimum stay requirement")
	ErrMaximumStayExceeded  = errors.New("booking exceeds maximum stay limit")
	ErrGuestProfileNotFound = errors.New("guest profile not found")
	ErrGuestCountExceeded   = errors.New("guest count exceeds property capacity")
	ErrBookingInPast        = errors.New("cannot create booking in the past")
	ErrLeadTimeNotMet       = errors.New("booking does not meet lead time requirement")
	ErrInvalidDateRange     = errors.New("checkout must be after checkin")
	ErrUnauthorized         = errors.New("unauthorized to access booking")
	ErrCannotConfirm        = errors.New("booking cannot be confirmed")
	ErrCannotCancel         = errors.New("booking cannot be cancelled")
	ErrBookingExpired       = errors.New("booking hold has expired")
	ErrCannotBePaid         = errors.New("booking cannot be paid")
	ErrTooCloseToCheckIn    = errors.New("too close to check-in for request-to-book")
	ErrCannotCheckIn        = errors.New("booking cannot be checked in")
	ErrCannotCheckOut       = errors.New("booking cannot be checked out")
)
