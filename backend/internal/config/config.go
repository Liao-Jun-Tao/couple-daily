// Package config reads process configuration for the API scaffold.
// Missing variables fall back to the local docker-compose defaults.
package config

import (
	"os"
	"strings"
	"time"
)

const DefaultJWTSecret = "change-me"

// Config is the runtime configuration for the API process.
type Config struct {
	HTTPAddr           string
	DatabaseURL        string
	RedisURL           string
	S3Endpoint         string
	S3Region           string
	S3Bucket           string
	S3AccessKey        string
	S3SecretKey        string
	S3UsePathStyle     bool
	JWTSecret          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	APNSKeyID          string
	APNSTeamID         string
	APNSBundleID       string
	APNSKeyPath        string
	APNSUseSandbox     bool
	FCMProjectID       string
	FCMCredentialsPath string
}

func Load() Config {
	return Config{
		HTTPAddr:           env("HTTP_ADDR", ":8080"),
		DatabaseURL:        env("DATABASE_URL", "postgres://couple:couple@localhost:5432/couple_daily?sslmode=disable"),
		RedisURL:           env("REDIS_URL", "redis://localhost:6379/0"),
		S3Endpoint:         env("S3_ENDPOINT", "http://localhost:9000"),
		S3Region:           env("S3_REGION", "us-east-1"),
		S3Bucket:           env("S3_BUCKET", "couple-daily"),
		S3AccessKey:        env("S3_ACCESS_KEY", "minio"),
		S3SecretKey:        env("S3_SECRET_KEY", "minio12345"),
		S3UsePathStyle:     envBool("S3_USE_PATH_STYLE", true),
		JWTSecret:          env("JWT_SECRET", DefaultJWTSecret),
		JWTAccessTTL:       envDuration("JWT_ACCESS_TTL", time.Hour),
		JWTRefreshTTL:      envDuration("JWT_REFRESH_TTL", 720*time.Hour),
		APNSKeyID:          env("APNS_KEY_ID", ""),
		APNSTeamID:         env("APNS_TEAM_ID", ""),
		APNSBundleID:       env("APNS_BUNDLE_ID", ""),
		APNSKeyPath:        env("APNS_KEY_PATH", ""),
		APNSUseSandbox:     envBool("APNS_USE_SANDBOX", false),
		FCMProjectID:       env("FCM_PROJECT_ID", ""),
		FCMCredentialsPath: env("FCM_CREDENTIALS_PATH", ""),
	}
}

func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
