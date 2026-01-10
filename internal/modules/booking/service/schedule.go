package service

import (
	calendardomain "hauslet/internal/modules/calendar/domain"
	"strings"
	"time"
)

const scheduleTimeLayout = "15:04"

func (s *BookingServiceImpl) buildScheduledTimes(checkIn, checkOut time.Time, constraints *ListingConstraints) (time.Time, time.Time) {
	var checkInTime *string
	var checkOutTime *string
	if constraints != nil {
		checkInTime = constraints.CheckInTime
		checkOutTime = constraints.CheckOutTime
	}

	return applyScheduleTime(checkIn, checkInTime, s.log),
		applyScheduleTime(checkOut, checkOutTime, s.log)
}

func applyScheduleTime(date time.Time, timeOfDay *string, log Logger) time.Time {
	if timeOfDay == nil || strings.TrimSpace(*timeOfDay) == "" {
		return date
	}

	parsed, err := time.Parse(scheduleTimeLayout, strings.TrimSpace(*timeOfDay))
	if err != nil {
		if log != nil {
			log.Warn("invalid schedule time format; using provided date time", "time", *timeOfDay, "error", err)
		}
		return date
	}

	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		parsed.Hour(),
		parsed.Minute(),
		0,
		0,
		date.Location(),
	)
}

// normalizeScheduledTimes aligns input check-in/out to the listing timezone and applies scheduled times.
func (s *BookingServiceImpl) normalizeScheduledTimes(
	checkIn, checkOut time.Time,
	constraints *ListingConstraints,
	calendarConfig *calendardomain.CalendarConfig,
) (time.Time, time.Time, *time.Location) {
	tz := ""
	if constraints != nil {
		tz = constraints.Timezone
	}
	if calendarConfig != nil && calendarConfig.Timezone != "" {
		tz = calendarConfig.Timezone
	}

	loc := checkIn.Location()
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		} else if s.log != nil {
			s.log.Warn("failed to load timezone, falling back to check-in location", "timezone", tz, "error", err)
		}
	}

	checkInInLoc := checkIn.In(loc)
	checkOutInLoc := checkOut.In(loc)

	scheduledCheckIn, scheduledCheckOut := s.buildScheduledTimes(checkInInLoc, checkOutInLoc, constraints)

	return scheduledCheckIn, scheduledCheckOut, loc
}

// Logger is a minimal interface to avoid importing slog in helpers.
type Logger interface {
	Warn(msg string, args ...any)
}
