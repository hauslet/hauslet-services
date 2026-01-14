package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hauslet/internal/modules/calendar/domain"
	"hauslet/internal/modules/calendar/repository/schema"

	"github.com/google/uuid"
)

// --- Viewing Events Implementation ---

// RequestShowing creates a new showing request for rent/sale listings.
func (s *CalendarServiceImpl) RequestShowing(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, details *domain.ShowingDetail, requestorID uuid.UUID) (*domain.CalendarEvent, error) {
	if s.log != nil {
		s.log.Info("requesting showing", "listing", listingID, "requestor", requestorID)
	}

	// Validate listing type (must be rent or sale)
	listingType, err := s.listingHooks.GetListingType(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get listing type: %w", err)
	}
	if listingType != "rent" && listingType != "sale" {
		return nil, fmt.Errorf("showings are only available for rent and sale listings")
	}

	// Validate calendar is enabled
	canUse, err := s.listingHooks.CanUseCalendar(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to check calendar eligibility: %w", err)
	}
	if !canUse {
		return nil, domain.ErrUnauthorized
	}

	// Prevent owners from requesting showings for their own listings
	ownerID, err := s.listingHooks.GetListingOwner(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get listing owner: %w", err)
	}
	if ownerID == requestorID {
		if s.log != nil {
			s.log.Warn("owner attempted to request showing for own listing", "listing_id", listingID, "user_id", requestorID)
		}
		return nil, domain.ErrCannotRequestOwnShowing
	}

	// Validate against showing availability windows
	availability, err := s.listingHooks.GetShowingAvailability(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get showing availability: %w", err)
	}

	if len(availability) > 0 {
		if err := validateShowingTime(startTime, endTime, availability); err != nil {
			return nil, err
		}
	}

	// Check for conflicts
	conflicts, err := s.repo.GetConflictingEvents(ctx, listingID, startTime, endTime, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, domain.ErrEventConflict
	}

	// Set request tracking fields
	now := time.Now()
	details.RequestedBy = &requestorID
	details.RequestedAt = &now
	details.RescheduleCount = 0

	// Create event
	event := &domain.CalendarEvent{
		ListingID:      listingID,
		EventType:      domain.EventTypeShowing,
		Status:         domain.EventStatusPending,
		StartTime:      startTime,
		EndTime:        endTime,
		ShowingDetails: details,
		CreatedBy:      &requestorID,
	}

	createdEvent, err := s.CreateEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.SendShowingRequest(ctx, createdEvent)
	}

	return createdEvent, nil
}

// ConfirmShowing confirms a pending showing request.
func (s *CalendarServiceImpl) ConfirmShowing(ctx context.Context, eventID uuid.UUID, confirmerID uuid.UUID) error {
	if s.log != nil {
		s.log.Info("confirming showing", "event_id", eventID, "confirmer", confirmerID)
	}

	// Get event
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Validate event type
	if event.EventType != schema.EventTypeShowing {
		return fmt.Errorf("event is not a showing")
	}

	// Verify access (must be listing owner)
	ownerID, err := s.listingHooks.GetListingOwner(ctx, event.ListingID)
	if err != nil {
		return err
	}
	if ownerID != confirmerID {
		return domain.ErrAccessDenied
	}

	// Update status
	event.Status = schema.EventStatusConfirmed
	event.UpdatedBy = &confirmerID

	if err := s.repo.UpdateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to confirm showing: %w", err)
	}

	// Invalidate caches
	s.invalidateEventCache(ctx, eventID)
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	if s.notifier != nil {
		s.notifier.SendShowingConfirmed(ctx, domain.MapEventFromSchemaToEntity(event))
	}

	if s.log != nil {
		s.log.Info("showing confirmed", "event_id", eventID)
	}

	return nil
}

// CancelShowing cancels a showing (pending or confirmed).
func (s *CalendarServiceImpl) CancelShowing(ctx context.Context, eventID uuid.UUID, cancelReason string, cancellerID uuid.UUID) error {
	if s.log != nil {
		s.log.Info("cancelling showing", "event_id", eventID, "canceller", cancellerID)
	}

	// Get event
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Validate event type
	if event.EventType != schema.EventTypeShowing {
		return fmt.Errorf("event is not a showing")
	}

	// Verify access (owner or prospect can cancel)
	ownerID, err := s.listingHooks.GetListingOwner(ctx, event.ListingID)
	if err != nil {
		return err
	}

	canCancel := ownerID == cancellerID
	if event.ShowingDetails != nil && event.ShowingDetails.ProspectID != nil {
		canCancel = canCancel || *event.ShowingDetails.ProspectID == cancellerID
	}
	if event.ShowingDetails != nil && event.ShowingDetails.RequestedBy != nil {
		canCancel = canCancel || *event.ShowingDetails.RequestedBy == cancellerID
	}

	if !canCancel {
		return domain.ErrAccessDenied
	}

	// Update showing details with cancel reason
	if event.ShowingDetails != nil {
		event.ShowingDetails.CancelReason = &cancelReason
	}

	// Update status
	event.Status = schema.EventStatusCancelled
	event.UpdatedBy = &cancellerID

	if err := s.repo.UpdateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to cancel showing: %w", err)
	}

	// Invalidate caches
	s.invalidateEventCache(ctx, eventID)
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	if s.log != nil {
		s.log.Info("showing cancelled", "event_id", eventID)
	}

	return nil
}

