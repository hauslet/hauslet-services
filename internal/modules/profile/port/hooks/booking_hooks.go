package hooks

import (
	"context"

	bookingservice "hauslet/internal/modules/booking/service"
	"hauslet/internal/modules/profile/service"

	"github.com/google/uuid"
)

// BookingProfileAdapter exposes minimal profile data to the booking module.
type BookingProfileAdapter struct {
	svc service.ProfileService
}

// NewBookingProfileAdapter wires the profile service into booking.
func NewBookingProfileAdapter(svc service.ProfileService) *BookingProfileAdapter {
	return &BookingProfileAdapter{svc: svc}
}

// GetUserContact returns the user's display name, primary email, and first phone number if available.
func (a *BookingProfileAdapter) GetUserContact(ctx context.Context, userID uuid.UUID) (*bookingservice.ContactInfo, error) {
	profile, err := a.svc.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, nil
	}

	var email string
	if profile.Email != nil {
		email = *profile.Email
	}
	if email == "" || profile.FullName == "" {
		return nil, nil
	}

	var phone *string
	if len(profile.PhoneNumbers) > 0 {
		primary := profile.PhoneNumbers[0]
		phone = &primary
	}

	return &bookingservice.ContactInfo{
		ID:    userID,
		Name:  profile.FullName,
		Email: email,
		Phone: phone,
	}, nil
}
