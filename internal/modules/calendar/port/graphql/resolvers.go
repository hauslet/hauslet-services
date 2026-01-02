package graphql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"hauslet/internal/modules/calendar/domain"
	"hauslet/internal/modules/calendar/service"
	"hauslet/internal/transport/graph/model"
	"hauslet/internal/transport/graph/viewer"

	"github.com/google/uuid"
)

// Resolver handles GraphQL queries and mutations for calendar events.
type Resolver struct {
	calendarService service.CalendarService
	log             *slog.Logger
}

// NewResolver creates a new GraphQL resolver for calendar.
func NewResolver(calendarService service.CalendarService, log *slog.Logger) *Resolver {
	return &Resolver{
		calendarService: calendarService,
		log:             log,
	}
}

// ============================================================================
// Query Resolvers
// ============================================================================

// CalendarEvent retrieves a single event by ID.
func (r *Resolver) CalendarEvent(ctx context.Context, id uuid.UUID) (*domain.CalendarEvent, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	event, err := r.calendarService.GetEvent(ctx, id, userID)
	if err != nil {
		r.log.Error("failed to get calendar event", "event_id", id, "error", err)
		return nil, err
	}

	return event, nil
}

// ListingEvents retrieves all events for a listing within a time range.
func (r *Resolver) ListingEvents(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, eventTypes []domain.EventType) ([]*domain.CalendarEvent, error) {
	events, err := r.calendarService.GetEventsForListing(ctx, listingID, startTime, endTime, eventTypes)
	if err != nil {
		r.log.Error("failed to get listing events", "listing_id", listingID, "error", err)
		return nil, err
	}

	return events, nil
}

// UpcomingListingEvents retrieves upcoming events for a listing.
func (r *Resolver) UpcomingListingEvents(ctx context.Context, listingID uuid.UUID, limit *int) ([]*domain.CalendarEvent, error) {
	defaultLimit := 10
	if limit != nil {
		defaultLimit = *limit
	}

	events, err := r.calendarService.GetUpcomingEvents(ctx, listingID, defaultLimit)
	if err != nil {
		r.log.Error("failed to get upcoming events", "listing_id", listingID, "error", err)
		return nil, err
	}

	return events, nil
}

// MyCalendarEvents retrieves events for all properties owned by the current user.
func (r *Resolver) MyCalendarEvents(ctx context.Context, startTime, endTime time.Time) ([]*domain.CalendarEvent, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	events, err := r.calendarService.GetEventsForOwner(ctx, userID, startTime, endTime)
	if err != nil {
		r.log.Error("failed to get owner events", "user_id", userID, "error", err)
		return nil, err
	}

	return events, nil
}

// OpenHouseAttendees retrieves attendees for an open house event.
func (r *Resolver) OpenHouseAttendees(ctx context.Context, eventID uuid.UUID) ([]*domain.Attendee, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	event, err := r.calendarService.GetEvent(ctx, eventID, userID)
	if err != nil {
		return nil, err
	}

	if event.EventType != domain.EventTypeOpenHouse {
		return nil, fmt.Errorf("event is not an open house")
	}

	if event.OpenHouseDetails == nil {
		return []*domain.Attendee{}, nil
	}

	attendees := make([]*domain.Attendee, 0, len(event.OpenHouseDetails.Attendees))
	for i := range event.OpenHouseDetails.Attendees {
		attendees = append(attendees, &event.OpenHouseDetails.Attendees[i])
	}

	return attendees, nil
}

// CheckListingAvailability checks if a listing is available for a time range.
func (r *Resolver) CheckListingAvailability(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) (bool, error) {
	result, err := r.calendarService.CheckAvailability(ctx, listingID, startTime, endTime)
	if err != nil {
		r.log.Error("failed to check availability", "listing_id", listingID, "error", err)
		return false, err
	}

	if result == nil {
		return false, nil
	}

	return result.Available, nil
}

// ============================================================================
// Mutation Resolvers
// ============================================================================

// RequestShowing creates a new showing request for a rent/sale listing.
func (r *Resolver) RequestShowing(ctx context.Context, input model.RequestShowingInput) (*domain.CalendarEvent, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch user profile data
	profile, err := r.calendarService.GetUserProfile(ctx, userID)
	if err != nil {
		r.log.Error("failed to get user profile", "user_id", userID, "error", err)
		return nil, fmt.Errorf("please complete your profile before requesting viewings: %w", err)
	}

	details := &domain.ShowingDetail{
		ProspectID:    &userID,
		ProspectName:  profile.FullName,
		ProspectEmail: profile.Email,
		ProspectPhone: profile.Phone,
		Notes:         input.Notes,
	}

	event, err := r.calendarService.RequestShowing(ctx, input.ListingID, input.StartTime, input.EndTime, details, userID)
	if err != nil {
		r.log.Error("failed to request showing", "listing_id", input.ListingID, "error", err)
		return nil, err
	}

	r.log.Info("showing requested", "event_id", event.ID, "listing_id", input.ListingID)
	return event, nil
}

