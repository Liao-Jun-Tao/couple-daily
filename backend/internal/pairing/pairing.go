// Package pairing scaffolds invite-code pairing.
// Creating an invite also creates the CoupleSpace with member_count=1.
// Codes are 6 uppercase alphanumeric characters, live for 24h, and are single-use.
package pairing

import (
	"crypto/rand"
	"errors"
	"strings"
	"time"
)

const (
	InviteCodeLength      = 6
	InviteTTL             = 24 * time.Hour
	MaxMembers            = 2
	MembersOnInviteCreate = 1
	inviteAlphabet        = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var ErrInvalidInviteCode = errors.New("invite code must be 6 letters or digits")

// NewInviteCode returns a new code and the absolute expiry (now + 24h).
func NewInviteCode(now time.Time) (string, time.Time, error) {
	out := make([]byte, InviteCodeLength)
	buf := make([]byte, InviteCodeLength*2)
	filled := 0
	// 36 * 7 = 252, so values >= 252 are rejected to avoid modulo bias.
	const limit = byte(len(inviteAlphabet) * (256 / len(inviteAlphabet)))
	for filled < InviteCodeLength {
		if _, err := rand.Read(buf); err != nil {
			return "", time.Time{}, err
		}
		for _, b := range buf {
			if b >= limit {
				continue
			}
			out[filled] = inviteAlphabet[int(b)%len(inviteAlphabet)]
			filled++
			if filled == InviteCodeLength {
				break
			}
		}
	}
	return string(out), now.UTC().Add(InviteTTL), nil
}

// NormalizeCode trims and uppercases a code, rejecting anything except 6 [A-Z0-9].
func NormalizeCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) != InviteCodeLength {
		return "", ErrInvalidInviteCode
	}
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return "", ErrInvalidInviteCode
		}
	}
	return code, nil
}
