package pairing

import (
	"testing"
	"time"
)

func TestNewInviteCode(t *testing.T) {
	now := time.Date(2026, 10, 29, 7, 0, 0, 0, time.UTC)
	code, exp, err := NewInviteCode(now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeCode(code); err != nil {
		t.Fatal(err)
	}
	if !exp.Equal(now.Add(InviteTTL)) {
		t.Fatalf("expiry %s", exp)
	}
	if MembersOnInviteCreate != 1 || MaxMembers != 2 {
		t.Fatal("space membership constants drifted")
	}
}

func TestNormalizeCode(t *testing.T) {
	got, err := NormalizeCode(" a3k9m2 ")
	if err != nil || got != "A3K9M2" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := NormalizeCode("A3K9M"); err != ErrInvalidInviteCode {
		t.Fatalf("err %v", err)
	}
	if _, err := NormalizeCode("A3K9M!"); err != ErrInvalidInviteCode {
		t.Fatalf("err %v", err)
	}
}
