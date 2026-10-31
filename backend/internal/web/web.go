// Package web holds small HTTP helpers shared by handlers.
package web

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/idutil"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/go-chi/chi/v5"
)

// RequireSpace loads the caller's couple space.
// missing is the status used when the user has not joined one (403 or 404).
func RequireSpace(w http.ResponseWriter, r *http.Request, db *store.DB, missing *apierr.Error) (userID, spaceID string, ok bool) {
	uid, ok := httpapi.CurrentUser(w, r)
	if !ok {
		return "", "", false
	}
	m, err := db.Membership(r.Context(), uid)
	if errors.Is(err, store.ErrNoSpace) {
		apierr.Write(w, missing)
		return "", "", false
	}
	if err != nil {
		apierr.Write(w, err)
		return "", "", false
	}
	return uid, m.SpaceID, true
}

// PathID reads a UUID path parameter.
func PathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := chi.URLParam(r, name)
	if !idutil.IsUUID(id) {
		apierr.Write(w, apierr.Validation("invalid id"))
		return "", false
	}
	return id, true
}

// Optional trims a nullable string and enforces a rune limit.
func Optional(s *string, max int, field string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(v) > max {
		return nil, apierr.Validation(field + " is too long")
	}
	return &v, nil
}
