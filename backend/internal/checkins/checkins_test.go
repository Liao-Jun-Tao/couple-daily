package checkins

import "testing"

func TestValidateManualName(t *testing.T) {
	name, err := Validate("  巷口小店  ", "")
	if err != nil || name != "巷口小店" {
		t.Fatalf("name %q err %v", name, err)
	}
	if _, err := Validate("   ", ""); err != ErrInvalidName {
		t.Fatalf("err %v", err)
	}
	if _, err := Validate("海底捞", "poi_demo_1"); err != nil {
		t.Fatal(err)
	}
}
