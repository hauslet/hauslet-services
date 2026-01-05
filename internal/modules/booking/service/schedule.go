package service

import (
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

// Logger is a minimal interface to avoid importing slog in helpers.
type Logger interface {
	Warn(msg string, args ...any)
}
