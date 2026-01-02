package domain

import (
	"hauslet/internal/modules/calendar/repository/schema"
)

// --- CalendarEvent Mappers ---

// MapEventFromSchemaToEntity converts schema.CalendarEvent to domain.CalendarEvent
func MapEventFromSchemaToEntity(s *schema.CalendarEvent) *CalendarEvent {
	if s == nil {
		return nil
	}

	e := &CalendarEvent{
		ID:          s.ID,
		ListingID:   s.ListingID,
		EventType:   EventType(s.EventType),
		Status:      EventStatus(s.Status),
		BookingID:   s.BookingID,
		StartTime:   s.StartTime,
		EndTime:     s.EndTime,
		CreatedBy:   s.CreatedBy,
		UpdatedBy:   s.UpdatedBy,
		CompletedAt: s.CompletedAt,
		ArchivedAt:  s.ArchivedAt,
		Version:     s.Version,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}

	if s.DeletedAt.Valid {
		deletedAt := s.DeletedAt.Time
		e.DeletedAt = &deletedAt
	}

	// Map polymorphic details
	if s.ShowingDetails != nil {
		e.ShowingDetails = mapShowingDetailFromSchema(s.ShowingDetails)
	}
	if s.MaintenanceDetails != nil {
		e.MaintenanceDetails = mapMaintenanceDetailFromSchema(s.MaintenanceDetails)
	}
	if s.BlockDetails != nil {
		e.BlockDetails = mapBlockDetailFromSchema(s.BlockDetails)
	}
	if s.OpenHouseDetails != nil {
		e.OpenHouseDetails = mapOpenHouseDetailFromSchema(s.OpenHouseDetails)
	}

	return e
}

// MapEventFromEntityToSchema converts domain.CalendarEvent to schema.CalendarEvent
func MapEventFromEntityToSchema(e *CalendarEvent) *schema.CalendarEvent {
	if e == nil {
		return nil
	}

	s := &schema.CalendarEvent{
		ID:          e.ID,
		ListingID:   e.ListingID,
		EventType:   schema.EventType(e.EventType),
		Status:      schema.EventStatus(e.Status),
		BookingID:   e.BookingID,
		StartTime:   e.StartTime,
		EndTime:     e.EndTime,
		CreatedBy:   e.CreatedBy,
		UpdatedBy:   e.UpdatedBy,
		CompletedAt: e.CompletedAt,
		ArchivedAt:  e.ArchivedAt,
		Version:     e.Version,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}

	// Map polymorphic details
	if e.ShowingDetails != nil {
		s.ShowingDetails = mapShowingDetailToSchema(e.ShowingDetails)
	}
	if e.MaintenanceDetails != nil {
		s.MaintenanceDetails = mapMaintenanceDetailToSchema(e.MaintenanceDetails)
	}
	if e.BlockDetails != nil {
		s.BlockDetails = mapBlockDetailToSchema(e.BlockDetails)
	}
	if e.OpenHouseDetails != nil {
		s.OpenHouseDetails = mapOpenHouseDetailToSchema(e.OpenHouseDetails)
	}

	return s
}

// --- ShowingDetail Mappers ---

func mapShowingDetailFromSchema(s *schema.ShowingDetail) *ShowingDetail {
	if s == nil {
		return nil
	}

	return &ShowingDetail{
		ProspectID:       s.ProspectID,
		ProspectName:     s.ProspectName,
		ProspectEmail:    s.ProspectEmail,
		ProspectPhone:    s.ProspectPhone,
		AgentID:          s.AgentID,
		AgentName:        s.AgentName,
		Notes:            s.Notes,
		Attended:         s.Attended,
		Feedback:         s.Feedback,
		RequestedBy:      s.RequestedBy,
		RequestedAt:      s.RequestedAt,
		RescheduledAt:    s.RescheduledAt,
		RescheduleCount:  s.RescheduleCount,
		RescheduleReason: s.RescheduleReason,
		CancelReason:     s.CancelReason,
	}
}

func mapShowingDetailToSchema(d *ShowingDetail) *schema.ShowingDetail {
	if d == nil {
		return nil
	}

	return &schema.ShowingDetail{
		ProspectID:       d.ProspectID,
		ProspectName:     d.ProspectName,
		ProspectEmail:    d.ProspectEmail,
		ProspectPhone:    d.ProspectPhone,
		AgentID:          d.AgentID,
		AgentName:        d.AgentName,
		Notes:            d.Notes,
		Attended:         d.Attended,
		Feedback:         d.Feedback,
		RequestedBy:      d.RequestedBy,
		RequestedAt:      d.RequestedAt,
		RescheduledAt:    d.RescheduledAt,
		RescheduleCount:  d.RescheduleCount,
		RescheduleReason: d.RescheduleReason,
		CancelReason:     d.CancelReason,
	}
}

