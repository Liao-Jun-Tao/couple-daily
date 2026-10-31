package invite

import (
	"errors"
	"testing"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
)

func TestDecide(t *testing.T) {
	now := time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)
	base := Input{
		Status:      "pending",
		ExpiresAt:   now.Add(time.Hour),
		MemberCount: 1,
		InviterID:   "inviter",
		AcceptorID:  "acceptor",
		Now:         now,
	}
	if err := Decide(base); err != nil {
		t.Fatal(err)
	}

	used := base
	used.Status = "accepted"
	if !errors.Is(Decide(used), apierr.ErrInviteUsed) {
		t.Fatalf("used: %v", Decide(used))
	}

	full := base
	full.Status = "accepted"
	full.MemberCount = 2
	if !errors.Is(Decide(full), apierr.ErrSpaceFull) {
		t.Fatalf("full: %v", Decide(full))
	}

	expired := base
	expired.ExpiresAt = now.Add(-time.Second)
	if !errors.Is(Decide(expired), apierr.ErrInviteExpired) {
		t.Fatalf("expired: %v", Decide(expired))
	}

	already := base
	already.Status = "expired"
	if !errors.Is(Decide(already), apierr.ErrInviteExpired) {
		t.Fatalf("status expired: %v", Decide(already))
	}

	self := base
	self.AcceptorID = base.InviterID
	if !errors.Is(Decide(self), apierr.ErrAlreadyPaired) {
		t.Fatalf("self: %v", Decide(self))
	}

	paired := base
	paired.AcceptorInSpace = true
	if !errors.Is(Decide(paired), apierr.ErrAlreadyPaired) {
		t.Fatalf("paired: %v", Decide(paired))
	}

	revoked := base
	revoked.Status = "revoked"
	if !errors.Is(Decide(revoked), apierr.ErrInviteInvalid) {
		t.Fatalf("revoked: %v", Decide(revoked))
	}
}
