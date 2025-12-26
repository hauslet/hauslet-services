package service

import (
	"context"
	"fmt"
	"time"

	"hauslet/internal/modules/calendar/domain"
	"hauslet/internal/modules/calendar/repository/schema"

	"github.com/google/uuid"
)

// --- Event Lifecycle ---

func (s *CalendarServiceImpl) CreateEvent(ctx context.Context, event *domain.CalendarEvent) (*domain.CalendarEvent, error) {
	if s.log != nil {
		s.log.Logf("INFO creating calendar event type=%s listing=%s", event.EventType, event.ListingID)
	}

	// Validate listing has calendar enabled
	canUse, err := s.listingHooks.CanUseCalendar(ctx, event.ListingID)
	if err != nil {
		return nil, fmt.Errorf("failed to check calendar eligibility: %w", err)
	}
	if !canUse {
		return nil, domain.ErrUnauthorized
	}

	// Check for conflicts
	conflicts, err := s.repo.GetConflictingEvents(ctx, event.ListingID, event.StartTime, event.EndTime, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}

	// Filter conflicts based on business rules
	for _, conflict := range conflicts {
		domainConflict := domain.MapEventFromSchemaToEntity(conflict)
		if event.ConflictsWith(domainConflict) {
			if s.log != nil {
				s.log.Logf("WARN event conflicts with existing event id=%s", conflict.ID)
			}
			return nil, domain.ErrEventConflict
		}
	}

	// Convert to schema and save
	schemaEvent := domain.MapEventFromEntityToSchema(event)
	if err := s.repo.CreateEvent(ctx, schemaEvent); err != nil {
		if s.log != nil {
			s.log.Logf("ERROR failed to create event: %v", err)
		}
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Update domain entity with generated ID
	event.ID = schemaEvent.ID
	event.CreatedAt = schemaEvent.CreatedAt
	event.UpdatedAt = schemaEvent.UpdatedAt

	// Invalidate availability cache
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	if s.log != nil {
		s.log.Logf("INFO created calendar event id=%s", event.ID)
	}

	return event, nil
}

func (s *CalendarServiceImpl) GetEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) (*domain.CalendarEvent, error) {
	// Try cache first
	cacheKey := eventCacheKey(eventID)
	var event *domain.CalendarEvent
	cached, err := s.getCachedValue(ctx, cacheKey, &event)
	if err == nil && cached && event != nil {
		// Verify access
		if err := s.verifyEventAccess(ctx, event, requestorID); err != nil {
			return nil, err
		}
		return event, nil
	}

	// Fetch from repository
	schemaEvent, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event = domain.MapEventFromSchemaToEntity(schemaEvent)

	// Cache for future requests
	s.cacheEvent(ctx, event)

	// Verify access
	if err := s.verifyEventAccess(ctx, event, requestorID); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *CalendarServiceImpl) UpdateEvent(ctx context.Context, event *domain.CalendarEvent, requestorID uuid.UUID) (*domain.CalendarEvent, error) {
	// Verify access
	if err := s.verifyEventAccess(ctx, event, requestorID); err != nil {
		return nil, err
	}

	// Check if event can be modified
	if !event.CanBeModified() {
		return nil, domain.ErrCannotCancelEvent
	}

	// Convert to schema and update
	schemaEvent := domain.MapEventFromEntityToSchema(event)
	if err := s.repo.UpdateEvent(ctx, schemaEvent); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	// Update domain entity
	event.UpdatedAt = schemaEvent.UpdatedAt

	// Invalidate caches
	s.invalidateEventCache(ctx, event.ID)
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	if s.log != nil {
		s.log.Logf("INFO updated calendar event id=%s", event.ID)
	}

	return event, nil
}

