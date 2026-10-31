// Package objectstore signs S3-compatible upload and download URLs.
package objectstore

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Signer creates URLs the client can use without the API proxying bytes.
type Signer interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, map[string]string, error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// New returns a path-style presigner, or an unsigned URL builder if setup fails.
func New(cfg config.Config) Signer {
	client := s3.New(s3.Options{
		Region:       cfg.S3Region,
		BaseEndpoint: aws.String(cfg.S3Endpoint),
		UsePathStyle: cfg.S3UsePathStyle,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, ""),
	})
	if cfg.S3Bucket == "" || cfg.S3Endpoint == "" {
		log.Printf("s3 presign disabled, using unsigned urls")
		return Unsigned{Endpoint: strings.TrimRight(cfg.S3Endpoint, "/"), Bucket: cfg.S3Bucket}
	}
	return &presigner{bucket: cfg.S3Bucket, presign: s3.NewPresignClient(client), fallback: Unsigned{
		Endpoint: strings.TrimRight(cfg.S3Endpoint, "/"),
		Bucket:   cfg.S3Bucket,
	}}
}

type presigner struct {
	bucket   string
	presign  *s3.PresignClient
	fallback Unsigned
}

func (p *presigner) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, map[string]string, error) {
	out, err := p.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	headers := map[string]string{"Content-Type": contentType}
	if err != nil {
		log.Printf("s3 presign put: %v", err)
		return p.fallback.PresignPut(ctx, key, contentType, ttl)
	}
	return out.URL, headers, nil
}

func (p *presigner) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	out, err := p.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		log.Printf("s3 presign get: %v", err)
		return p.fallback.PresignGet(ctx, key, ttl)
	}
	return out.URL, nil
}

// Unsigned builds path-style URLs when signing is unavailable.
type Unsigned struct {
	Endpoint string
	Bucket   string
}

func (u Unsigned) PresignPut(_ context.Context, key, contentType string, _ time.Duration) (string, map[string]string, error) {
	return u.objectURL(key), map[string]string{"Content-Type": contentType}, nil
}

func (u Unsigned) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return u.objectURL(key), nil
}

func (u Unsigned) objectURL(key string) string {
	base := strings.TrimRight(u.Endpoint, "/")
	if base == "" {
		base = "http://localhost:9000"
	}
	escaped := strings.ReplaceAll(url.PathEscape(key), "%2F", "/")
	return fmt.Sprintf("%s/%s/%s", base, url.PathEscape(u.Bucket), escaped)
}
