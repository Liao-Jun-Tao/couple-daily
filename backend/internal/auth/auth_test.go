package auth

import "testing"

func TestValidateRegister(t *testing.T) {
	email, name, err := ValidateRegister("  Alice@Example.com ", "password123", " Alice ")
	if err != nil {
		t.Fatal(err)
	}
	if email != "alice@example.com" || name != "Alice" {
		t.Fatalf("email %q name %q", email, name)
	}

	if _, _, err := ValidateRegister("not-an-email", "password123", "Alice"); err != ErrInvalidEmail {
		t.Fatalf("email err %v", err)
	}
	if _, _, err := ValidateRegister("a@example.com", "short", "Alice"); err != ErrInvalidPassword {
		t.Fatalf("password err %v", err)
	}
	if _, _, err := ValidateRegister("a@example.com", "password123", "  "); err != ErrInvalidDisplayName {
		t.Fatalf("name err %v", err)
	}
}
