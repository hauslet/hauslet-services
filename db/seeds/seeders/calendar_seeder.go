package seeders

import (
	"fmt"
	"time"

	"hauslet/db/seeds/utils"
	calendarSchema "hauslet/internal/modules/calendar/repository/schema"

	"github.com/google/uuid"
)

// SeedCalendar seeds calendar events (blocks, maintenance, etc.) for existing listings.
// NOTE: Bookings will create their own calendar events when the Booking module is seeded.
func SeedCalendar(ctx *SeedContext) error {
	var listings []struct {
		ID      uuid.UUID
		OwnerID uuid.UUID
	}

	// Fetch listings that support calendars (e.g. shortlets)
	err := ctx.DB.Table("listings").Select("id, owner_id").Where("has_calendar = ?", true).Scan(&listings).Error
	if err != nil {
		return fmt.Errorf("failed to fetch listings for calendar seeds: %v", err)
	}

	if len(listings) == 0 {
		fmt.Println("    No shortlet listings found to seed calendar events")
		return nil
	}

	var events []calendarSchema.CalendarEvent
	now := time.Now()

	for _, l := range listings {
		// Create 2 to 5 block/maintenance events per listing
		numEvents := utils.RandomInt(2, 5)
		for i := 0; i < numEvents; i++ {

			// Schedule events randomly between 10 days in the past and 45 days in the future
			daysOffset := utils.RandomInt(-10, 45)
			startTime := now.Add(time.Duration(daysOffset) * 24 * time.Hour)

			// Event lasts between 1 to 3 days
			durationDays := utils.RandomInt(1, 3)
			endTime := startTime.Add(time.Duration(durationDays) * 24 * time.Hour)

			eventType := utils.RandomChoice([]calendarSchema.EventType{
				calendarSchema.EventTypeBlock,
				calendarSchema.EventTypeMaintenance,
			})

			status := calendarSchema.EventStatusConfirmed
			if startTime.After(now) {
				status = utils.RandomChoice([]calendarSchema.EventStatus{
					calendarSchema.EventStatusPending,
					calendarSchema.EventStatusConfirmed,
				})
			} else if endTime.Before(now) {
				status = calendarSchema.EventStatusCompleted
			} else {
				status = calendarSchema.EventStatusInProgress
			}

			event := calendarSchema.CalendarEvent{
				ID:        uuid.New(),
				ListingID: l.ID,
				EventType: eventType,
				Status:    status,
				StartTime: startTime,
				EndTime:   endTime,
				CreatedBy: &l.OwnerID,
				UpdatedBy: &l.OwnerID,
				CreatedAt: now.Add(-time.Hour * 24 * 7),
				UpdatedAt: now,
			}

			// Add detail structs
			switch eventType {
			case calendarSchema.EventTypeBlock:
				event.BlockDetails = &calendarSchema.BlockDetail{
					Reason:    utils.RandomChoice([]string{"Owner stay", "Deep Cleaning", "Blocked for friends"}),
					OwnerStay: utils.RandomBoolWithProbability(0.5),
				}
			case calendarSchema.EventTypeMaintenance:
				event.MaintenanceDetails = &calendarSchema.MaintenanceDetail{
					MaintenanceType: calendarSchema.MaintenanceRepair,
					Title:           utils.RandomChoice([]string{"Plumbing check", "AC Servicing", "Painting"}),
					Disruptive:      true,
				}
			}

			events = append(events, event)
		}
	}

	if len(events) > 0 {
		// Batch insert events
		if err := ctx.DB.Create(&events).Error; err != nil {
			return fmt.Errorf("failed to bulk insert calendar events: %w", err)
		}
	}

	return nil
}
