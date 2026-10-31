// Package cursor encodes the opaque list cursors described in the data model.
package cursor

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

// Encode packs a timeline position as base64url(RFC3339Nano|id).
func Encode(t time.Time, id string) string {
	raw := t.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// Decode reverses Encode.
func Decode(s string) (time.Time, string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, "", err
	}
	ts, id, ok := strings.Cut(string(b), "|")
	if !ok || id == "" {
		return time.Time{}, "", errors.New("invalid cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, "", err
	}
	return t.UTC(), id, nil
}

// Trim keeps at most limit items and returns a cursor when another page exists.
func Trim[T any](items []T, limit int, at func(T) (time.Time, string)) ([]T, *string) {
	if items == nil {
		items = []T{}
	}
	if len(items) <= limit {
		return items, nil
	}
	items = items[:limit]
	t, id := at(items[len(items)-1])
	encoded := Encode(t, id)
	return items, &encoded
}