func (s *CalendarServiceImpl) CancelEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) error {
	event, err := s.GetEvent(ctx, eventID, requestorID)
	if err != nil {
		return err
	}

	if !event.CanBeCancelled() {
		return domain.ErrCannotCancelEvent
	}

	// Update status to cancelled
	if err := s.repo.UpdateEventStatus(ctx, eventID, schema.EventStatusCancelled, event.Version); err != nil {
		return fmt.Errorf("failed to cancel event: %w", err)
	}

	// Invalidate caches
	s.invalidateEventCache(ctx, eventID)
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	if s.log != nil {
		s.log.Logf("INFO cancelled event id=%s", eventID)
	}

	return nil
}

func (s *CalendarServiceImpl) DeleteEvent(ctx context.Context, eventID uuid.UUID, requestorID uuid.UUID) error {
	event, err := s.GetEvent(ctx, eventID, requestorID)
	if err != nil {
		return err
	}

	// Soft delete
	if err := s.repo.DeleteEvent(ctx, eventID); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	// Invalidate caches
	s.invalidateEventCache(ctx, eventID)
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	if s.log != nil {
		s.log.Logf("INFO deleted event id=%s", eventID)
	}

	return nil
}

// --- Availability Checks ---

func (s *CalendarServiceImpl) CheckAvailability(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) (*domain.AvailabilityResult, error) {
	// Check cache first
	cacheKey := availabilityCacheKey(listingID, startTime, endTime)
	var result *domain.AvailabilityResult
	cached, err := s.getCachedValue(ctx, cacheKey, &result)
	if err == nil && cached && result != nil {
		return result, nil
	}

	// Get conflicts from repository
	schemaConflicts, err := s.repo.GetConflictingEvents(ctx, listingID, startTime, endTime, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}

	conflicts := domain.MapEventsFromSchema(schemaConflicts)

	// Check for blocking conflicts
	hasConflict := false
	reason := ""
	for _, conflict := range conflicts {
		if conflict.BlocksBookings() {
			hasConflict = true
			reason = fmt.Sprintf("%s event already booked", conflict.EventType)
			break
		}
	}

	result = &domain.AvailabilityResult{
		Available: !hasConflict,
		Conflicts: conflicts,
		Reason:    reason,
	}

	// Cache result for 15 minutes
	s.cacheAvailability(ctx, listingID, startTime, endTime, result)

	return result, nil
}

func (s *CalendarServiceImpl) GetConflictingEvents(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) ([]*domain.CalendarEvent, error) {
	schemaEvents, err := s.repo.GetConflictingEvents(ctx, listingID, startTime, endTime, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflicting events: %w", err)
	}
	return domain.MapEventsFromSchema(schemaEvents), nil
}

// --- Event Queries ---

