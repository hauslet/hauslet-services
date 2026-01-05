package repository

import (
	"context"
	"fmt"
	"hauslet/internal/modules/calendar/repository/schema"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Core Event CRUD ---

func (r *CalendarRepositoryImpl) CreateEvent(ctx context.Context, event *schema.CalendarEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *CalendarRepositoryImpl) GetEventByID(ctx context.Context, id uuid.UUID) (*schema.CalendarEvent, error) {
	var event schema.CalendarEvent
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *CalendarRepositoryImpl) UpdateEvent(ctx context.Context, event *schema.CalendarEvent) error {
	return r.db.WithContext(ctx).Save(event).Error
}

func (r *CalendarRepositoryImpl) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.CalendarEvent{}).Error
}

func (r *CalendarRepositoryImpl) HardDeleteEvent(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&schema.CalendarEvent{}).Error
}

// --- Event Queries ---

func (r *CalendarRepositoryImpl) GetEventsForListing(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, eventTypes []schema.EventType) ([]*schema.CalendarEvent, error) {
	query := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("end_time >= ?", startTime).
		Where("start_time <= ?", endTime).
		Order("start_time ASC")

	if len(eventTypes) > 0 {
		query = query.Where("event_type IN ?", eventTypes)
	}

	var events []*schema.CalendarEvent
	err := query.Find(&events).Error
	return events, err
}

