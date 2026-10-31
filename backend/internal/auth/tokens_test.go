package auth

import (
	"testing"
	"time"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	issuer := NewIssuer("test-secret", time.Hour)
	raw, err := issuer.Sign("user-1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseAccess("test-secret", "Bearer "+raw)
	if err != nil || got != "user-1" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := ParseAccess("other-secret", "Bearer "+raw); err == nil {
		t.Fatal("expected bad signature")
	}
	if _, err := ParseAccess("test-secret", raw); err == nil {
		t.Fatal("expected missing bearer")
	}
}

func TestRefreshHashStable(t *testing.T) {
	raw, hash, err := NewRefresh()
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || raw == "" || HashRefresh(raw) != hash {
		t.Fatalf("raw %q hash %q", raw, hash)
	}
}
