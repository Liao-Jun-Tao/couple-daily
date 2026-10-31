package checkins

import (
	"net/http"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/cursor"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/present"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/Liao-Jun-Tao/couple-daily/internal/web"
	"github.com/go-chi/chi/v5"
)

// Handler serves check-ins and favorites.
type Handler struct {
	DB    *store.DB
	Authn func(http.Handler) http.Handler
}

// Mount registers check-in and favorite routes.
func Mount(r chi.Router, h *Handler) {
	r.Group(func(r chi.Router) {
		r.Use(h.Authn)
		r.Get("/checkins", h.listCheckins)
		r.Post("/checkins", h.createCheckin)
		r.Delete("/checkins/{checkinId}", h.deleteCheckin)
		r.Get("/favorites", h.listFavorites)
		r.Post("/favorites", h.createFavorite)
		r.Delete("/favorites/{favoriteId}", h.deleteFavorite)
	})
}

type placeReq struct {
	Name    string   `json:"name"`
	Address *string  `json:"address"`
	Lat     *float64 `json:"lat"`
	Lng     *float64 `json:"lng"`
	PlaceID *string  `json:"place_id"`
	Note    *string  `json:"note"`
	Checked *string  `json:"checked_in_at"`
}

func (h *Handler) listCheckins(w http.ResponseWriter, r *http.Request) {
	_, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	limit, before, ok := page(w, r)
	if !ok {
		return
	}
	items, err := h.DB.ListCheckins(r.Context(), spaceID, limit+1, before)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	items, next := cursor.Trim(items, limit, func(c store.Checkin) (time.Time, string) {
		return c.CheckedInAt, c.ID
	})
	out := make([]present.Checkin, 0, len(items))
	for _, item := range items {
		out = append(out, present.CheckinFrom(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, present.NewList(out, next))
}

func (h *Handler) createCheckin(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	var req placeReq
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	name, err := Validate(req.Name, deref(req.PlaceID))
	if err != nil {
		apierr.Write(w, apierr.Validation(err.Error()))
		return
	}
	address, placeID, lat, lng, err := placeFields(req)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	when := time.Now().UTC()
	if req.Checked != nil && *req.Checked != "" {
		when, err = time.Parse(time.RFC3339, *req.Checked)
		if err != nil {
			apierr.Write(w, apierr.Validation("checked_in_at must be RFC3339"))
			return
		}
	}
	item, err := h.DB.CreateCheckin(r.Context(), spaceID, uid, name, address, placeID, lat, lng, when.UTC())
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, present.CheckinFrom(item))
}

func (h *Handler) deleteCheckin(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	id, ok := web.PathID(w, r, "checkinId")
	if !ok {
		return
	}
	if err := h.DB.DeleteCheckin(r.Context(), spaceID, id, uid); err != nil {
		apierr.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listFavorites(w http.ResponseWriter, r *http.Request) {
	_, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	limit, before, ok := page(w, r)
	if !ok {
		return
	}
	items, err := h.DB.ListFavorites(r.Context(), spaceID, limit+1, before)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	items, next := cursor.Trim(items, limit, func(f store.Favorite) (time.Time, string) {
		return f.CreatedAt, f.ID
	})
	out := make([]present.Favorite, 0, len(items))
	for _, item := range items {
		out = append(out, present.FavoriteFrom(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, present.NewList(out, next))
}

func (h *Handler) createFavorite(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	var req placeReq
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	name, err := Validate(req.Name, deref(req.PlaceID))
	if err != nil {
		apierr.Write(w, apierr.Validation(err.Error()))
		return
	}
	address, placeID, lat, lng, err := placeFields(req)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	note, err := web.Optional(req.Note, 512, "note")
	if err != nil {
		apierr.Write(w, err)
		return
	}
	item, err := h.DB.CreateFavorite(r.Context(), spaceID, uid, name, address, placeID, note, lat, lng)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, present.FavoriteFrom(item))
}

func (h *Handler) deleteFavorite(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	id, ok := web.PathID(w, r, "favoriteId")
	if !ok {
		return
	}
	if err := h.DB.DeleteFavorite(r.Context(), spaceID, id, uid); err != nil {
		apierr.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func page(w http.ResponseWriter, r *http.Request) (int, store.Page, bool) {
	limit, err := httpapi.Limit(r)
	if err != nil {
		apierr.Write(w, err)
		return 0, store.Page{}, false
	}
	t, id, err := httpapi.Cursor(r)
	if err != nil {
		apierr.Write(w, err)
		return 0, store.Page{}, false
	}
	return limit, store.PageFrom(t, id), true
}

func placeFields(req placeReq) (address, placeID *string, lat, lng *float64, err error) {
	if (req.Lat == nil) != (req.Lng == nil) {
		return nil, nil, nil, nil, apierr.Validation("lat and lng must both be set or both be omitted")
	}
	if req.Lat != nil && (*req.Lat < -90 || *req.Lat > 90 || *req.Lng < -180 || *req.Lng > 180) {
		return nil, nil, nil, nil, apierr.Validation("lat/lng out of range")
	}
	address, err = web.Optional(req.Address, 256, "address")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	placeID, err = web.Optional(req.PlaceID, 128, "place_id")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return address, placeID, req.Lat, req.Lng, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
