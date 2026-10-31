package httpapi

import (
	"context"
	"net/http"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
)

type ctxKey int

const userIDKey ctxKey = 1

// WithUserID stores the authenticated user id.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID returns the authenticated user id.
func UserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}

// CurrentUser writes 401 when the request has no authenticated user.
func CurrentUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := UserID(r.Context())
	if !ok {
		apierr.Write(w, apierr.ErrUnauthorized)
		return "", false
	}
	return id, true
}