func (r *CalendarRepositoryImpl) GetUpcomingEvents(ctx context.Context, listingID uuid.UUID, limit int) ([]*schema.CalendarEvent, error) {
	var events []*schema.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("start_time >= ?", time.Now()).
		Order("start_time ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *CalendarRepositoryImpl) GetEventsByStatus(ctx context.Context, listingID uuid.UUID, status schema.EventStatus, limit, offset int) ([]*schema.CalendarEvent, error) {
	var events []*schema.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("status = ?", status).
		Order("start_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error
	return events, err
}

func (r *CalendarRepositoryImpl) GetEventsByType(ctx context.Context, listingID uuid.UUID, eventType schema.EventType, limit, offset int) ([]*schema.CalendarEvent, error) {
	var events []*schema.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("event_type = ?", eventType).
		Order("start_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error
	return events, err
}

func (r *CalendarRepositoryImpl) GetEventsForOwner(ctx context.Context, ownerID uuid.UUID, startTime, endTime time.Time) ([]*schema.CalendarEvent, error) {
	// This requires joining with listings table to get owner_id
	var events []*schema.CalendarEvent
	err := r.db.WithContext(ctx).
		Joins("JOIN listings ON listings.id = calendar_events.listing_id").
		Where("listings.owner_id = ?", ownerID).
		Where("calendar_events.end_time >= ?", startTime).
		Where("calendar_events.start_time <= ?", endTime).
		Where("calendar_events.deleted_at IS NULL").
		Order("calendar_events.start_time ASC").
		Find(&events).Error
	return events, err
}

func (r *CalendarRepositoryImpl) GetEventsStartingBetween(ctx context.Context, startTime, endTime time.Time, eventTypes []schema.EventType, statuses []schema.EventStatus) ([]*schema.CalendarEvent, error) {
	query := r.db.WithContext(ctx).
		Where("start_time >= ?", startTime).
		Where("start_time <= ?", endTime).
		Where("deleted_at IS NULL")

	if len(eventTypes) > 0 {
		query = query.Where("event_type IN ?", eventTypes)
	}
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	var events []*schema.CalendarEvent
	err := query.Order("start_time ASC").Find(&events).Error
	return events, err
}

// --- Availability Checks ---

func (r *CalendarRepositoryImpl) CheckAvailability(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time) (bool, error) {
	hasConflict, err := r.HasConflictingBooking(ctx, listingID, startTime, endTime, nil)
	if err != nil {
		return false, err
	}
	return !hasConflict, nil
}

func (r *CalendarRepositoryImpl) GetConflictingEvents(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, excludeEventID *uuid.UUID) ([]*schema.CalendarEvent, error) {
	query := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("start_time < ?", endTime).
		Where("end_time > ?", startTime).
		Where("status NOT IN ?", []schema.EventStatus{schema.EventStatusCancelled}).
		Where("deleted_at IS NULL")

	if excludeEventID != nil {
		query = query.Where("id != ?", *excludeEventID)
	}

	var events []*schema.CalendarEvent
	err := query.Find(&events).Error
	return events, err
}

func (r *CalendarRepositoryImpl) HasConflictingBooking(ctx context.Context, listingID uuid.UUID, startTime, endTime time.Time, excludeEventID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&schema.CalendarEvent{}).
		Where("listing_id = ?", listingID).
		Where("start_time < ?", endTime).
		Where("end_time > ?", startTime).
		Where("deleted_at IS NULL")

	// Check for confirmed bookings or disruptive maintenance
	query = query.Where(r.db.Where("event_type = ? AND status IN ?", schema.EventTypeBooking, []schema.EventStatus{
		schema.EventStatusPending,
		schema.EventStatusConfirmed,
		schema.EventStatusInProgress,
	}).Or("event_type = ?", schema.EventTypeBlock).
		Or("event_type = ? AND maintenance_details->>'disruptive' = 'true'", schema.EventTypeMaintenance))

	if excludeEventID != nil {
		query = query.Where("id != ?", *excludeEventID)
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// --- Event Status Management ---

func (r *CalendarRepositoryImpl) UpdateEventStatus(ctx context.Context, eventID uuid.UUID, newStatus schema.EventStatus, version int) error {
	switch newStatus {
	case schema.EventStatusPending,
		schema.EventStatusConfirmed,
		schema.EventStatusInProgress,
		schema.EventStatusCompleted,
		schema.EventStatusCancelled,
		schema.EventStatusNoShow:
	default:
		return schema.ErrInvalidEventStatus
	}

	result := r.db.WithContext(ctx).
		Session(&gorm.Session{SkipHooks: true}).
		Model(&schema.CalendarEvent{}).
		Where("id = ?", eventID).
		Where("version = ?", version).
		Updates(map[string]interface{}{
			"status":  newStatus,
			"version": gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("optimistic lock error: event was modified by another transaction")
	}

	return nil
}

func (r *CalendarRepositoryImpl) MarkEventCompleted(ctx context.Context, eventID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&schema.CalendarEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"status":       schema.EventStatusCompleted,
			"completed_at": now,
		}).Error
}

func (r *CalendarRepositoryImpl) MarkEventsCompletedByEndTime(ctx context.Context, beforeTime time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&schema.CalendarEvent{}).
		Where("end_time < ?", beforeTime).
		Where("status IN ?", []schema.EventStatus{
			schema.EventStatusConfirmed,
			schema.EventStatusInProgress,
		}).
		Updates(map[string]interface{}{
			"status":       schema.EventStatusCompleted,
			"completed_at": time.Now(),
		})

	return result.RowsAffected, result.Error
}

// --- Calendar Config ---

func (r *CalendarRepositoryImpl) CreateCalendarConfig(ctx context.Context, config *schema.CalendarConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *CalendarRepositoryImpl) GetCalendarConfig(ctx context.Context, listingID uuid.UUID) (*schema.CalendarConfig, error) {
	var config schema.CalendarConfig
	err := r.db.WithContext(ctx).Where("listing_id = ?", listingID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *CalendarRepositoryImpl) UpdateCalendarConfig(ctx context.Context, config *schema.CalendarConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *CalendarRepositoryImpl) DeleteCalendarConfig(ctx context.Context, listingID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("listing_id = ?", listingID).Delete(&schema.CalendarConfig{}).Error
}

func (r *CalendarRepositoryImpl) ConfigExists(ctx context.Context, listingID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&schema.CalendarConfig{}).
		Where("listing_id = ?", listingID).
		Count(&count).Error
	return count > 0, err
}

// --- Recurring Event Patterns ---

func (r *CalendarRepositoryImpl) CreateRecurringPattern(ctx context.Context, pattern *schema.RecurringEventPattern) error {
	return r.db.WithContext(ctx).Create(pattern).Error
}

func (r *CalendarRepositoryImpl) GetRecurringPattern(ctx context.Context, id uuid.UUID) (*schema.RecurringEventPattern, error) {
	var pattern schema.RecurringEventPattern
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&pattern).Error
	if err != nil {
		return nil, err
	}
	return &pattern, nil
}

func (r *CalendarRepositoryImpl) GetRecurringPatternsForListing(ctx context.Context, listingID uuid.UUID) ([]*schema.RecurringEventPattern, error) {
	var patterns []*schema.RecurringEventPattern
	err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Where("active = ?", true).
		Order("created_at ASC").
		Find(&patterns).Error
	return patterns, err
}

func (r *CalendarRepositoryImpl) UpdateRecurringPattern(ctx context.Context, pattern *schema.RecurringEventPattern) error {
	return r.db.WithContext(ctx).Save(pattern).Error
}

func (r *CalendarRepositoryImpl) DeleteRecurringPattern(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&schema.RecurringEventPattern{}).Error
}

// --- Archival ---

func (r *CalendarRepositoryImpl) ArchiveCompletedEvents(ctx context.Context, completedBefore time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ?", schema.EventStatusCompleted).
		Where("completed_at < ?", completedBefore).
		Delete(&schema.CalendarEvent{})

	return result.RowsAffected, result.Error
}

// --- Transactions ---

func (r *CalendarRepositoryImpl) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
