package media

import "testing"

func TestValidateUpload(t *testing.T) {
	if err := ValidateUpload("image/jpeg", MaxImageBytes); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUpload("image/gif", 10); err != ErrUnsupportedType {
		t.Fatalf("err %v", err)
	}
	if err := ValidateUpload("image/png", MaxImageBytes+1); err != ErrInvalidSize {
		t.Fatalf("err %v", err)
	}
	if err := ValidateUpload("image/webp", 0); err != ErrInvalidSize {
		t.Fatalf("err %v", err)
	}
}
