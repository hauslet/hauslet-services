package hooks

import (
	"context"
	"fmt"

	calendarnotification "hauslet/internal/modules/calendar/notification"
	calendarservice "hauslet/internal/modules/calendar/service"
	"hauslet/internal/modules/profile/service"

	"github.com/google/uuid"
)

// CalendarProfileAdapter exposes minimal profile data to calendar notifications.
type CalendarProfileAdapter struct {
	svc service.ProfileService
}

// NewCalendarProfileAdapter wires the profile service into calendar notifications.
func NewCalendarProfileAdapter(svc service.ProfileService) *CalendarProfileAdapter {
	return &CalendarProfileAdapter{svc: svc}
}

// GetUserContact returns the user's display name, primary email, and first phone number if available.
func (a *CalendarProfileAdapter) GetUserContact(ctx context.Context, userID uuid.UUID) (*calendarnotification.ContactInfo, error) {
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
	if email == "" {
		return nil, nil
	}

	name := profile.FullName
	var phone *string
	if len(profile.PhoneNumbers) > 0 {
		primary := profile.PhoneNumbers[0]
		phone = &primary
	}

	return &calendarnotification.ContactInfo{
		ID:    userID,
		Name:  name,
		Email: email,
		Phone: phone,
	}, nil
}

// GetUserProfile returns profile data needed for viewing appointments.
func (a *CalendarProfileAdapter) GetUserProfile(ctx context.Context, userID uuid.UUID) (*calendarservice.UserProfile, error) {
	profile, err := a.svc.GetProfileByUserID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	if profile == nil {
		return nil, fmt.Errorf("profile not found for user %s", userID)
	}

	// Validate required fields
	if profile.FullName == "" {
		return nil, fmt.Errorf("user profile incomplete: full name required")
	}
	if profile.Email == nil || *profile.Email == "" {
		return nil, fmt.Errorf("user profile incomplete: email required")
	}

	var phone *string
	if len(profile.PhoneNumbers) > 0 {
		phone = &profile.PhoneNumbers[0]
	}

	return &calendarservice.UserProfile{
		UserID:       userID,
		FullName:     profile.FullName,
		Email:        *profile.Email,
		Phone:        phone,
		IsIDVerified: profile.IDVerified,
	}, nil
}
