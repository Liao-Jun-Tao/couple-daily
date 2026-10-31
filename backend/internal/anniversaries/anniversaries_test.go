package anniversaries

import "testing"

func TestNormalizeReminderDefaultsToLocalNine(t *testing.T) {
	got, err := NormalizeReminder("", "", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.LocalTime != "09:00" || got.Timezone != "Asia/Shanghai" || got.DaysBefore != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestNormalizeReminderRejectsBadClock(t *testing.T) {
	if _, err := NormalizeReminder("9:00", "Asia/Shanghai", 0, true); err != ErrInvalidReminder {
		t.Fatalf("err %v", err)
	}
	if _, err := NormalizeReminder("25:00", "Asia/Shanghai", 0, true); err != ErrInvalidReminder {
		t.Fatalf("err %v", err)
	}
	if _, err := NormalizeReminder("09:00", "Not/AZone", 0, true); err != ErrInvalidReminder {
		t.Fatalf("err %v", err)
	}
}
