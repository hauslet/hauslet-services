package schema

// BookingStatus mirrors the domain layer statuses.
type BookingStatus string

const (
	BookingStatusDraft           BookingStatus = "draft"
	BookingStatusPendingApproval BookingStatus = "pending_host_approval"
	BookingStatusAwaitingPayment BookingStatus = "awaiting_payment"
	BookingStatusPaymentFailed   BookingStatus = "payment_failed"
	BookingStatusConfirmed       BookingStatus = "confirmed"
	BookingStatusActive          BookingStatus = "active"
	BookingStatusCompleted       BookingStatus = "completed"
	BookingStatusCancelled       BookingStatus = "cancelled"
	BookingStatusArchived        BookingStatus = "archived"
	BookingStatusDisputed        BookingStatus = "disputed"
	BookingStatusSettled         BookingStatus = "settled"
)

// BookingType tracks instant vs request bookings.
type BookingType string

const (
	BookingInstant BookingType = "instant"
	BookingRequest BookingType = "request"
)
