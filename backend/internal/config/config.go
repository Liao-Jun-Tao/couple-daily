// Package config reads process configuration for the API scaffold.
// Missing variables fall back to the local docker-compose defaults.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// DefaultJWTSecret is the example value in .env.example.
	// It is accepted only when APP_ENV is local, dev, or development.
	DefaultJWTSecret = "change-me"
	// MinJWTSecretLen is the minimum length outside local/dev.
	MinJWTSecretLen = 32
)

// Config is the runtime configuration for the API process.
type Config struct {
	// AppEnv is APP_ENV, or ENV when APP_ENV is empty.
	// local, dev, and development may boot with the example JWT and MinIO secrets.
	// Any other value, including empty, refuses those secrets.
	AppEnv             string
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
		AppEnv:             currentAppEnv(),
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

// Validate rejects a boot configuration that could forge access tokens or
// sign object-storage URLs with missing credentials.
func (c Config) Validate() error {
	if err := c.validateJWT(); err != nil {
		return err
	}
	return c.validateS3()
}

// AllowsInsecureDev reports whether this process opted into local/dev secrets.
func (c Config) AllowsInsecureDev() bool {
	switch strings.ToLower(strings.TrimSpace(c.AppEnv)) {
	case "local", "dev", "development":
		return true
	default:
		return false
	}
}

func (c Config) validateJWT() error {
	secret := strings.TrimSpace(c.JWTSecret)
	weak := secret == "" || strings.EqualFold(secret, DefaultJWTSecret) || len(secret) < MinJWTSecretLen
	if !weak {
		return nil
	}
	if c.AllowsInsecureDev() && secret != "" {
		return nil
	}
	envName := c.AppEnv
	if envName == "" {
		envName = "(unset)"
	}
	return fmt.Errorf("JWT_SECRET is empty, %q, or shorter than %d characters; refusing to start with APP_ENV=%s. Set a random secret of at least %d characters, or set APP_ENV=local only on a development machine", DefaultJWTSecret, MinJWTSecretLen, envName, MinJWTSecretLen)
}

func (c Config) validateS3() error {
	enabled := strings.TrimSpace(c.S3Endpoint) != "" || strings.TrimSpace(c.S3Bucket) != ""
	if !enabled {
		return nil
	}
	if strings.TrimSpace(c.S3Endpoint) == "" || strings.TrimSpace(c.S3Bucket) == "" || strings.TrimSpace(c.S3AccessKey) == "" || strings.TrimSpace(c.S3SecretKey) == "" {
		return fmt.Errorf("object storage is enabled (S3_ENDPOINT or S3_BUCKET is set) but S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, and S3_SECRET_KEY must all be set")
	}
	exampleKey := c.S3AccessKey == "minio" && c.S3SecretKey == "minio12345"
	if exampleKey && !c.AllowsInsecureDev() {
		return fmt.Errorf("S3_ACCESS_KEY/S3_SECRET_KEY are the local MinIO example; refusing to start with APP_ENV=%s. Set real object-storage credentials, or set APP_ENV=local only on a development machine", c.AppEnv)
	}
	return nil
}

func currentAppEnv() string {
	if v := strings.TrimSpace(os.Getenv("APP_ENV")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("ENV"))
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
