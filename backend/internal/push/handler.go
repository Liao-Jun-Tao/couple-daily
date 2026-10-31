package push

import (
	"net/http"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/present"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/go-chi/chi/v5"
)

// Handler serves device-token registration. It does not require push credentials.
type Handler struct {
	DB    *store.DB
	Authn func(http.Handler) http.Handler
}

// Mount registers /devices routes.
func Mount(r chi.Router, h *Handler) {
	r.Group(func(r chi.Router) {
		r.Use(h.Authn)
		r.Post("/devices", h.register)
		r.Delete("/devices", h.unregister)
	})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	uid, ok := httpapi.CurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Platform string `json:"platform"`
		Token    string `json:"token"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	platform, token, err := NormalizeDevice(req.Platform, req.Token)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	device, err := h.DB.UpsertDevice(r.Context(), uid, platform, token)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, present.Device{
		ID:        device.ID,
		Platform:  device.Platform,
		Token:     device.Token,
		UpdatedAt: present.FormatTime(device.UpdatedAt),
	})
}

func (h *Handler) unregister(w http.ResponseWriter, r *http.Request) {
	uid, ok := httpapi.CurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil || req.Token == "" {
		apierr.Write(w, apierr.Validation("token is required"))
		return
	}
	if err := h.DB.DeleteDevice(r.Context(), uid, req.Token); err != nil {
		apierr.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
