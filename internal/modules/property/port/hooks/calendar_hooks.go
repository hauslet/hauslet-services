package hooks

import (
	"context"
	"fmt"

	calendarservice "hauslet/internal/modules/calendar/service"
	"hauslet/internal/modules/property/domain"
	propertyrepository "hauslet/internal/modules/property/repository"
	listingService "hauslet/internal/modules/property/service"

	"github.com/google/uuid"
)

const (
	defaultListingTimezone = "Africa/Lagos"
	defaultListingCurrency = "NGN"
)

// listingHookAdapter bundles the property service so multiple adapters can share helpers.
type listingHookAdapter struct {
	svc listingService.Service
}

func newListingHookAdapter(svc listingService.Service) *listingHookAdapter {
	return &listingHookAdapter{svc: svc}
}

type listingRepoHookAdapter struct {
	repo propertyrepository.Repository
}

func newListingRepoHookAdapter(repo propertyrepository.Repository) *listingRepoHookAdapter {
	return &listingRepoHookAdapter{repo: repo}
}

// getListing loads a listing without media for hook operations.
func (a *listingHookAdapter) getListing(ctx context.Context, listingID uuid.UUID) (*domain.Listing, error) {
	listing, err := a.svc.GetListingByID(ctx, listingID, false)
	if err != nil {
		return nil, err
	}
	if listing == nil {
		return nil, domain.ErrListingNotFound
	}
	return listing, nil
}

func (a *listingRepoHookAdapter) getListing(ctx context.Context, listingID uuid.UUID) (*domain.Listing, error) {
	if a.repo == nil {
		return nil, fmt.Errorf("property repository not configured")
	}

	listing, err := a.repo.GetListingByID(ctx, listingID, false)
	if err != nil {
		return nil, err
	}
	if listing == nil {
		return nil, domain.ErrListingNotFound
	}

	mapped := domain.MapListingFromSchema(listing)
	if mapped == nil {
		return nil, domain.ErrListingNotFound
	}

	return mapped, nil
}

// GetListingOwner is shared by both calendar and pricing hook implementations.
func (a *listingHookAdapter) GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error) {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return uuid.Nil, err
	}
	return listing.OwnerID, nil
}

// CalendarHooksAdapter implements calendar service ListingHooks using the property service.
type CalendarHooksAdapter struct {
	*listingHookAdapter
	timezone string
}

// NewCalendarHooksAdapter wires the property service into the calendar hooks contract.
func NewCalendarHooksAdapter(svc listingService.Service) *CalendarHooksAdapter {
	return &CalendarHooksAdapter{
		listingHookAdapter: newListingHookAdapter(svc),
		timezone:           defaultListingTimezone,
	}
}

func (a *CalendarHooksAdapter) CanUseCalendar(ctx context.Context, listingID uuid.UUID) (bool, error) {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return false, err
	}

	// Calendar can only be used when explicitly enabled on the listing.
	return listing.HasCalendar, nil
}

func (a *CalendarHooksAdapter) GetListingConstraints(ctx context.Context, listingID uuid.UUID) (*calendarservice.ListingConstraints, error) {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return nil, err
	}

	if listing.ShortletDetails == nil {
		return nil, fmt.Errorf("listing %s does not have shortlet details for booking constraints", listingID)
	}

	detail := listing.ShortletDetails
	constraints := &calendarservice.ListingConstraints{
		ListingID:    listing.ID,
		MinNights:    detail.MinNights,
		MaxNights:    detail.MaxNights,
		MaxGuests:    detail.MaxGuests,
		CheckInTime:  detail.CheckInTime,
		CheckOutTime: detail.CheckOutTime,
		Currency:     currencyOrDefault(listing.Currency),
		Timezone:     a.timezone,
	}

	return constraints, nil
}

func (a *CalendarHooksAdapter) MarkCalendarEnabled(ctx context.Context, listingID uuid.UUID, enabled bool) error {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return err
	}

	if listing.HasCalendar == enabled {
		return nil
	}

	_, err = a.svc.PatchListing(ctx, listingID, map[string]any{
		"has_calendar": enabled,
	})
	return err
}

// CalendarHooksRepoAdapter implements calendar hooks using the property repository.
type CalendarHooksRepoAdapter struct {
	*listingRepoHookAdapter
	timezone string
}

func NewCalendarHooksRepoAdapter(repo propertyrepository.Repository) *CalendarHooksRepoAdapter {
	return &CalendarHooksRepoAdapter{
		listingRepoHookAdapter: newListingRepoHookAdapter(repo),
		timezone:               defaultListingTimezone,
	}
}

func (a *CalendarHooksRepoAdapter) CanUseCalendar(ctx context.Context, listingID uuid.UUID) (bool, error) {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return false, err
	}

	return listing.HasCalendar, nil
}

func (a *CalendarHooksRepoAdapter) GetListingOwner(ctx context.Context, listingID uuid.UUID) (uuid.UUID, error) {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return uuid.Nil, err
	}
	return listing.OwnerID, nil
}

func (a *CalendarHooksRepoAdapter) GetListingConstraints(ctx context.Context, listingID uuid.UUID) (*calendarservice.ListingConstraints, error) {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return nil, err
	}

	if listing.ShortletDetails == nil {
		return nil, fmt.Errorf("listing %s does not have shortlet details for booking constraints", listingID)
	}

	detail := listing.ShortletDetails
	constraints := &calendarservice.ListingConstraints{
		ListingID:    listing.ID,
		MinNights:    detail.MinNights,
		MaxNights:    detail.MaxNights,
		MaxGuests:    detail.MaxGuests,
		CheckInTime:  detail.CheckInTime,
		CheckOutTime: detail.CheckOutTime,
		Currency:     currencyOrDefault(listing.Currency),
		Timezone:     a.timezone,
	}

	return constraints, nil
}

func (a *CalendarHooksRepoAdapter) MarkCalendarEnabled(ctx context.Context, listingID uuid.UUID, enabled bool) error {
	listing, err := a.getListing(ctx, listingID)
	if err != nil {
		return err
	}

	if listing.HasCalendar == enabled {
		return nil
	}

	return a.repo.PatchListing(ctx, listingID, map[string]any{
		"has_calendar": enabled,
	})
}
