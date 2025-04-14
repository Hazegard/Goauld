package config

import (
	"errors"
	"fmt"
	"time"
	_ "time/tzdata"
)

type WorkingDay struct {
	Start string
	End   string
	TZ    string
}

func NewWorkingDay(start, end, tz string) *WorkingDay {
	if tz == "" {
		tz = time.Local.String()
	}
	return &WorkingDay{
		Start: start,
		End:   end,
		TZ:    tz,
	}
}

// Date parser layout, don't change
const hourLayout = "15:04" // hh:mm

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

func parseHour(hour string) (time.Time, error) {
	return time.Parse(hourLayout, hour)
}

func (wd *WorkingDay) getCurrentTime() time.Time {
	loc, _ := time.LoadLocation(wd.TZ)
	return time.Now().In(loc)
}

// Trims days year etc
func (wd *WorkingDay) getCurrentTimeTrimed() time.Time {
	t := wd.getCurrentTime()
	n, err := time.Parse(hourLayout, fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute()))
	if err != nil {
		fmt.Println(err)
	}
	return n
}

// isWorkingDay return whether the current day is a week day (Monday->Friday)
func (wd *WorkingDay) isWorkingDay() bool {
	dayOfWeek := wd.getCurrentTime().Weekday()
	switch dayOfWeek {
	case time.Sunday, time.Saturday:
		return false
	default:
		return true
	}
}

// isWorkingHour return whether the  current time is between start and end of working hours
// Without checking the current day
func (wd *WorkingDay) isWorkingHour() bool {
	cur := wd.getCurrentTimeTrimed()

	min, _ := parseHour(wd.Start)
	max, _ := parseHour(wd.End)
	return cur.After(min) && cur.Before(max)
}

// IsWorkingPeriod returns whenever it is a working period or not
func (wd *WorkingDay) IsWorkingPeriod() bool {
	return wd.isWorkingDay() && wd.isWorkingHour()
}
