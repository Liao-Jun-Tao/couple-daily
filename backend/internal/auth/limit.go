package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	loginPerEmail = 10
	loginPerIP    = 30
	loginWindow   = time.Minute
)

// attemptLimiter is a fixed-window counter used to throttle login guesses.
type attemptLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

func newAttemptLimiter(limit int, window time.Duration) *attemptLimiter {
	return &attemptLimiter{limit: limit, window: window, hits: map[string][]time.Time{}}
}

func (l *attemptLimiter) allow(key string, now time.Time) bool {
	if l == nil || l.limit <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	kept := make([]time.Time, 0, len(l.hits[key])+1)
	for _, ts := range l.hits[key] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

// loginGate throttles by client IP and by email. A nil gate allows the attempt.
type loginGate struct {
	byIP    *attemptLimiter
	byEmail *attemptLimiter
}

// NewLoginGate throttles login attempts: 30 per minute per IP and 10 per minute per email.
func NewLoginGate() *loginGate {
	return &loginGate{
		byIP:    newAttemptLimiter(loginPerIP, loginWindow),
		byEmail: newAttemptLimiter(loginPerEmail, loginWindow),
	}
}

func (g *loginGate) allow(ip, email string, now time.Time) bool {
	if g == nil {
		return true
	}
	if ip != "" && !g.byIP.allow(ip, now) {
		return false
	}
	if email != "" && !g.byEmail.allow(strings.ToLower(strings.TrimSpace(email)), now) {
		return false
	}
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
