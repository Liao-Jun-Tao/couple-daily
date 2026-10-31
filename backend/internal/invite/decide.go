// Package invite decides whether an invite code may be accepted.
package invite

import (
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
)

// Input is the locked state checked inside the accept transaction.
type Input struct {
	Status          string
	ExpiresAt       time.Time
	MemberCount     int
	InviterID       string
	AcceptorID      string
	AcceptorInSpace bool
	Now             time.Time
}

// Decide returns nil when the acceptor may join.
// A full space is SPACE_FULL even when the code was already accepted.
func Decide(in Input) error {
	if in.MemberCount >= 2 {
		return apierr.ErrSpaceFull
	}
	switch in.Status {
	case "accepted":
		return apierr.ErrInviteUsed
	case "revoked":
		return apierr.ErrInviteInvalid
	case "expired":
		return apierr.ErrInviteExpired
	case "pending":
	default:
		return apierr.ErrInviteInvalid
	}
	if !in.ExpiresAt.After(in.Now) {
		return apierr.ErrInviteExpired
	}
	if in.AcceptorInSpace || in.AcceptorID == "" || in.AcceptorID == in.InviterID {
		return apierr.ErrAlreadyPaired
	}
	if in.MemberCount != 1 {
		return apierr.ErrSpaceFull
	}
	return nil
}
