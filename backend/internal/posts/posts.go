// Package posts scaffolds the timeline, likes, and comments.
// A post may be text-only or image-only. Both empty is rejected.
// Only the author may delete a post or comment; a like is removed only by its owner.
package posts

import (
	"errors"
	"strings"

	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/go-chi/chi/v5"
)

const (
	MaxMediaPerPost           = 9
	MaxBodyLength             = 5000
	DefaultLocationVisibility = "exact"
)

var (
	ErrEmptyPost         = errors.New("post body and media are both empty")
	ErrTooManyMedia      = errors.New("a post can include at most 9 media")
	ErrInvalidVisibility = errors.New("location visibility must be exact, city, or hidden")
)

// Mount registers post, like, and comment routes. Handlers return 501 until implemented.
func Mount(r chi.Router) {
	r.Get("/posts", httpapi.NotImplemented)
	r.Post("/posts", httpapi.NotImplemented)
	r.Get("/posts/{postId}", httpapi.NotImplemented)
	r.Patch("/posts/{postId}", httpapi.NotImplemented)
	r.Delete("/posts/{postId}", httpapi.NotImplemented)
	r.Put("/posts/{postId}/likes", httpapi.NotImplemented)
	r.Delete("/posts/{postId}/likes", httpapi.NotImplemented)
	r.Get("/posts/{postId}/comments", httpapi.NotImplemented)
	r.Post("/posts/{postId}/comments", httpapi.NotImplemented)
	r.Delete("/comments/{commentId}", httpapi.NotImplemented)
}

// ValidateContent enforces the locked non-empty rule and the 9-image cap.
func ValidateContent(body string, mediaCount int) error {
	if mediaCount < 0 || mediaCount > MaxMediaPerPost {
		return ErrTooManyMedia
	}
	if strings.TrimSpace(body) == "" && mediaCount == 0 {
		return ErrEmptyPost
	}
	return nil
}

// NormalizeVisibility defaults an omitted visibility to exact.
func NormalizeVisibility(v string) (string, error) {
	if v == "" {
		return DefaultLocationVisibility, nil
	}
	switch v {
	case "exact", "city", "hidden":
		return v, nil
	default:
		return "", ErrInvalidVisibility
	}
}
