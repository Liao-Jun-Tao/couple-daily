package push

import (
	"context"
	"testing"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
)

func TestUnconfiguredSendersStub(t *testing.T) {
	apns := NewAPNs(config.Config{})
	fcm := NewFCM(config.Config{})
	if apns.Configured() || fcm.Configured() {
		t.Fatal("empty config should not count as configured")
	}
	msg := Message{Title: "在一起", Body: "今天是「在一起」"}
	if err := apns.Send(context.Background(), "apns-token", msg); err != nil {
		t.Fatal(err)
	}
	if err := fcm.Send(context.Background(), "fcm-token", msg); err != nil {
		t.Fatal(err)
	}
}

func TestNotifierRoutesAndroidToFCM(t *testing.T) {
	ios := &fakeSender{platform: PlatformIOS}
	android := &fakeSender{platform: PlatformAndroid}
	n := &Notifier{by: map[string]Sender{PlatformIOS: ios, PlatformAndroid: android}}
	if err := n.Send(context.Background(), PlatformAndroid, "tok", Message{Title: "hi"}); err != nil {
		t.Fatal(err)
	}
	if android.n != 1 || ios.n != 0 {
		t.Fatalf("android %d ios %d", android.n, ios.n)
	}
}

func TestNormalizeDevice(t *testing.T) {
	platform, token, err := NormalizeDevice(" Android ", " fcm-token ")
	if err != nil || platform != PlatformAndroid || token != "fcm-token" {
		t.Fatalf("platform %q token %q err %v", platform, token, err)
	}
	if _, _, err := NormalizeDevice("web", "tok"); err == nil {
		t.Fatal("expected bad platform")
	}
	if _, _, err := NormalizeDevice("ios", " "); err == nil {
		t.Fatal("expected empty token")
	}
}

func TestShouldFireAtLocalNine(t *testing.T) {
	now := time.Date(2026, 5, 20, 1, 0, 30, 0, time.UTC)
	fire, ok := ShouldFire("2024-05-20", "yearly", "09:00", "Asia/Shanghai", 0, now)
	if !ok {
		t.Fatal("expected fire")
	}
	if fire.UTC().Format(time.RFC3339) != "2026-05-20T01:00:00Z" {
		t.Fatalf("fire %s", fire.UTC().Format(time.RFC3339))
	}
	if _, ok := ShouldFire("2024-05-20", "yearly", "09:00", "Asia/Shanghai", 0, now.Add(3*time.Minute)); ok {
		t.Fatal("outside window")
	}
	if _, ok := ShouldFire("2024-05-20", "once", "09:00", "Asia/Shanghai", 0, now); ok {
		t.Fatal("once should stay in its year")
	}
	dayBefore := time.Date(2026, 5, 19, 1, 0, 10, 0, time.UTC)
	if _, ok := ShouldFire("2024-05-20", "yearly", "09:00", "Asia/Shanghai", 1, dayBefore); !ok {
		t.Fatal("expected day-before reminder")
	}
}

type fakeSender struct {
	platform string
	n        int
}

func (f *fakeSender) Platform() string { return f.platform }
func (f *fakeSender) Configured() bool { return true }
func (f *fakeSender) Send(context.Context, string, Message) error {
	f.n++
	return nil
}
