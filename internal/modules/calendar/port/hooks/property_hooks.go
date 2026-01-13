package hooks

import (
	"context"
	"fmt"
	"hauslet/internal/modules/calendar/service"
	"hauslet/internal/modules/property/repository"

	"github.com/google/uuid"
)

// PropertyHooksAdapter adapts property repository for calendar's ListingHooks interface.
type PropertyHooksAdapter struct {
	propertyRepo repository.Repository
}

// NewPropertyHooksAdapter creates a new property hooks adapter for calendar.
func NewPropertyHooksAdapter(propertyRepo repository.Repository) service.ListingHooks {
	return &PropertyHooksAdapter{
		propertyRepo: propertyRepo,
	}
}

// CanUseCalendar checks if a listing has calendar enabled.
func (a *PropertyHooksAdapter) CanUseCalendar(ctx context.Context, listingID uuid.UUID) (bool, error) {
	listing, err := a.propertyRepo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return false, err
	}
	if listing == nil {
		return false, fmt.Errorf("listing not found")
	}
	return listing.HasCalendar, nil
}

// GetListingOwner retrieves the owner ID of a listing.
func (a *PropertyHooksAdapter) GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error) {
	listing, err := a.propertyRepo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return uuid.Nil, err
	}
	if listing == nil {
		return uuid.Nil, fmt.Errorf("listing not found")
	}
	return listing.OwnerID, nil
}

// GetListingConstraints retrieves calendar constraints for a listing.
func (a *PropertyHooksAdapter) GetListingConstraints(ctx context.Context, listingID uuid.UUID) (*service.ListingConstraints, error) {
	listing, err := a.propertyRepo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return nil, err
	}
	if listing == nil {
		return nil, fmt.Errorf("listing not found")
	}

	constraints := &service.ListingConstraints{
		ListingID: listing.ID,
		Currency:  string(listing.Currency),
		Timezone:  "Africa/Lagos", // Default timezone
	}

	// Extract constraints based on listing type
	if listing.ShortletDetails != nil {
		detail := listing.ShortletDetails

		// Basic constraints
		constraints.MinNights = detail.StayLimits.MinNights
		constraints.MaxNights = detail.StayLimits.MaxNights
		constraints.MaxGuests = detail.MaxGuests
		constraints.CheckInTime = detail.CheckInTime
		constraints.CheckOutTime = detail.CheckOutTime

		// Convert custom fees
		if len(detail.Fees) > 0 {
			constraints.Fees = make([]service.CustomFee, len(detail.Fees))
			for i, fee := range detail.Fees {
				constraints.Fees[i] = service.CustomFee{
					Name:         fee.Name,
					Amount:       fee.Amount,
					Frequency:    string(fee.Frequency),
					Category:     string(fee.Category),
					IsRefundable: fee.IsRefundable,
					IsOptional:   fee.IsOptional,
				}
			}
		}

		// Convert discounts
		if len(detail.Discounts) > 0 {
			constraints.Discounts = make([]service.Discount, len(detail.Discounts))
			for i, discount := range detail.Discounts {
				constraints.Discounts[i] = service.Discount{
					Name:       discount.Name,
					Type:       string(discount.Type),
					Percentage: discount.Percentage,
					MinNights:  discount.MinNights,
					Active:     discount.Active,
				}
			}
		}

		// Convert booking settings
		constraints.BookingSettings = &service.BookingSettings{
			ApprovalMethod:       string(detail.BookingSettings.ApprovalMethod),
			VerifiedID:           detail.BookingSettings.GuestRequirements.VerifiedID,
			PositiveReviewsOnly:  detail.BookingSettings.GuestRequirements.PositiveReviewsOnly,
			ProfilePhotoRequired: detail.BookingSettings.GuestRequirements.ProfilePhotoRequired,
			PreBookingMessage:    detail.BookingSettings.PreBookingMessage,
		}

		// Convert advance booking settings
		constraints.AdvanceBooking = &service.AdvanceBooking{
			MonthsAhead:    detail.AdvanceBooking.MonthsAhead,
			MinNoticeHours: detail.AdvanceBooking.MinNoticeHours,
		}
	}

	return constraints, nil
}

// MarkCalendarEnabled updates the has_calendar flag in the listing.
func (a *PropertyHooksAdapter) MarkCalendarEnabled(ctx context.Context, listingID uuid.UUID, enabled bool) error {
	return a.propertyRepo.PatchListing(ctx, listingID, map[string]interface{}{
		"has_calendar": enabled,
	})
}

// GetShowingAvailability returns showing availability windows for rent/sale listings.
func (a *PropertyHooksAdapter) GetShowingAvailability(ctx context.Context, listingID uuid.UUID) ([]service.ShowingAvailability, error) {
	listing, err := a.propertyRepo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return nil, err
	}
	if listing == nil {
		return nil, fmt.Errorf("listing not found")
	}

	var availability []service.ShowingAvailability

	// Check rental details
	if listing.RentalDetails != nil && listing.RentalDetails.ShowingAvailability != nil {
		for _, slot := range *listing.RentalDetails.ShowingAvailability {
			availability = append(availability, service.ShowingAvailability{
				DayOfWeek: slot.DayOfWeek,
				StartTime: slot.StartTime,
				EndTime:   slot.EndTime,
				Timezone:  slot.Timezone,
			})
		}
	}

	// Check sale details
	if listing.SaleDetails != nil && listing.SaleDetails.ShowingAvailability != nil {
		for _, slot := range *listing.SaleDetails.ShowingAvailability {
			availability = append(availability, service.ShowingAvailability{
				DayOfWeek: slot.DayOfWeek,
				StartTime: slot.StartTime,
				EndTime:   slot.EndTime,
				Timezone:  slot.Timezone,
			})
		}
	}

	return availability, nil
}

// GetListingType returns the listing type (sale, rent, shortlet).
func (a *PropertyHooksAdapter) GetListingType(ctx context.Context, listingID uuid.UUID) (string, error) {
	listing, err := a.propertyRepo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return "", err
	}
	if listing == nil {
		return "", fmt.Errorf("listing not found")
	}
	return string(listing.ListingType), nil
}
