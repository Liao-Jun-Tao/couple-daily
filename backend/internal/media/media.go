// Package media scaffolds presigned uploads.
// A single image is at most 10MB and must be jpeg, png, heic, or webp.
package media

import (
	"errors"

	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/go-chi/chi/v5"
)

const MaxImageBytes int64 = 10 * 1024 * 1024

var AllowedContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/heic": {},
	"image/webp": {},
}

var (
	ErrUnsupportedType = errors.New("content_type must be image/jpeg, image/png, image/heic, or image/webp")
	ErrInvalidSize     = errors.New("byte_size must be between 1 and 10485760")
)

// Mount registers the upload-intent route. The handler returns 501 until implemented.
func Mount(r chi.Router) {
	r.Post("/media/uploads", httpapi.NotImplemented)
}

// ValidateUpload checks the presign request against the locked media limits.
func ValidateUpload(contentType string, byteSize int64) error {
	if _, ok := AllowedContentTypes[contentType]; !ok {
		return ErrUnsupportedType
	}
	if byteSize < 1 || byteSize > MaxImageBytes {
		return ErrInvalidSize
	}
	return nil
}
