// Package checkins scaffolds restaurant check-ins and P1 favorites.
// The place name is required and may be typed by hand; place_id may be empty.
// Only the author may delete a check-in or a favorite.
package checkins

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const maxNameLen = 128
const maxPlaceIDLen = 128

var (
	ErrInvalidName    = errors.New("name is required and must be at most 128 characters")
	ErrInvalidPlaceID = errors.New("place_id must be at most 128 characters")
)

// Validate accepts a hand-typed place name. An empty placeID is valid.
func Validate(name, placeID string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > maxNameLen {
		return "", ErrInvalidName
	}
	placeID = strings.TrimSpace(placeID)
	if utf8.RuneCountInString(placeID) > maxPlaceIDLen {
		return "", ErrInvalidPlaceID
	}
	return name, nil
}
