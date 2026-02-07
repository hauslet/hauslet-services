package service

import (
	"context"
	"testing"
	"time"

	"hauslet/internal/modules/booking/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type stubProfileProvider struct {
	contact *ContactInfo
}

func (s *stubProfileProvider) GetUserContact(_ context.Context, _ uuid.UUID) (*ContactInfo, error) {
	return s.contact, nil
}

func TestQuoteBooking_ReturnsUnavailableForSuspendedListing(t *testing.T) {
	ctx := context.Background()
	listingID := uuid.New()
	checkIn := time.Now().Add(48 * time.Hour)
	checkOut := checkIn.Add(24 * time.Hour)

	mockListing := new(MockListingHooks)
	mockListing.On("GetListingConstraints", ctx, listingID).Return(nil, domain.ErrListingSuspended)

	svc := &BookingServiceImpl{
		listingHooks: mockListing,
	}

	quote, err := svc.QuoteBooking(ctx, listingID, checkIn, checkOut, 2)
	assert.NoError(t, err)
	if assert.NotNil(t, quote) {
		assert.False(t, quote.Available)
		assert.NotNil(t, quote.UnavailabilityReason)
		assert.Equal(t, domain.ErrListingSuspended.Error(), *quote.UnavailabilityReason)
	}

	mockListing.AssertExpectations(t)
}

func TestRequestBooking_BlocksSuspendedListing(t *testing.T) {
	ctx := context.Background()
	listingID := uuid.New()
	guestID := uuid.New()
	checkIn := time.Now().Add(72 * time.Hour)
	checkOut := checkIn.Add(24 * time.Hour)

	mockListing := new(MockListingHooks)
	mockListing.On("GetListingConstraints", ctx, listingID).Return(nil, domain.ErrListingSuspended)

	svc := &BookingServiceImpl{
		listingHooks: mockListing,
	}

	booking, err := svc.RequestBooking(ctx, listingID, guestID, checkIn, checkOut, 2, nil)
	assert.Nil(t, booking)
	assert.ErrorIs(t, err, domain.ErrListingSuspended)

	mockListing.AssertExpectations(t)
}

func TestReserveBooking_BlocksUnavailableListing(t *testing.T) {
	ctx := context.Background()
	listingID := uuid.New()
	guestID := uuid.New()
	checkIn := time.Now().Add(72 * time.Hour)
	checkOut := checkIn.Add(24 * time.Hour)

	mockListing := new(MockListingHooks)
	mockListing.On("GetListingConstraints", ctx, listingID).Return(nil, domain.ErrListingUnavailable)

	svc := &BookingServiceImpl{
		listingHooks: mockListing,
		profiles: &stubProfileProvider{
			contact: &ContactInfo{
				ID:    guestID,
				Name:  "Guest",
				Email: "guest@example.com",
			},
		},
	}

	booking, payment, err := svc.ReserveBooking(ctx, listingID, guestID, checkIn, checkOut, 2, nil, nil)
	assert.Nil(t, booking)
	assert.Nil(t, payment)
	assert.ErrorIs(t, err, domain.ErrListingUnavailable)

	mockListing.AssertExpectations(t)
}