// --- MaintenanceDetail Mappers ---

func mapMaintenanceDetailFromSchema(s *schema.MaintenanceDetail) *MaintenanceDetail {
	if s == nil {
		return nil
	}

	return &MaintenanceDetail{
		MaintenanceType:           MaintenanceType(s.MaintenanceType),
		Title:                     s.Title,
		Description:               s.Description,
		VendorID:                  s.VendorID,
		VendorName:                s.VendorName,
		VendorContact:             s.VendorContact,
		EstimatedCost:             s.EstimatedCost,
		ActualCost:                s.ActualCost,
		Disruptive:                s.Disruptive,
		RequiresGuestNotification: s.RequiresGuestNotification,
		WorkCompleted:             s.WorkCompleted,
		CompletionNotes:           s.CompletionNotes,
	}
}

func mapMaintenanceDetailToSchema(d *MaintenanceDetail) *schema.MaintenanceDetail {
	if d == nil {
		return nil
	}

	return &schema.MaintenanceDetail{
		MaintenanceType:           schema.MaintenanceType(d.MaintenanceType),
		Title:                     d.Title,
		Description:               d.Description,
		VendorID:                  d.VendorID,
		VendorName:                d.VendorName,
		VendorContact:             d.VendorContact,
		EstimatedCost:             d.EstimatedCost,
		ActualCost:                d.ActualCost,
		Disruptive:                d.Disruptive,
		RequiresGuestNotification: d.RequiresGuestNotification,
		WorkCompleted:             d.WorkCompleted,
		CompletionNotes:           d.CompletionNotes,
	}
}

// --- BlockDetail Mappers ---

func mapBlockDetailFromSchema(s *schema.BlockDetail) *BlockDetail {
	if s == nil {
		return nil
	}

	return &BlockDetail{
		Reason:    s.Reason,
		Notes:     s.Notes,
		OwnerStay: s.OwnerStay,
	}
}

func mapBlockDetailToSchema(d *BlockDetail) *schema.BlockDetail {
	if d == nil {
		return nil
	}

	return &schema.BlockDetail{
		Reason:    d.Reason,
		Notes:     d.Notes,
		OwnerStay: d.OwnerStay,
	}
}

// --- OpenHouseDetail Mappers ---

func mapOpenHouseDetailFromSchema(s *schema.OpenHouseDetail) *OpenHouseDetail {
	if s == nil {
		return nil
	}

	d := &OpenHouseDetail{
		Title:                s.Title,
		Description:          s.Description,
		MaxAttendees:         s.MaxAttendees,
		AgentID:              s.AgentID,
		AgentName:            s.AgentName,
		RegistrationDeadline: s.RegistrationDeadline,
	}

	// Map attendees
	if len(s.Attendees) > 0 {
		d.Attendees = make([]Attendee, len(s.Attendees))
		for i, attendee := range s.Attendees {
			d.Attendees[i] = Attendee{
				ID:                attendee.ID,
				Name:              attendee.Name,
				Email:             attendee.Email,
				Phone:             attendee.Phone,
				UserID:            attendee.UserID,
				RegisteredAt:      attendee.RegisteredAt,
				Attended:          attendee.Attended,
				RegistrationToken: attendee.RegistrationToken,
			}
		}
	}

	return d
}

func mapOpenHouseDetailToSchema(d *OpenHouseDetail) *schema.OpenHouseDetail {
	if d == nil {
		return nil
	}

	s := &schema.OpenHouseDetail{
		Title:                d.Title,
		Description:          d.Description,
		MaxAttendees:         d.MaxAttendees,
		AgentID:              d.AgentID,
		AgentName:            d.AgentName,
		RegistrationDeadline: d.RegistrationDeadline,
	}

	// Map attendees
	if len(d.Attendees) > 0 {
		s.Attendees = make([]schema.Attendee, len(d.Attendees))
		for i, attendee := range d.Attendees {
			s.Attendees[i] = schema.Attendee{
				ID:                attendee.ID,
				Name:              attendee.Name,
				Email:             attendee.Email,
				Phone:             attendee.Phone,
				UserID:            attendee.UserID,
				RegisteredAt:      attendee.RegisteredAt,
				Attended:          attendee.Attended,
				RegistrationToken: attendee.RegistrationToken,
			}
		}
	}

	return s
}

// --- CalendarConfig Mappers ---