func (s *CalendarServiceImpl) GetEventsForListing(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, eventTypes []domain.EventType) ([]*domain.CalendarEvent, error) {
	// Convert domain event types to schema
	var schemaTypes []schema.EventType
	for _, et := range eventTypes {
		schemaTypes = append(schemaTypes, schema.EventType(et))
	}

	schemaEvents, err := s.repo.GetEventsForListing(ctx, listingID, startTime, endTime, schemaTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return domain.MapEventsFromSchema(schemaEvents), nil
}

func (s *CalendarServiceImpl) GetUpcomingEvents(ctx context.Context, listingID uuid.UUID, limit int) ([]*domain.CalendarEvent, error) {
	schemaEvents, err := s.repo.GetUpcomingEvents(ctx, listingID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}
	return domain.MapEventsFromSchema(schemaEvents), nil
}

func (s *CalendarServiceImpl) GetEventsByStatus(ctx context.Context, listingID uuid.UUID, status domain.EventStatus, limit, offset int) ([]*domain.CalendarEvent, error) {
	schemaEvents, err := s.repo.GetEventsByStatus(ctx, listingID, schema.EventStatus(status), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by status: %w", err)
	}
	return domain.MapEventsFromSchema(schemaEvents), nil
}

func (s *CalendarServiceImpl) GetEventsForOwner(ctx context.Context, ownerID uuid.UUID, startTime, endTime time.Time) ([]*domain.CalendarEvent, error) {
	schemaEvents, err := s.repo.GetEventsForOwner(ctx, ownerID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for owner: %w", err)
	}
	return domain.MapEventsFromSchema(schemaEvents), nil
}

// --- Showing Management ---

func (s *CalendarServiceImpl) ScheduleShowing(ctx context.Context, listingID uuid.UUID, showingTime time.Time, duration time.Duration, details *domain.ShowingDetail) (*domain.CalendarEvent, error) {
	endTime := showingTime.Add(duration)

	event := &domain.CalendarEvent{
		ListingID:      listingID,
		EventType:      domain.EventTypeShowing,
		Status:         domain.EventStatusConfirmed,
		StartTime:      showingTime,
		EndTime:        endTime,
		ShowingDetails: details,
	}

	return s.CreateEvent(ctx, event)
}

func (s *CalendarServiceImpl) UpdateShowing(ctx context.Context, eventID uuid.UUID, updates *domain.ShowingDetail) (*domain.CalendarEvent, error) {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	domainEvent := domain.MapEventFromSchemaToEntity(event)
	if domainEvent.EventType != domain.EventTypeShowing {
		return nil, fmt.Errorf("event is not a showing")
	}

	domainEvent.ShowingDetails = updates
	return s.UpdateEvent(ctx, domainEvent, uuid.Nil)
}

func (s *CalendarServiceImpl) MarkShowingAttendance(ctx context.Context, eventID uuid.UUID, attended bool, feedback *string) error {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}

	if event.ShowingDetails != nil {
		event.ShowingDetails.Attended = &attended
		event.ShowingDetails.Feedback = feedback
	}

	if err := s.repo.UpdateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to mark attendance: %w", err)
	}

	s.invalidateEventCache(ctx, eventID)
	return nil
}

// --- Maintenance Management ---

func (s *CalendarServiceImpl) ScheduleMaintenance(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, details *domain.MaintenanceDetail) (*domain.CalendarEvent, error) {
	event := &domain.CalendarEvent{
		ListingID:          listingID,
		EventType:          domain.EventTypeMaintenance,
		Status:             domain.EventStatusConfirmed,
		StartTime:          startTime,
		EndTime:            endTime,
		MaintenanceDetails: details,
	}

	return s.CreateEvent(ctx, event)
}

func (s *CalendarServiceImpl) CompleteMaintenance(ctx context.Context, eventID uuid.UUID, actualCost *float64, notes *string) error {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}

	if event.MaintenanceDetails != nil {
		event.MaintenanceDetails.ActualCost = actualCost
		event.MaintenanceDetails.CompletionNotes = notes
		completed := true
		event.MaintenanceDetails.WorkCompleted = &completed
	}

	if err := s.repo.MarkEventCompleted(ctx, eventID); err != nil {
		return fmt.Errorf("failed to complete maintenance: %w", err)
	}

	if err := s.repo.UpdateEvent(ctx, event); err != nil {
		return err
	}

	s.invalidateEventCache(ctx, eventID)
	s.invalidateAvailabilityCache(ctx, event.ListingID)

	return nil
}

// --- Block Management ---

