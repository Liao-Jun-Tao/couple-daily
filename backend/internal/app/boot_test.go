package app

import (
	"context"
	"strings"
	"testing"

	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
)

func TestNewRejectsDefaultJWTOutsideLocal(t *testing.T) {
	_, err := New(context.Background(), config.Config{
		AppEnv:    "production",
		JWTSecret: config.DefaultJWTSecret,
	})
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("err %v", err)
	}
}
