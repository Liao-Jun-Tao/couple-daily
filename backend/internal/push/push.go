// Package push registers device tokens and delivers anniversary reminders.
// iOS uses APNs and Android uses FCM. Missing credentials stub the send and
// still allow token registration, so an Android debug build can pair first.
package push

import (
	"context"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
)

const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
)

// Message is a user-visible notification.
type Message struct {
	Title string
	Body  string
}

// Sender delivers a message to one device token.
type Sender interface {
	Platform() string
	Configured() bool
	Send(ctx context.Context, deviceToken string, msg Message) error
}

// Notifier picks APNs or FCM from the device platform.
type Notifier struct {
	by map[string]Sender
}

// NewNotifier wires both providers. Unconfigured providers log and succeed.
func NewNotifier(cfg config.Config) *Notifier {
	apns := NewAPNs(cfg)
	fcm := NewFCM(cfg)
	log.Printf("push: apns configured=%t fcm configured=%t", apns.Configured(), fcm.Configured())
	return &Notifier{by: map[string]Sender{
		PlatformIOS:     apns,
		PlatformAndroid: fcm,
	}}
}

// Send delivers msg through the provider for platform.
func (n *Notifier) Send(ctx context.Context, platform, token string, msg Message) error {
	sender, ok := n.by[platform]
	if !ok || sender == nil {
		log.Printf("push stub: unknown platform=%s token_len=%d", platform, len(token))
		return nil
	}
	return sender.Send(ctx, token, msg)
}

// NormalizeDevice checks a registration against the OpenAPI device schema.
func NormalizeDevice(platform, token string) (string, string, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != PlatformIOS && platform != PlatformAndroid {
		return "", "", apierr.Validation("platform must be ios or android")
	}
	token = strings.TrimSpace(token)
	if token == "" || utf8.RuneCountInString(token) > 512 {
		return "", "", apierr.Validation("token must be 1 to 512 characters")
	}
	return platform, token, nil
}
