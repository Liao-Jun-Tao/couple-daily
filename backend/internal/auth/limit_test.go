package auth

import (
	"testing"
	"time"
)

func TestLoginGateStopsRepeatedEmail(t *testing.T) {
	gate := &loginGate{
		byIP:    newAttemptLimiter(100, time.Minute),
		byEmail: newAttemptLimiter(2, time.Minute),
	}
	now := time.Now()
	if !gate.allow("10.0.0.1", "a@example.com", now) || !gate.allow("10.0.0.2", "a@example.com", now) {
		t.Fatal("first attempts should pass")
	}
	if gate.allow("10.0.0.3", "a@example.com", now) {
		t.Fatal("email throttle should trip")
	}
	if !gate.allow("10.0.0.3", "b@example.com", now) {
		t.Fatal("a different email should still pass")
	}
}

func TestAttemptLimiterWindow(t *testing.T) {
	lim := newAttemptLimiter(1, time.Minute)
	now := time.Now()
	if !lim.allow("k", now) {
		t.Fatal("first")
	}
	if lim.allow("k", now.Add(time.Second)) {
		t.Fatal("second inside window")
	}
	if !lim.allow("k", now.Add(2*time.Minute)) {
		t.Fatal("after window")
	}
}
