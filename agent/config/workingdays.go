package config

import (
	"fmt"
	"time"
)

type WorkingDay struct {
	Start string
	End   string
	TZ    string
}

// Date parser layout, don't change
const hourLayout = "15:04:05" // hh:mm:ss
// const defaultTimezone = "Europe/Paris"

func parseHour(hour string) (time.Time, error) {
	t, err := time.Parse(hourLayout, hour)
	return t, err
}

func (wd *WorkingDay) getCurrentTime() time.Time {
	loc, _ := time.LoadLocation(wd.TZ)
	return time.Now().In(loc)
}

// Trims days year etc
func (wd *WorkingDay) getCurrentTimeTrimed() time.Time {
	t := wd.getCurrentTime()
	n, _ := time.Parse(hourLayout, fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second()))
	return n
}

func (wd *WorkingDay) isWorkingDay() bool {
	dayOfWeek := wd.getCurrentTime().Weekday()
	switch dayOfWeek {
	case time.Sunday, time.Saturday:
		return false
	default:
		return true
	}
}

func (wd *WorkingDay) isWorkingHour() bool {
	cur := wd.getCurrentTimeTrimed()

	min, _ := parseHour(wd.Start)
	max, _ := parseHour(wd.End)

	// logger.Get().Debug("min: %s", min)
	// logger.Get().Debug("cur: %s", cur)
	// logger.Get().Debug("max: %s", max)

	return cur.After(min) && cur.Before(max)
}

// IsWorkingPeriod returns whenever it is a working period or not
func (wd *WorkingDay) IsWorkingPeriod() bool {
	return wd.isWorkingDay() && wd.isWorkingHour()
}
