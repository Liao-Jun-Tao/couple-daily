package cursor

import (
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	want := time.Date(2026, 10, 29, 7, 0, 0, 123456789, time.UTC)
	id := "550e8400-e29b-41d4-a716-446655440000"
	gotT, gotID, err := Decode(Encode(want, id))
	if err != nil {
		t.Fatal(err)
	}
	if !gotT.Equal(want) || gotID != id {
		t.Fatalf("time %s id %s", gotT, gotID)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	if _, _, err := Decode("not-a-cursor"); err == nil {
		t.Fatal("expected error")
	}
}

func TestTrim(t *testing.T) {
	items := []int{1, 2, 3}
	page, next := Trim(items, 2, func(n int) (time.Time, string) {
		return time.Unix(int64(n), 0).UTC(), "id"
	})
	if len(page) != 2 || next == nil {
		t.Fatalf("page %v next %v", page, next)
	}
	page, next = Trim([]int{1}, 2, func(n int) (time.Time, string) {
		return time.Time{}, "id"
	})
	if len(page) != 1 || next != nil {
		t.Fatalf("page %v next %v", page, next)
	}
}
