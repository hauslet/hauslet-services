package domain

import "errors"

var (
	// Lead entity errors
	ErrInvalidLeadID    = errors.New("invalid lead ID")
	ErrLeadNotFound     = errors.New("lead not found")
	ErrDuplicateLead    = errors.New("duplicate lead detected: lead already exists for this email and listing")
	ErrLeadAlreadyExists = errors.New("lead already exists")

	// Listing errors
	ErrInvalidListingID = errors.New("invalid listing ID")
	ErrListingNotFound  = errors.New("listing not found for lead")

	// Validation errors
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrInvalidPhoneNumber = errors.New("invalid phone number format")
	ErrInvalidMessage     = errors.New("invalid message: message must be between 10 and 5000 characters")
	ErrMessageTooShort    = errors.New("message too short: minimum 10 characters required")
	ErrMessageTooLong     = errors.New("message too long: maximum 5000 characters allowed")

	// Rate limiting errors
	ErrRateLimitExceeded     = errors.New("rate limit exceeded: too many leads from this source")
	ErrEmailRateLimitReached = errors.New("rate limit exceeded: maximum leads per email per day reached")
	ErrIPRateLimitReached    = errors.New("rate limit exceeded: maximum leads per IP address per day reached")
	ErrListingRateLimitReached = errors.New("rate limit exceeded: maximum leads for this listing reached")

	// Spam detection errors
	ErrSpamDetected = errors.New("lead blocked: spam detected")
	ErrHighSpamScore = errors.New("lead has high spam score and requires review")

	// Assignment errors
	ErrCannotAssignLead    = errors.New("cannot assign lead")
	ErrLeadAlreadyAssigned = errors.New("lead is already assigned to an agent")
	ErrCannotAssignSpam    = errors.New("cannot assign spam lead")
	ErrInvalidAssignee     = errors.New("invalid assignee ID")

	// Authorization errors
	ErrUnauthorized = errors.New("unauthorized: authentication required")
	ErrForbidden    = errors.New("forbidden: insufficient permissions")
	ErrNotOwner     = errors.New("forbidden: only listing owner can perform this action")
	ErrNotBusinessMember = errors.New("forbidden: user is not a member of the business")

	// General errors
	ErrInvalidInput = errors.New("invalid input")
	ErrInternalError = errors.New("internal server error")
)