// RescheduleViewing reschedules a showing to a new time.
func (s *CalendarServiceImpl) RescheduleViewing(ctx context.Context, eventID uuid.UUID, newStartTime, newEndTime time.Time, reason string, requestorID uuid.UUID) (*domain.CalendarEvent, error) {
	if s.log != nil {
		s.log.Info("rescheduling viewing", "event_id", eventID, "requestor", requestorID)
	}

	// Get event
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Validate event type
	if event.EventType != schema.EventTypeShowing {
		return nil, fmt.Errorf("event is not a showing")
	}

	// Verify access (owner or prospect can reschedule)
	ownerID, err := s.listingHooks.GetListingOwner(ctx, event.ListingID)
	if err != nil {
		return nil, err
	}

	canReschedule := ownerID == requestorID
	if event.ShowingDetails != nil && event.ShowingDetails.ProspectID != nil {
		canReschedule = canReschedule || *event.ShowingDetails.ProspectID == requestorID
	}
	if event.ShowingDetails != nil && event.ShowingDetails.RequestedBy != nil {
		canReschedule = canReschedule || *event.ShowingDetails.RequestedBy == requestorID
	}

	if !canReschedule {
		return nil, domain.ErrAccessDenied
	}

	// Validate against showing availability windows
	availability, err := s.listingHooks.GetShowingAvailability(ctx, event.ListingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get showing availability: %w", err)
	}

	if len(availability) > 0 {
		if err := validateShowingTime(newStartTime, newEndTime, availability); err != nil {
			return nil, err
		}
	}

	// Check for conflicts (excluding current event)
	conflicts, err := s.repo.GetConflictingEvents(ctx, event.ListingID, newStartTime, newEndTime, &eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, domain.ErrEventConflict
	}

	// Update showing details
	now := time.Now()
	if event.ShowingDetails != nil {
		event.ShowingDetails.RescheduledAt = &now
		event.ShowingDetails.RescheduleCount++
		event.ShowingDetails.RescheduleReason = &reason
	}

	// Update event
	event.StartTime = newStartTime
	event.EndTime = newEndTime
	event.Status = schema.EventStatusPending // Reset to pending after reschedule
	event.UpdatedBy = &requestorID

	if err := s.repo.UpdateEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to reschedule viewing: %w", err)
	}

	// Get updated event
	updatedEvent, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated event: %w", err)
	}

	// Invalidate caches
	s.invalidateEventCache(ctx, eventID)
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	if s.log != nil {
		s.log.Info("viewing rescheduled", "event_id", eventID)
	}

	return domain.MapEventFromSchemaToEntity(updatedEvent), nil
}

// RegisterOpenHouseAttendee registers an attendee for an open house (idempotent).
func (s *CalendarServiceImpl) RegisterOpenHouseAttendee(ctx context.Context, eventID uuid.UUID, attendee domain.Attendee) error {
	if s.log != nil {
		s.log.Info("registering open house attendee", "event_id", eventID, "email", attendee.Email)
	}

	// Get event
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Validate event type
	if event.EventType != schema.EventTypeOpenHouse {
		return fmt.Errorf("event is not an open house")
	}

	// Prevent owners from registering for their own open house events
	if attendee.UserID != nil {
		ownerID, err := s.listingHooks.GetListingOwner(ctx, event.ListingID)
		if err != nil {
			return fmt.Errorf("failed to get listing owner: %w", err)
		}
		if ownerID == *attendee.UserID {
			if s.log != nil {
				s.log.Warn("owner attempted to register for own open house", "event_id", eventID, "user_id", *attendee.UserID)
			}
			return domain.ErrCannotRegisterOwnOpenHouse
		}
	}

	// Check if registration deadline has passed
	if event.OpenHouseDetails != nil && event.OpenHouseDetails.RegistrationDeadline != nil {
		if time.Now().After(*event.OpenHouseDetails.RegistrationDeadline) {
			return fmt.Errorf("registration deadline has passed")
		}
	}

	// Check capacity using atomic count
	if event.OpenHouseDetails != nil && event.OpenHouseDetails.MaxAttendees > 0 {
		currentCount, err := s.repo.GetOpenHouseAttendeeCount(ctx, eventID)
		if err != nil {
			return fmt.Errorf("failed to check attendee count: %w", err)
		}
		if currentCount >= event.OpenHouseDetails.MaxAttendees {
			return fmt.Errorf("open house is at capacity")
		}
	}

	// Check for duplicate registration (by email) - still need to fetch for this check
	if event.OpenHouseDetails != nil {
		for _, existing := range event.OpenHouseDetails.Attendees {
			if strings.EqualFold(existing.Email, attendee.Email) {
				// Already registered, return success (idempotent)
				return nil
			}
		}
	}

	// Prepare attendee record
	schemaAttendee := schema.Attendee{
		ID:                uuid.New(),
		Name:              attendee.Name,
		Email:             attendee.Email,
		Phone:             attendee.Phone,
		UserID:            attendee.UserID,
		RegisteredAt:      time.Now(),
		RegistrationToken: attendee.RegistrationToken,
	}

	// Atomically add attendee using JSONB operations
	if err := s.repo.AddOpenHouseAttendee(ctx, eventID, schemaAttendee); err != nil {
		return fmt.Errorf("failed to register attendee: %w", err)
	}

	// Invalidate cache
	s.invalidateEventCache(ctx, eventID)

	if s.notifier != nil {
		s.notifier.SendOpenHouseRegistration(ctx, domain.MapEventFromSchemaToEntity(event), attendee)
	}

	if s.log != nil {
		s.log.Info("attendee registered", "event_id", eventID, "attendee_id", schemaAttendee.ID)
	}

	return nil
}