func (s *CalendarServiceImpl) BlockDates(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, reason string, ownerID uuid.UUID, ownerStay bool) (*domain.CalendarEvent, error) {
	// Verify owner
	listingOwner, err := s.listingHooks.GetListingOwner(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if listingOwner != ownerID {
		return nil, domain.ErrUnauthorized
	}

	event := &domain.CalendarEvent{
		ListingID: listingID,
		EventType: domain.EventTypeBlock,
		Status:    domain.EventStatusConfirmed,
		StartTime: startTime,
		EndTime:   endTime,
		BlockDetails: &domain.BlockDetail{
			Reason:    reason,
			OwnerStay: ownerStay,
		},
		CreatedBy: &ownerID,
	}

	return s.CreateEvent(ctx, event)
}

func (s *CalendarServiceImpl) UnblockDates(ctx context.Context, eventID uuid.UUID, ownerID uuid.UUID) error {
	return s.DeleteEvent(ctx, eventID, ownerID)
}

// --- Open House Management ---

func (s *CalendarServiceImpl) CreateOpenHouse(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, details *domain.OpenHouseDetail) (*domain.CalendarEvent, error) {
	event := &domain.CalendarEvent{
		ListingID:        listingID,
		EventType:        domain.EventTypeOpenHouse,
		Status:           domain.EventStatusConfirmed,
		StartTime:        startTime,
		EndTime:          endTime,
		OpenHouseDetails: details,
	}

	return s.CreateEvent(ctx, event)
}

func (s *CalendarServiceImpl) RegisterAttendee(ctx context.Context, eventID uuid.UUID, attendee domain.Attendee) error {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}

	domainEvent := domain.MapEventFromSchemaToEntity(event)
	if !domainEvent.CanAddAttendee() {
		return fmt.Errorf("cannot add attendee: event at capacity or not accepting registrations")
	}

	if event.OpenHouseDetails != nil {
		attendee.ID = uuid.New()
		attendee.RegisteredAt = time.Now()
		event.OpenHouseDetails.Attendees = append(event.OpenHouseDetails.Attendees, schema.Attendee{
			ID:           attendee.ID,
			Name:         attendee.Name,
			Email:        attendee.Email,
			Phone:        attendee.Phone,
			RegisteredAt: attendee.RegisteredAt,
		})
	}

	if err := s.repo.UpdateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to register attendee: %w", err)
	}

	s.invalidateEventCache(ctx, eventID)
	return nil
}

func (s *CalendarServiceImpl) MarkAttendeePresence(ctx context.Context, eventID uuid.UUID, attendeeID uuid.UUID, attended bool) error {
	event, err := s.repo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}

	if event.OpenHouseDetails != nil {
		for i := range event.OpenHouseDetails.Attendees {
			if event.OpenHouseDetails.Attendees[i].ID == attendeeID {
				event.OpenHouseDetails.Attendees[i].Attended = &attended
				break
			}
		}
	}

	if err := s.repo.UpdateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to mark attendance: %w", err)
	}

	s.invalidateEventCache(ctx, eventID)
	return nil
}

// --- Calendar Configuration ---

func (s *CalendarServiceImpl) InitializeCalendar(ctx context.Context, listingID uuid.UUID, config *domain.CalendarConfig) (*domain.CalendarConfig, error) {
	// Check if config already exists
	exists, err := s.repo.ConfigExists(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("calendar already initialized for listing")
	}

	// Set defaults if not provided
	if config.BufferHours == 0 {
		config.BufferHours = 2
	}
	if config.LeadTimeHours == 0 {
		config.LeadTimeHours = 24
	}
	if config.BookingWindowMonths == 0 {
		config.BookingWindowMonths = 12
	}
	if config.Timezone == "" {
		config.Timezone = "Africa/Lagos"
	}

	schemaConfig := domain.MapConfigFromEntityToSchema(config)
	if err := s.repo.CreateCalendarConfig(ctx, schemaConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize calendar: %w", err)
	}

	// Mark calendar as enabled in listing
	if err := s.listingHooks.MarkCalendarEnabled(ctx, listingID, true); err != nil {
		if s.log != nil {
			s.log.Logf("WARN failed to mark calendar enabled: %v", err)
		}
	}

	config.ID = schemaConfig.ID
	config.CreatedAt = schemaConfig.CreatedAt
	config.UpdatedAt = schemaConfig.UpdatedAt

	if s.log != nil {
		s.log.Logf("INFO initialized calendar for listing=%s", listingID)
	}

	return config, nil
}

func (s *CalendarServiceImpl) GetCalendarConfig(ctx context.Context, listingID uuid.UUID) (*domain.CalendarConfig, error) {
	schemaConfig, err := s.repo.GetCalendarConfig(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar config: %w", err)
	}
	return domain.MapConfigFromSchemaToEntity(schemaConfig), nil
}

