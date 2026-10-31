package posts

import "testing"

func TestValidateContent(t *testing.T) {
	if err := ValidateContent("hello", 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateContent("   ", 1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateContent("  ", 0); err != ErrEmptyPost {
		t.Fatalf("err %v", err)
	}
	if err := ValidateContent("", 0); err != ErrEmptyPost {
		t.Fatalf("err %v", err)
	}
	if err := ValidateContent("ok", 10); err != ErrTooManyMedia {
		t.Fatalf("err %v", err)
	}
	if err := ValidateContent("ok", MaxMediaPerPost); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeVisibility(t *testing.T) {
	got, err := NormalizeVisibility("")
	if err != nil || got != DefaultLocationVisibility {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := NormalizeVisibility("street"); err != ErrInvalidVisibility {
		t.Fatalf("err %v", err)
	}
}
