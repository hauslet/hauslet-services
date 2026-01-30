package hooks

import (
	"context"
	"time"

	calendarservice "hauslet/internal/modules/calendar/service"
	discoveryservice "hauslet/internal/modules/discovery/service"

	"github.com/google/uuid"
)

// CalendarDiscoveryAdapter wraps CalendarService to implement CalendarDiscoveryHooks
type CalendarDiscoveryAdapter struct {
	calendarSvc calendarservice.CalendarService
}

// NewCalendarDiscoveryAdapter creates a new CalendarDiscoveryAdapter
func NewCalendarDiscoveryAdapter(calendarSvc calendarservice.CalendarService) discoveryservice.CalendarDiscoveryHooks {
	return &CalendarDiscoveryAdapter{
		calendarSvc: calendarSvc,
	}
}

// GetUnavailableListingIDs returns IDs of listings that are busy/booked in the given range
func (a *CalendarDiscoveryAdapter) GetUnavailableListingIDs(ctx context.Context, startTime, endTime time.Time) ([]uuid.UUID, error) {
	return a.calendarSvc.GetBusyListings(ctx, startTime, endTime)
}
