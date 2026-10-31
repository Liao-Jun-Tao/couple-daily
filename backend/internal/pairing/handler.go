package pairing

import (
	"net/http"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/present"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/go-chi/chi/v5"
)

// Handler serves invite and space routes.
type Handler struct {
	DB    *store.DB
	Authn func(http.Handler) http.Handler
}

// Mount registers pairing routes.
func Mount(r chi.Router, h *Handler) {
	r.Group(func(r chi.Router) {
		r.Use(h.Authn)
		r.Post("/pairing/invites", h.create)
		r.Post("/pairing/invites/accept", h.accept)
		r.Get("/pairing/space", h.space)
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := httpapi.CurrentUser(w, r)
	if !ok {
		return
	}
	for i := 0; i < 5; i++ {
		code, exp, err := NewInviteCode(time.Now().UTC())
		if err != nil {
			apierr.Write(w, err)
			return
		}
		space, err := h.DB.CreateInvite(r.Context(), uid, code, exp)
		if err == store.ErrInviteCodeConflict {
			continue
		}
		if err != nil {
			apierr.Write(w, err)
			return
		}
		httpapi.WriteJSON(w, http.StatusCreated, present.Invite{
			InviteCode: code,
			ExpiresAt:  present.FormatTime(exp),
			Space:      present.SpaceFrom(space),
		})
		return
	}
	apierr.Write(w, apierr.Validation("could not allocate an invite code"))
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	uid, ok := httpapi.CurrentUser(w, r)
	if !ok {
		return
	}
	var req struct {
		InviteCode string `json:"invite_code"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	code, err := NormalizeCode(req.InviteCode)
	if err != nil {
		apierr.Write(w, apierr.Validation(err.Error()))
		return
	}
	space, err := h.DB.AcceptInvite(r.Context(), uid, code, time.Now().UTC())
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, present.SpaceFrom(space))
}

func (h *Handler) space(w http.ResponseWriter, r *http.Request) {
	uid, ok := httpapi.CurrentUser(w, r)
	if !ok {
		return
	}
	space, err := h.DB.SpaceForUser(r.Context(), uid)
	if err == store.ErrNoSpace {
		apierr.Write(w, apierr.ErrSpaceMissing)
		return
	}
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, present.SpaceFrom(space))
}