// MapConfigFromSchemaToEntity converts schema.CalendarConfig to domain.CalendarConfig
func MapConfigFromSchemaToEntity(s *schema.CalendarConfig) *CalendarConfig {
	if s == nil {
		return nil
	}

	return &CalendarConfig{
		ID:                  s.ID,
		ListingID:           s.ListingID,
		BufferHours:         s.BufferHours,
		LeadTimeHours:       s.LeadTimeHours,
		Timezone:            s.Timezone,
		BookingWindowMonths: s.BookingWindowMonths,
		InstantBooking:      s.InstantBooking,
		SameDayBooking:      s.SameDayBooking,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

// MapConfigFromEntityToSchema converts domain.CalendarConfig to schema.CalendarConfig
func MapConfigFromEntityToSchema(d *CalendarConfig) *schema.CalendarConfig {
	if d == nil {
		return nil
	}

	return &schema.CalendarConfig{
		ID:                  d.ID,
		ListingID:           d.ListingID,
		BufferHours:         d.BufferHours,
		LeadTimeHours:       d.LeadTimeHours,
		Timezone:            d.Timezone,
		BookingWindowMonths: d.BookingWindowMonths,
		InstantBooking:      d.InstantBooking,
		SameDayBooking:      d.SameDayBooking,
		CreatedAt:           d.CreatedAt,
		UpdatedAt:           d.UpdatedAt,
	}
}

// --- RecurringEventPattern Mappers ---

// MapRecurringPatternFromSchemaToEntity converts schema.RecurringEventPattern to domain.RecurringEventPattern
func MapRecurringPatternFromSchemaToEntity(s *schema.RecurringEventPattern) *RecurringEventPattern {
	if s == nil {
		return nil
	}

	d := &RecurringEventPattern{
		ID:             s.ID,
		ListingID:      s.ListingID,
		EventType:      EventType(s.EventType),
		Frequency:      s.Frequency,
		Interval:       s.Interval,
		DaysOfWeek:     s.DaysOfWeek,
		DayOfMonth:     s.DayOfMonth,
		StartTimeOfDay: s.StartTimeOfDay,
		EndTimeOfDay:   s.EndTimeOfDay,
		StartDate:      s.StartDate,
		EndDate:        s.EndDate,
		Exceptions:     s.Exceptions,
		Active:         s.Active,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}

	if s.DeletedAt.Valid {
		deletedAt := s.DeletedAt.Time
		d.DeletedAt = &deletedAt
	}

	if s.MaintenanceDetails != nil {
		d.MaintenanceDetails = mapMaintenanceDetailFromSchema(s.MaintenanceDetails)
	}

	if s.OpenHouseDetails != nil {
		d.OpenHouseDetails = mapOpenHouseDetailFromSchema(s.OpenHouseDetails)
	}

	return d
}

// MapRecurringPatternFromEntityToSchema converts domain.RecurringEventPattern to schema.RecurringEventPattern
func MapRecurringPatternFromEntityToSchema(d *RecurringEventPattern) *schema.RecurringEventPattern {
	if d == nil {
		return nil
	}

	s := &schema.RecurringEventPattern{
		ID:             d.ID,
		ListingID:      d.ListingID,
		EventType:      schema.EventType(d.EventType),
		Frequency:      d.Frequency,
		Interval:       d.Interval,
		DaysOfWeek:     d.DaysOfWeek,
		DayOfMonth:     d.DayOfMonth,
		StartTimeOfDay: d.StartTimeOfDay,
		EndTimeOfDay:   d.EndTimeOfDay,
		StartDate:      d.StartDate,
		EndDate:        d.EndDate,
		Exceptions:     d.Exceptions,
		Active:         d.Active,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}

	if d.MaintenanceDetails != nil {
		s.MaintenanceDetails = mapMaintenanceDetailToSchema(d.MaintenanceDetails)
	}

	if d.OpenHouseDetails != nil {
		s.OpenHouseDetails = mapOpenHouseDetailToSchema(d.OpenHouseDetails)
	}

	return s
}

// --- Helper Functions for Batch Mapping ---

// MapEventsFromSchema converts a slice of schema events to domain events
func MapEventsFromSchema(events []*schema.CalendarEvent) []*CalendarEvent {
	if events == nil {
		return nil
	}

	result := make([]*CalendarEvent, len(events))
	for i, event := range events {
		result[i] = MapEventFromSchemaToEntity(event)
	}
	return result
}

// MapEventsToSchema converts a slice of domain events to schema events
func MapEventsToSchema(events []*CalendarEvent) []*schema.CalendarEvent {
	if events == nil {
		return nil
	}

	result := make([]*schema.CalendarEvent, len(events))
	for i, event := range events {
		result[i] = MapEventFromEntityToSchema(event)
	}
	return result
}
