package auth

import "testing"

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "password123") {
		t.Fatal("expected match")
	}
	if VerifyPassword(hash, "password124") {
		t.Fatal("expected mismatch")
	}
	if VerifyPassword("nope", "password123") {
		t.Fatal("expected malformed hash to fail")
	}
}
