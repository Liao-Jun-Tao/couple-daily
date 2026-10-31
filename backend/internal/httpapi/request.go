package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/cursor"
	"github.com/Liao-Jun-Tao/couple-daily/internal/idutil"
)

// ReadJSON decodes a single JSON object from the request body.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Limit reads the OpenAPI limit query (default 20, max 50).
func Limit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 20, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 50 {
		return 0, apierr.Validation("limit must be between 1 and 50")
	}
	return n, nil
}

// Cursor reads an optional list cursor.
func Cursor(r *http.Request) (*time.Time, string, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if raw == "" {
		return nil, "", nil
	}
	t, id, err := cursor.Decode(raw)
	if err != nil || !idutil.IsUUID(id) {
		return nil, "", apierr.Validation("invalid cursor")
	}
	return &t, id, nil
}

// Discard rests the body so keep-alive connections can be reused.
func Discard(r *http.Request) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body)
	}
}
