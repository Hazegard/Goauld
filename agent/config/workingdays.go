//go:build !mini

package config

import (
	"Goauld/common/log"
	"errors"
	"fmt"
	"time"

	// tzdata imported for windows build.
	_ "time/tzdata"
)

// WorkingDay holds the working day configuration.
type WorkingDay struct {
	Start string
	End   string
	TZ    string
}

// NewWorkingDay returns a new WorkingDay.
func NewWorkingDay(start, end, tz string) *WorkingDay {
	if tz == "" {
		//nolint:gosmopolitan
		tz = time.Local.String()
	}

	return &WorkingDay{
		Start: start,
		End:   end,
		TZ:    tz,
	}
}

// Date parser layout.
const hourLayout = "15:04" // hh:mm

// Validate checks if the working day is valid.
func (wd *WorkingDay) Validate() error {
	var errs []error
	_, err := parseHour(wd.Start)
	errs = append(errs, err)
	_, err = parseHour(wd.End)
	errs = append(errs, err)
	_, err = time.LoadLocation(wd.TZ)
	errs = append(errs, err)

	return errors.Join(errs...)
}

// IsWorkingPeriod returns whenever it is a working period or not.
func (wd *WorkingDay) IsWorkingPeriod() bool {
	return wd.isWorkingDay() && wd.isWorkingHour()
}

// NextStartAndNow return the current date as well as the next date the agent will be allowed to start
// Dates are returned in the configured timezone.
func (wd *WorkingDay) NextStartAndNow() (time.Time, time.Time, error) {
	var nextHourMin time.Time
	var err error
	nextHourMin, err = parseHour(wd.Start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	tz, _ := time.LoadLocation(wd.TZ)
	now := time.Now().In(tz)
	// Construct a new time using today’s Y/M/D + parsed H/M/S/nsec.
	next := time.Date(
		now.Year(), now.Month(), now.Day(),
		nextHourMin.Hour(), nextHourMin.Minute(), nextHourMin.Second(), nextHourMin.Nanosecond(),
		now.Location(),
	)

	if wd.isWorkingDay() {
		return next, now, nil
	}
	next = moveToMondayIfWeekend(next)

	return next, now, nil
}

// parseHour parses the hour and returns the time.Time object
// If the hour is invalid, an error is returned.
// The hour is expected to be in the format hh:mm.
func parseHour(hour string) (time.Time, error) {
	return time.Parse(hourLayout, hour)
}

// getCurrentTime returns the current time in the configured timezone
// The timezone is loaded from the config file.
// If the timezone is invalid, the local timezone is used.
func (wd *WorkingDay) getCurrentTime() time.Time {
	loc, _ := time.LoadLocation(wd.TZ)

	return time.Now().In(loc)
}

// Trims day year etc.
func (wd *WorkingDay) getCurrentTimeTrimed() time.Time {
	t := wd.getCurrentTime()
	n, err := time.Parse(hourLayout, fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute()))
	if err != nil {
		log.Warn().Err(err).Str("tz", wd.TZ).Msg("failed to parse time")
	}

	return n
}

// isWorkingDay return whether the current day is a week day (Monday->Friday).
func (wd *WorkingDay) isWorkingDay() bool {
	dayOfWeek := wd.getCurrentTime().Weekday()
	//nolint:exhaustive
	switch dayOfWeek {
	case time.Sunday, time.Saturday:
		return false
	default:
		return true
	}
}

// isWorkingHour return whether the current time is between start and end of working hours
// Without checking the current day.
func (wd *WorkingDay) isWorkingHour() bool {
	cur := wd.getCurrentTimeTrimed()

	minH, _ := parseHour(wd.Start)
	maxH, _ := parseHour(wd.End)

	return cur.After(minH) && cur.Before(maxH)
}

// moveToMondayIfWeekend returns t unchanged, unless it’s Saturday or Sunday,
// in which case it returns t shifted forward to the next Monday.
func moveToMondayIfWeekend(t time.Time) time.Time {
	//nolint:exhaustive
	switch t.Weekday() {
	case time.Saturday:
		// Saturday → +2 days
		return t.AddDate(0, 0, 2)
	case time.Sunday:
		// Sunday → +1 day
		return t.AddDate(0, 0, 1)
	default:
		return t
	}
}
