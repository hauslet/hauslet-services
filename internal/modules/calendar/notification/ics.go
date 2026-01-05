package notification

import (
	"fmt"
	"time"

	"hauslet/internal/modules/calendar/domain"

	ics "github.com/arran4/golang-ical"
)

// BuildICS generates an iCalendar file content for a calendar event.
func BuildICS(event *domain.CalendarEvent, summary, description, location string, method ics.Method) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is nil")
	}

	// 1. Initialize the Calendar
	cal := ics.NewCalendar()
	cal.SetMethod(method) // "REQUEST" allows the user to Accept/Decline in their email client
	cal.SetProductId("-//Hauslet//Calendar//EN")
	cal.SetVersion("2.0")
	cal.SetCalscale("GREGORIAN")

	// 2. Create the Event
	// We append @hauslet.com to ensure global uniqueness for the UID
	evt := cal.AddEvent(event.ID.String() + "@hauslet.com")

	// 3. Set Time Fields (Always use UTC for interoperability)
	now := time.Now().UTC()
	evt.SetCreatedTime(now)
	evt.SetDtStampTime(now)
	evt.SetStartAt(event.StartTime.UTC())
	evt.SetEndAt(event.EndTime.UTC())

	// 4. Set Content Fields
	evt.SetSummary(summary)
	evt.SetDescription(description)

	if location != "" {
		evt.SetLocation(location)
	}

	// 5. Set Organizer (The "Sender" of the invite)
	// We use the functional options pattern provided by the library (ics.WithCN)
	evt.SetOrganizer("mailto:support@hauslet.com", ics.WithCN("Hauslet Calendar"))

	// 6. (Optional) Add Attendees
	// If you want to include the prospect or agent so it auto-adds to their calendar:
	// evt.AddAttendee("mailto:agent@example.com", ics.WithCN("Agent Name"), ics.WithRole(ics.RoleReqParticipant))

	// 7. Serialize to string
	return cal.Serialize(), nil
}