func (s *CalendarServiceImpl) UpdateCalendarConfig(ctx context.Context, config *domain.CalendarConfig) (*domain.CalendarConfig, error) {
	schemaConfig := domain.MapConfigFromEntityToSchema(config)
	if err := s.repo.UpdateCalendarConfig(ctx, schemaConfig); err != nil {
		return nil, fmt.Errorf("failed to update calendar config: %w", err)
	}

	config.UpdatedAt = schemaConfig.UpdatedAt
	return config, nil
}

// --- Recurring Events ---

func (s *CalendarServiceImpl) CreateRecurringPattern(ctx context.Context, pattern *domain.RecurringEventPattern) (*domain.RecurringEventPattern, error) {
	schemaPattern := domain.MapRecurringPatternFromEntityToSchema(pattern)
	if err := s.repo.CreateRecurringPattern(ctx, schemaPattern); err != nil {
		return nil, fmt.Errorf("failed to create recurring pattern: %w", err)
	}

	pattern.ID = schemaPattern.ID
	pattern.CreatedAt = schemaPattern.CreatedAt
	pattern.UpdatedAt = schemaPattern.UpdatedAt

	return pattern, nil
}

func (s *CalendarServiceImpl) GetRecurringPatternsForListing(ctx context.Context, listingID uuid.UUID) ([]*domain.RecurringEventPattern, error) {
	schemaPatterns, err := s.repo.GetRecurringPatternsForListing(ctx, listingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring patterns: %w", err)
	}

	patterns := make([]*domain.RecurringEventPattern, len(schemaPatterns))
	for i, sp := range schemaPatterns {
		patterns[i] = domain.MapRecurringPatternFromSchemaToEntity(sp)
	}

	return patterns, nil
}

func (s *CalendarServiceImpl) UpdateRecurringPattern(ctx context.Context, pattern *domain.RecurringEventPattern) (*domain.RecurringEventPattern, error) {
	schemaPattern := domain.MapRecurringPatternFromEntityToSchema(pattern)
	if err := s.repo.UpdateRecurringPattern(ctx, schemaPattern); err != nil {
		return nil, fmt.Errorf("failed to update recurring pattern: %w", err)
	}

	pattern.UpdatedAt = schemaPattern.UpdatedAt
	return pattern, nil
}

func (s *CalendarServiceImpl) DeleteRecurringPattern(ctx context.Context, patternID uuid.UUID, ownerID uuid.UUID) error {
	// TODO: Verify ownership
	if err := s.repo.DeleteRecurringPattern(ctx, patternID); err != nil {
		return fmt.Errorf("failed to delete recurring pattern: %w", err)
	}
	return nil
}

// --- Maintenance Tasks ---

func (s *CalendarServiceImpl) MarkCompletedEvents(ctx context.Context) (int64, error) {
	count, err := s.repo.MarkEventsCompletedByEndTime(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to mark completed events: %w", err)
	}

	if s.log != nil {
		s.log.Logf("INFO marked %d events as completed", count)
	}

	return count, nil
}

func (s *CalendarServiceImpl) ArchiveOldEvents(ctx context.Context, completedBefore time.Time) (int64, error) {
	count, err := s.repo.ArchiveCompletedEvents(ctx, completedBefore)
	if err != nil {
		return 0, fmt.Errorf("failed to archive old events: %w", err)
	}

	if s.log != nil {
		s.log.Logf("INFO archived %d old events", count)
	}

	return count, nil
}

// --- Helper Methods ---

func (s *CalendarServiceImpl) verifyEventAccess(ctx context.Context, event *domain.CalendarEvent, requestorID uuid.UUID) error {
	// Get listing owner
	ownerID, err := s.listingHooks.GetListingOwner(ctx, event.ListingID)
	if err != nil {
		return err
	}

	// Owner can always access
	if ownerID == requestorID {
		return nil
	}

	// Prospect can access their own showing
	if event.ShowingDetails != nil && event.ShowingDetails.ProspectID != nil && *event.ShowingDetails.ProspectID == requestorID {
		return nil
	}

	return domain.ErrAccessDenied
}
