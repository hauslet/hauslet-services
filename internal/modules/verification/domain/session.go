package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// VerificationSession is the aggregate root for all verification types
type VerificationSession struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TargetID   *uuid.UUID       // ID of the entity being verified (if different from UserID)
	TargetType TargetType       // user, listing
	Type       VerificationType // identity, phone, address, business
	Tier       VerificationTier
	Status     SessionStatus

	// Type-specific data (union type - only one populated based on Type)
	Data VerificationData

	Country string // ISO 3166-1 alpha-2

	// Attempts tracking
	MaxAttempts   int
	AttemptsUsed  int
	LastAttemptAt *time.Time

	// Resolution
	ApprovedAt      *time.Time
	RejectedAt      *time.Time
	RejectionReason *RejectionReason
	RejectionNotes  *string

	// Metadata
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
	CompletedAt *time.Time

	// Client context
	IPAddress *string
	UserAgent *string
}

// NewVerificationSession creates a new verification session
func NewVerificationSession(
	userID uuid.UUID,
	vType VerificationType,
	tier VerificationTier,
	data VerificationData,
	country string,
	targetID *uuid.UUID, // Optional target, defaults to UserID for user verification
) (*VerificationSession, error) {
	// Validate inputs
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID is required")
	}
	if !vType.IsValid() {
		return nil, fmt.Errorf("invalid verification type: %s", vType)
	}
	if !tier.IsValid() {
		return nil, ErrInvalidTier
	}
	if err := data.ValidateForType(vType); err != nil {
		return nil, fmt.Errorf("invalid verification data: %w", err)
	}
	if country == "" || len(country) != 2 {
		return nil, ErrInvalidCountryCode
	}

	now := time.Now()

	// Set default expiry based on verification type
	var expiryDuration time.Duration
	switch vType {
	case VerificationPhone:
		expiryDuration = 24 * time.Hour // Phone OTP expires in 24 hours
	case VerificationIdentity, VerificationAddress, VerificationBusiness:
		expiryDuration = 30 * 24 * time.Hour // 30 days for document-based verification
	default:
		expiryDuration = 7 * 24 * time.Hour // Default 7 days
	}

	// Determine TargetType
	var targetType TargetType
	if vType == VerificationListing {
		targetType = TargetListing
		if targetID == nil {
			return nil, fmt.Errorf("targetID required for listing verification")
		}
	} else {
		targetType = TargetUser
		// For user verification, target is the user if not specified
		if targetID == nil {
			targetID = &userID
		}
	}

	return &VerificationSession{
		ID:           uuid.New(),
		UserID:       userID,
		TargetID:     targetID,
		TargetType:   targetType,
		Type:         vType,
		Tier:         tier,
		Status:       SessionPending,
		Data:         data,
		Country:      country,
		MaxAttempts:  3, // Default max attempts
		AttemptsUsed: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    now.Add(expiryDuration),
	}, nil
}

// CanSubmitAttempt checks if a new attempt can be submitted
func (s *VerificationSession) CanSubmitAttempt() error {
	if s.Status.IsFinal() {
		return ErrSessionAlreadyFinal
	}
	if s.IsExpired() {
		return ErrSessionExpired
	}
	if s.AttemptsUsed >= s.MaxAttempts {
		return ErrMaxAttemptsExceeded
	}
	return nil
}

// StartAttempt transitions session to in_progress
func (s *VerificationSession) StartAttempt() error {
	if err := s.CanSubmitAttempt(); err != nil {
		return err
	}

	s.Status = SessionInProgress
	s.AttemptsUsed++
	now := time.Now()
	s.LastAttemptAt = &now
	s.UpdatedAt = now

	return nil
}

// Approve marks the session as approved
func (s *VerificationSession) Approve() error {
	if s.Status.IsFinal() {
		return ErrSessionAlreadyFinal
	}

	now := time.Now()
	s.Status = SessionApproved
	s.ApprovedAt = &now
	s.CompletedAt = &now
	s.UpdatedAt = now

	return nil
}

// Reject marks the session as rejected
func (s *VerificationSession) Reject(reason RejectionReason, notes *string) error {
	if s.Status.IsFinal() {
		return ErrSessionAlreadyFinal
	}
	if !reason.IsValid() {
		return fmt.Errorf("invalid rejection reason: %s", reason)
	}

	now := time.Now()
	s.Status = SessionRejected
	s.RejectedAt = &now
	s.RejectionReason = &reason
	s.RejectionNotes = notes
	s.CompletedAt = &now
	s.UpdatedAt = now

	return nil
}

// MarkExpired marks the session as expired
func (s *VerificationSession) MarkExpired() error {
	if s.Status.IsFinal() {
		return ErrSessionAlreadyFinal
	}

	now := time.Now()
	s.Status = SessionExpired
	s.CompletedAt = &now
	s.UpdatedAt = now

	return nil
}

// IsExpired checks if the session has expired
func (s *VerificationSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// CanRetry checks if the user can retry after a failed attempt
func (s *VerificationSession) CanRetry() bool {
	return !s.Status.IsFinal() && s.AttemptsUsed < s.MaxAttempts && !s.IsExpired()
}

// AttemptsRemaining returns the number of attempts left
func (s *VerificationSession) AttemptsRemaining() int {
	remaining := s.MaxAttempts - s.AttemptsUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// UpdateData updates the verification data
func (s *VerificationSession) UpdateData(data VerificationData) error {
	if err := data.ValidateForType(s.Type); err != nil {
		return fmt.Errorf("invalid data for verification type %s: %w", s.Type, err)
	}
	s.Data = data
	s.UpdatedAt = time.Now()
	return nil
}

// GetIdentityData returns identity data if session is identity type
func (s *VerificationSession) GetIdentityData() (*IdentityData, error) {
	if s.Type != VerificationIdentity {
		return nil, fmt.Errorf("session is not identity verification type")
	}
	if s.Data.Identity == nil {
		return nil, fmt.Errorf("identity data not set")
	}
	return s.Data.Identity, nil
}

// GetPhoneData returns phone data if session is phone type
func (s *VerificationSession) GetPhoneData() (*PhoneData, error) {
	if s.Type != VerificationPhone {
		return nil, fmt.Errorf("session is not phone verification type")
	}
	if s.Data.Phone == nil {
		return nil, fmt.Errorf("phone data not set")
	}
	return s.Data.Phone, nil
}

// GetAddressData returns address data if session is address type
func (s *VerificationSession) GetAddressData() (*AddressData, error) {
	if s.Type != VerificationAddress {
		return nil, fmt.Errorf("session is not address verification type")
	}
	if s.Data.Address == nil {
		return nil, fmt.Errorf("address data not set")
	}
	return s.Data.Address, nil
}

// GetBusinessData returns business data if session is business type
func (s *VerificationSession) GetBusinessData() (*BusinessData, error) {
	if s.Type != VerificationBusiness {
		return nil, fmt.Errorf("session is not business verification type")
	}
	if s.Data.Business == nil {
		return nil, fmt.Errorf("business data not set")
	}
	return s.Data.Business, nil
}

// GetListingData returns listing data if session is listing type
func (s *VerificationSession) GetListingData() (*ListingData, error) {
	if s.Type != VerificationListing {
		return nil, fmt.Errorf("session is not listing verification type")
	}
	if s.Data.Listing == nil {
		return nil, fmt.Errorf("listing data not set")
	}
	return s.Data.Listing, nil
}
