// Package posts scaffolds the timeline, likes, and comments.
// A post may be text-only or image-only. Both empty is rejected.
// Only the author may delete a post or comment; a like is removed only by its owner.
package posts

import (
	"errors"
	"strings"
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
