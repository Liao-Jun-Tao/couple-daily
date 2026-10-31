package push

import "time"

// ShouldFire reports whether now is inside the two-minute window after the
// anniversary's local reminder time. Yearly dates clamp to the last valid day.
func ShouldFire(eventDate, recurrence, clock, timezone string, daysBefore int, now time.Time) (time.Time, bool) {
	event, err := time.Parse("2006-01-02", eventDate)
	if err != nil || !validClock(clock) {
		return time.Time{}, false
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, false
	}
	hour, minute := clockParts(clock)
	localNow := now.In(loc)
	for _, year := range []int{localNow.Year() - 1, localNow.Year(), localNow.Year() + 1} {
		if recurrence == "once" && year != event.Year() {
			continue
		}
		day := event.Day()
		if dim := daysIn(year, event.Month()); day > dim {
			day = dim
		}
		occurrence := time.Date(year, event.Month(), day, 0, 0, 0, 0, loc)
		fireDay := occurrence.AddDate(0, 0, -daysBefore)
		fireAt := time.Date(fireDay.Year(), fireDay.Month(), fireDay.Day(), hour, minute, 0, 0, loc)
		delta := localNow.Sub(fireAt)
		if delta >= 0 && delta < 2*time.Minute {
			return fireAt, true
		}
	}
	return time.Time{}, false
}

func validClock(v string) bool {
	if len(v) != 5 || v[2] != ':' {
		return false
	}
	hour, minute := clockParts(v)
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

func clockParts(v string) (int, int) {
	if len(v) != 5 {
		return -1, -1
	}
	hour := int(v[0]-'0')*10 + int(v[1]-'0')
	minute := int(v[3]-'0')*10 + int(v[4]-'0')
	if v[0] < '0' || v[1] < '0' || v[3] < '0' || v[4] < '0' {
		return -1, -1
	}
	return hour, minute
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