// RemoveOpenHouseAttendee removes an attendee from an open house.
func (s *CalendarServiceImpl) RemoveOpenHouseAttendee(ctx context.Context, eventID uuid.UUID, attendeeID uuid.UUID, removerID uuid.UUID) error {
	if s.log != nil {
		s.log.Info("removing open house attendee", "event_id", eventID, "attendee_id", attendeeID)
	}

	// Get event
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	// Validate event type
	if event.EventType != schema.EventTypeOpenHouse {
		return fmt.Errorf("event is not an open house")
	}

	// Verify access (owner can remove anyone, attendee can remove themselves)
	ownerID, err := s.listingHooks.GetListingOwner(ctx, event.ListingID)
	if err != nil {
		return err
	}

	canRemove := ownerID == removerID
	var attendeeEmail string

	// Check if remover is the attendee themselves and find the email
	if event.OpenHouseDetails != nil {
		for _, att := range event.OpenHouseDetails.Attendees {
			if att.ID == attendeeID {
				attendeeEmail = att.Email
				if att.UserID != nil && *att.UserID == removerID {
					canRemove = true
				}
				break
			}
		}
	}

	if !canRemove {
		return domain.ErrAccessDenied
	}

	if attendeeEmail == "" {
		return fmt.Errorf("attendee not found")
	}

	// Atomically remove attendee using JSONB operations
	if err := s.repo.RemoveOpenHouseAttendee(ctx, eventID, attendeeEmail); err != nil {
		return fmt.Errorf("failed to remove attendee: %w", err)
	}

	// Invalidate cache
	s.invalidateEventCache(ctx, eventID)

	if s.log != nil {
		s.log.Info("attendee removed", "event_id", eventID, "attendee_id", attendeeID)
	}

	return nil
}

// --- Helper Functions ---

// validateShowingTime checks if the requested time falls within availability windows.
func validateShowingTime(startTime, endTime time.Time, availability []ShowingAvailability) error {
	if len(availability) == 0 {
		return nil // No restrictions
	}

	// Get day of week from start time
	dayOfWeek := strings.ToLower(startTime.Weekday().String())

	// Find matching availability window for this day
	var matchingWindow *ShowingAvailability
	for _, slot := range availability {
		if strings.ToLower(slot.DayOfWeek) == dayOfWeek {
			matchingWindow = &slot
			break
		}
	}

	if matchingWindow == nil {
		return fmt.Errorf("no showing availability on %s", startTime.Weekday())
	}

	// Parse window times (HH:MM format)
	windowStart, err := parseTimeOfDay(matchingWindow.StartTime)
	if err != nil {
		return fmt.Errorf("invalid availability start time: %w", err)
	}
	windowEnd, err := parseTimeOfDay(matchingWindow.EndTime)
	if err != nil {
		return fmt.Errorf("invalid availability end time: %w", err)
	}

	// Extract time of day from request
	requestStart := startTime.Hour()*60 + startTime.Minute()
	requestEnd := endTime.Hour()*60 + endTime.Minute()

	// Validate within window
	if requestStart < windowStart || requestEnd > windowEnd {
		return fmt.Errorf("showing time outside availability window (%s - %s)", matchingWindow.StartTime, matchingWindow.EndTime)
	}

	return nil
}

// parseTimeOfDay converts "HH:MM" to minutes since midnight.
func parseTimeOfDay(timeStr string) (int, error) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}