// ConfirmShowing confirms a pending showing (owner only).
func (r *Resolver) ConfirmShowing(ctx context.Context, eventID uuid.UUID) (*domain.CalendarEvent, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.calendarService.ConfirmShowing(ctx, eventID, userID); err != nil {
		r.log.Error("failed to confirm showing", "event_id", eventID, "error", err)
		return nil, err
	}

	event, err := r.calendarService.GetEvent(ctx, eventID, userID)
	if err != nil {
		return nil, err
	}

	r.log.Info("showing confirmed", "event_id", eventID)
	return event, nil
}

// CancelShowing cancels a showing (owner or prospect).
func (r *Resolver) CancelShowing(ctx context.Context, input model.CancelShowingInput) (*domain.CalendarEvent, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	reason := ""
	if input.Reason != nil {
		reason = *input.Reason
	}

	if err := r.calendarService.CancelShowing(ctx, input.EventID, reason, userID); err != nil {
		r.log.Error("failed to cancel showing", "event_id", input.EventID, "error", err)
		return nil, err
	}

	event, err := r.calendarService.GetEvent(ctx, input.EventID, userID)
	if err != nil {
		return nil, err
	}

	r.log.Info("showing cancelled", "event_id", input.EventID)
	return event, nil
}

// RescheduleShowing reschedules a showing to a new time.
func (r *Resolver) RescheduleShowing(ctx context.Context, input model.RescheduleShowingInput) (*domain.CalendarEvent, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	reason := ""
	if input.Reason != nil {
		reason = *input.Reason
	}

	event, err := r.calendarService.RescheduleViewing(ctx, input.EventID, input.NewStartTime, input.NewEndTime, reason, userID)
	if err != nil {
		r.log.Error("failed to reschedule showing", "event_id", input.EventID, "error", err)
		return nil, err
	}

	r.log.Info("showing rescheduled", "event_id", input.EventID)
	return event, nil
}

// CreateOpenHouse creates a new open house event.
func (r *Resolver) CreateOpenHouse(ctx context.Context, input model.CreateOpenHouseInput) (*domain.CalendarEvent, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	details := &domain.OpenHouseDetail{
		Title:                input.Title,
		Description:          input.Description,
		MaxAttendees:         input.MaxAttendees,
		RegistrationDeadline: input.RegistrationDeadline,
		AgentID:              &userID,
	}

	event, err := r.calendarService.CreateOpenHouse(ctx, input.ListingID, input.StartTime, input.EndTime, details)
	if err != nil {
		r.log.Error("failed to create open house", "listing_id", input.ListingID, "error", err)
		return nil, err
	}

	r.log.Info("open house created", "event_id", event.ID, "listing_id", input.ListingID)
	return event, nil
}

// RegisterOpenHouse registers an attendee for an open house.
func (r *Resolver) RegisterOpenHouse(ctx context.Context, input model.RegisterOpenHouseInput) (bool, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	// Fetch user profile data
	profile, err := r.calendarService.GetUserProfile(ctx, userID)
	if err != nil {
		r.log.Error("failed to get user profile", "user_id", userID, "error", err)
		return false, fmt.Errorf("please complete your profile before registering: %w", err)
	}

	attendee := domain.Attendee{
		Name:   profile.FullName,
		Email:  profile.Email,
		Phone:  profile.Phone,
		UserID: &userID,
	}

	if err := r.calendarService.RegisterOpenHouseAttendee(ctx, input.EventID, attendee); err != nil {
		r.log.Error("failed to register open house attendee", "event_id", input.EventID, "error", err)
		return false, err
	}

	r.log.Info("attendee registered for open house", "event_id", input.EventID, "email", profile.Email)
	return true, nil
}

// RemoveOpenHouseAttendee removes an attendee from an open house.
func (r *Resolver) RemoveOpenHouseAttendee(ctx context.Context, eventID uuid.UUID, attendeeID uuid.UUID) (bool, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	if err := r.calendarService.RemoveOpenHouseAttendee(ctx, eventID, attendeeID, userID); err != nil {
		r.log.Error("failed to remove open house attendee", "event_id", eventID, "attendee_id", attendeeID, "error", err)
		return false, err
	}

	r.log.Info("attendee removed from open house", "event_id", eventID, "attendee_id", attendeeID)
	return true, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// getUserIDFromContext extracts the authenticated user ID from the context.
func getUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	v := viewer.FromContext(ctx)
	if v == nil || v.UserID == "" {
		return uuid.Nil, fmt.Errorf("unauthenticated")
	}

	userID, err := uuid.Parse(v.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID")
	}

	return userID, nil
}
