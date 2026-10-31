package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("ENV", "")
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

func TestValidateJWTAndS3(t *testing.T) {
	local := Config{
		AppEnv:      "local",
		JWTSecret:   DefaultJWTSecret,
		S3Endpoint:  "http://localhost:9000",
		S3Bucket:    "couple-daily",
		S3AccessKey: "minio",
		S3SecretKey: "minio12345",
	}
	if err := local.Validate(); err != nil {
		t.Fatal(err)
	}
	dev := local
	dev.AppEnv = "dev"
	if err := dev.Validate(); err != nil {
		t.Fatal(err)
	}

	prod := local
	prod.AppEnv = "production"
	if err := prod.Validate(); err == nil {
		t.Fatal("production accepted the example JWT secret")
	}
	prod.JWTSecret = "short-secret"
	if err := prod.Validate(); err == nil {
		t.Fatal("production accepted a short JWT secret")
	}
	prod.JWTSecret = ""
	if err := prod.Validate(); err == nil {
		t.Fatal("production accepted an empty JWT secret")
	}
	unset := prod
	unset.AppEnv = ""
	unset.JWTSecret = DefaultJWTSecret
	if err := unset.Validate(); err == nil {
		t.Fatal("unset APP_ENV accepted the example JWT secret")
	}

	prod.JWTSecret = "0123456789abcdef0123456789abcdef"
	if err := prod.Validate(); err == nil {
		t.Fatal("production accepted the example MinIO credentials")
	}
	prod.S3AccessKey = "real-key"
	prod.S3SecretKey = "real-secret"
	if err := prod.Validate(); err != nil {
		t.Fatal(err)
	}
	prod.S3AccessKey = ""
	if err := prod.Validate(); err == nil {
		t.Fatal("enabled object storage accepted a missing access key")
	}
	disabled := Config{AppEnv: "production", JWTSecret: prod.JWTSecret}
	if err := disabled.Validate(); err != nil {
		t.Fatal(err)
	}
}
