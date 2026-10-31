package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("S3_USE_PATH_STYLE", "")
	t.Setenv("JWT_ACCESS_TTL", "")

	cfg := Load()
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("addr %s", cfg.HTTPAddr)
	}
	if cfg.JWTSecret != DefaultJWTSecret {
		t.Fatalf("secret %s", cfg.JWTSecret)
	}
	if !cfg.S3UsePathStyle {
		t.Fatal("expected path-style S3 for MinIO")
	}
	if cfg.JWTAccessTTL.String() != "1h0m0s" {
		t.Fatalf("ttl %s", cfg.JWTAccessTTL)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("S3_USE_PATH_STYLE", "false")
	t.Setenv("JWT_ACCESS_TTL", "15m")

	cfg := Load()
	if cfg.HTTPAddr != ":9090" || cfg.S3UsePathStyle || cfg.JWTAccessTTL.String() != "15m0s" {
		t.Fatalf("%+v", cfg)
	}
}
