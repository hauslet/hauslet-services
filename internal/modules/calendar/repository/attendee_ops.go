package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"hauslet/internal/modules/calendar/repository/schema"

	"github.com/google/uuid"
)

// AddOpenHouseAttendee atomically adds an attendee to an open house event's JSONB attendees array.
// This uses PostgreSQL's || operator to append to the JSONB array without race conditions.
func (r *CalendarRepositoryImpl) AddOpenHouseAttendee(ctx context.Context, eventID uuid.UUID, attendee schema.Attendee) error {
	// Marshal the attendee to JSON
	attendeeJSON, err := json.Marshal(attendee)
	if err != nil {
		return fmt.Errorf("failed to marshal attendee: %w", err)
	}

	// Use raw SQL to atomically append the attendee to the JSONB array
	// The || operator concatenates JSONB values
	// COALESCE ensures we initialize an empty array if open_house_details is null
	query := `
		UPDATE calendar_events 
		SET open_house_details = COALESCE(open_house_details, '{}'::jsonb) 
		    || jsonb_build_object(
		        'attendees', 
		        COALESCE(open_house_details->'attendees', '[]'::jsonb) || ?::jsonb
		    ),
		    updated_at = NOW()
		WHERE id = ? 
		AND event_type = 'open_house'
		AND deleted_at IS NULL
	`

	result := r.db.WithContext(ctx).Exec(query, attendeeJSON, eventID)
	if result.Error != nil {
		return fmt.Errorf("failed to add open house attendee: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("open house event not found or invalid event type")
	}

	return nil
}

// RemoveOpenHouseAttendee atomically removes an attendee from an open house event by email.
// This uses PostgreSQL's jsonb_path_query_array to filter out the matching attendee.
func (r *CalendarRepositoryImpl) RemoveOpenHouseAttendee(ctx context.Context, eventID uuid.UUID, email string) error {
	// Use jsonb_path_query_array to filter out the attendee with matching email
	// The $.attendees[*] path selects all attendees
	// The ? (@.email != $email) filter removes the one with the target email
	query := `
		UPDATE calendar_events 
		SET open_house_details = COALESCE(open_house_details, '{}'::jsonb) 
		    || jsonb_build_object(
		        'attendees',
		        (
		            SELECT COALESCE(jsonb_agg(elem), '[]'::jsonb)
		            FROM jsonb_array_elements(COALESCE(open_house_details->'attendees', '[]'::jsonb)) AS elem
		            WHERE elem->>'email' != ?
		        )
		    ),
		    updated_at = NOW()
		WHERE id = ? 
		AND event_type = 'open_house'
		AND deleted_at IS NULL
	`

	result := r.db.WithContext(ctx).Exec(query, email, eventID)
	if result.Error != nil {
		return fmt.Errorf("failed to remove open house attendee: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("open house event not found or invalid event type")
	}

	return nil
}

// GetOpenHouseAttendeeCount returns the current number of registered attendees for an open house.
// This uses PostgreSQL's jsonb_array_length to count the attendees array.
func (r *CalendarRepositoryImpl) GetOpenHouseAttendeeCount(ctx context.Context, eventID uuid.UUID) (int, error) {
	var count int

	query := `
		SELECT COALESCE(jsonb_array_length(open_house_details->'attendees'), 0) AS count
		FROM calendar_events
		WHERE id = ?
		AND event_type = 'open_house'
		AND deleted_at IS NULL
	`

	if err := r.db.WithContext(ctx).Raw(query, eventID).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get open house attendee count: %w", err)
	}

	return count, nil
}
