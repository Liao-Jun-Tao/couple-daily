// Package push scaffolds device-token registration.
// iOS tokens are delivered through APNs and Android tokens through FCM.
// The API only stores the token; a user who has not enabled notifications still
// gets a successful registration. The client must not crash in that case.
package push

import (
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/go-chi/chi/v5"
)

const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
)

// Mount registers device routes. Handlers return 501 until implemented.
func Mount(r chi.Router) {
	r.Post("/devices", httpapi.NotImplemented)
	r.Delete("/devices", httpapi.NotImplemented)
}
