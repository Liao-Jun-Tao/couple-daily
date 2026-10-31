// Package anniversaries scaffolds anniversary reminders.
// The locked default is 09:00 local time on the anniversary day.
package anniversaries

import (
	"errors"
	"strconv"
	"time"
	_ "time/tzdata"
)

const (
	DefaultRemindLocalTime  = "09:00"
	DefaultTimezone         = "Asia/Shanghai"
	DefaultRemindDaysBefore = 0
	maxRemindDaysBefore     = 30
)

var ErrInvalidReminder = errors.New("invalid anniversary reminder")

// Reminder is the normalized push schedule stored for an anniversary.
type Reminder struct {
	LocalTime  string
	Timezone   string
	DaysBefore int
}

// NormalizeReminder fills the locked defaults: 09:00, Asia/Shanghai, days_before 0.
// hasDaysBefore distinguishes an omitted offset from an explicit 0.
func NormalizeReminder(localTime, timezone string, daysBefore int, hasDaysBefore bool) (Reminder, error) {
	if localTime == "" {
		localTime = DefaultRemindLocalTime
	}
	if !validClock(localTime) {
		return Reminder{}, ErrInvalidReminder
	}
	if timezone == "" {
		timezone = DefaultTimezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Reminder{}, ErrInvalidReminder
	}
	if !hasDaysBefore {
		daysBefore = DefaultRemindDaysBefore
	}
	if daysBefore < 0 || daysBefore > maxRemindDaysBefore {
		return Reminder{}, ErrInvalidReminder
	}
	return Reminder{LocalTime: localTime, Timezone: timezone, DaysBefore: daysBefore}, nil
}

func validClock(v string) bool {
	if len(v) != 5 || v[2] != ':' {
		return false
	}
	hour, errH := strconv.Atoi(v[:2])
	minute, errM := strconv.Atoi(v[3:])
	if errH != nil || errM != nil {
		return false
	}
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}
