package hooks

import (
	"context"
	"fmt"
	bookingrepository "hauslet/internal/modules/booking/repository"
	bookingservice "hauslet/internal/modules/booking/service"
	businessservice "hauslet/internal/modules/business/service"
	propertyrepository "hauslet/internal/modules/property/repository"
	propertyschema "hauslet/internal/modules/property/repository/schema"
	"hauslet/internal/modules/review/notification"
	reviewservice "hauslet/internal/modules/review/service"
	"log/slog"

	"github.com/google/uuid"
)

// Ensure compile-time conformance with the booking review hooks.
var _ bookingservice.ReviewHooks = (*ReviewBookingHooksAdapter)(nil)

// ReviewBookingHooksAdapter sends review invites after booking completion.
type ReviewBookingHooksAdapter struct {
	bookingRepo      bookingrepository.BookingRepository
	listingRepo      propertyrepository.Repository
	businessSvc      businessservice.BusinessService
	userQuerier      reviewservice.UserQuerier
	notificationSvc  *notification.NotificationService
	reviewWindowDays int
	log              *slog.Logger
}

// NewReviewBookingHooksAdapter creates a new adapter for booking completion hooks.
func NewReviewBookingHooksAdapter(
	bookingRepo bookingrepository.BookingRepository,
	listingRepo propertyrepository.Repository,
	businessSvc businessservice.BusinessService,
	userQuerier reviewservice.UserQuerier,
	notificationSvc *notification.NotificationService,
	reviewWindowDays int,
	log *slog.Logger,
) *ReviewBookingHooksAdapter {
	return &ReviewBookingHooksAdapter{
		bookingRepo:      bookingRepo,
		listingRepo:      listingRepo,
		businessSvc:      businessSvc,
		userQuerier:      userQuerier,
		notificationSvc:  notificationSvc,
		reviewWindowDays: reviewWindowDays,
		log:              log,
	}
}

// SendReviewInvites notifies both guest and host that it's time to review.
func (a *ReviewBookingHooksAdapter) SendReviewInvites(ctx context.Context, bookingID uuid.UUID) error {
	if a.notificationSvc == nil {
		return nil
	}

	booking, err := a.bookingRepo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking %s: %w", bookingID, err)
	}
	if booking == nil {
		return fmt.Errorf("booking not found: %s", bookingID)
	}

	listing, err := a.listingRepo.GetListingByID(ctx, booking.ListingID, false)
	if err != nil {
		return fmt.Errorf("failed to get listing %s: %w", booking.ListingID, err)
	}
	if listing == nil {
		return fmt.Errorf("listing not found: %s", booking.ListingID)
	}

	days := a.reviewWindowDays
	if days <= 0 {
		days = 14
	}

	listingTitle := listing.Title
	if listingTitle == "" {
		listingTitle = "your stay"
	}

	guestContact := notification.ContactInfo{
		ID:    booking.GuestID,
		Name:  booking.GuestName,
		Email: booking.GuestEmail,
	}

	guestSent := false
	if guestContact.Email != "" {
		a.notificationSvc.SendReviewInviteToGuest(ctx, booking.ID, guestContact, listingTitle, days)
		guestSent = true
	} else if a.log != nil {
		a.log.Warn("guest email missing for booking %s; review invite skipped", booking.ID)
	}

	hostContact, err := a.resolveHostContact(ctx, listing)
	if err != nil && a.log != nil {
		a.log.Warn("failed to resolve host contact for booking %s: %v", booking.ID, err)
	}

	hostSent := false
	if hostContact.Email != "" {
		a.notificationSvc.SendReviewInviteToHost(ctx, booking.ID, hostContact, booking.GuestName, days)
		hostSent = true
	} else if a.log != nil {
		a.log.Warn("host email missing for booking %s; review invite skipped", booking.ID)
	}

	if !guestSent && !hostSent {
		return fmt.Errorf("no review invites sent for booking %s", booking.ID)
	}

	return nil
}

func (a *ReviewBookingHooksAdapter) resolveHostContact(ctx context.Context, listing *propertyschema.Listing) (notification.ContactInfo, error) {
	if listing == nil {
		return notification.ContactInfo{}, fmt.Errorf("listing is nil")
	}

	if listing.OwnerType != propertyschema.OwnerBusiness {
		if a.userQuerier == nil {
			return notification.ContactInfo{}, fmt.Errorf("user contact resolver not configured")
		}
		name, email, err := a.userQuerier.GetUserContact(ctx, listing.OwnerID)
		if err != nil {
			return notification.ContactInfo{}, err
		}
		return notification.ContactInfo{ID: listing.OwnerID, Name: name, Email: email}, nil
	}

	if a.businessSvc == nil {
		return notification.ContactInfo{}, fmt.Errorf("business service not configured")
	}

	members, err := a.businessSvc.GetBusinessMembers(ctx, listing.OwnerID)
	if err != nil {
		return notification.ContactInfo{}, err
	}

	if a.userQuerier != nil {
		for _, m := range members {
			if !m.IsActive {
				continue
			}
			if !(m.IsOwner() || m.IsAdmin() || m.Permissions.CanPublishListings) {
				continue
			}
			name, email, err := a.userQuerier.GetUserContact(ctx, m.UserID)
			if err != nil || email == "" {
				continue
			}
			return notification.ContactInfo{ID: m.UserID, Name: name, Email: email}, nil
		}
	}

	business, err := a.businessSvc.GetBusiness(ctx, listing.OwnerID)
	if err != nil {
		return notification.ContactInfo{}, err
	}
	if business == nil {
		return notification.ContactInfo{}, nil
	}

	email := business.Email
	if business.BillingEmail != nil && *business.BillingEmail != "" {
		email = *business.BillingEmail
	}

	name := business.DisplayName
	if name == "" {
		name = business.Name
	}

	return notification.ContactInfo{
		ID:    listing.OwnerID,
		Name:  name,
		Email: email,
	}, nil
}
