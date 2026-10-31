// Package httpapi holds HTTP helpers shared by scaffold route mounts.
package httpapi

import "net/http"

// NotImplemented writes the scaffold response for a contract route whose
// handler is not implemented yet.
func NotImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"code":"NOT_IMPLEMENTED","message":"endpoint is specified but not implemented"}`))
}
