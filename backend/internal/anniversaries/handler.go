package anniversaries

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/cursor"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/present"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/Liao-Jun-Tao/couple-daily/internal/web"
	"github.com/go-chi/chi/v5"
)

// Handler serves anniversary routes.
type Handler struct {
	DB    *store.DB
	Authn func(http.Handler) http.Handler
}

// Mount registers anniversary routes.
func Mount(r chi.Router, h *Handler) {
	r.Group(func(r chi.Router) {
		r.Use(h.Authn)
		r.Get("/anniversaries", h.list)
		r.Post("/anniversaries", h.create)
		r.Patch("/anniversaries/{anniversaryId}", h.update)
		r.Delete("/anniversaries/{anniversaryId}", h.delete)
	})
}

type body struct {
	Title            *string `json:"title"`
	EventDate        *string `json:"event_date"`
	Recurrence       *string `json:"recurrence"`
	Remind           *bool   `json:"remind"`
	RemindDaysBefore *int    `json:"remind_days_before"`
	RemindLocalTime  *string `json:"remind_local_time"`
	Timezone         *string `json:"timezone"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	_, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	limit, err := httpapi.Limit(r)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	t, id, err := httpapi.Cursor(r)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	items, err := h.DB.ListAnniversaries(r.Context(), spaceID, limit+1, store.PageFrom(t, id))
	if err != nil {
		apierr.Write(w, err)
		return
	}
	items, next := cursor.Trim(items, limit, func(a store.Anniversary) (time.Time, string) {
		return a.CreatedAt, a.ID
	})
	out := make([]present.Anniversary, 0, len(items))
	for _, item := range items {
		out = append(out, present.AnniversaryFrom(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, present.NewList(out, next))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	var req body
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	if req.Title == nil || req.EventDate == nil {
		apierr.Write(w, apierr.Validation("title and event_date are required"))
		return
	}
	item, err := buildAnniversary(store.Anniversary{Remind: true}, req, true)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	created, err := h.DB.CreateAnniversary(r.Context(), spaceID, uid, item)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, present.AnniversaryFrom(created))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	_, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	id, ok := web.PathID(w, r, "anniversaryId")
	if !ok {
		return
	}
	current, err := h.DB.GetAnniversary(r.Context(), spaceID, id)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	var req body
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	item, err := buildAnniversary(current, req, false)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	updated, err := h.DB.UpdateAnniversary(r.Context(), spaceID, id, item)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, present.AnniversaryFrom(updated))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	id, ok := web.PathID(w, r, "anniversaryId")
	if !ok {
		return
	}
	if err := h.DB.DeleteAnniversary(r.Context(), spaceID, id, uid); err != nil {
		apierr.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func buildAnniversary(base store.Anniversary, req body, creating bool) (store.Anniversary, error) {
	if req.Title != nil || creating {
		title := ""
		if req.Title != nil {
			title = strings.TrimSpace(*req.Title)
		}
		if n := utf8.RuneCountInString(title); n < 1 || n > 128 {
			return store.Anniversary{}, apierr.Validation("title must be 1 to 128 characters")
		}
		base.Title = title
	}
	if req.EventDate != nil || creating {
		raw := ""
		if req.EventDate != nil {
			raw = strings.TrimSpace(*req.EventDate)
		}
		if _, err := time.Parse("2006-01-02", raw); err != nil {
			return store.Anniversary{}, apierr.Validation("event_date must be YYYY-MM-DD")
		}
		base.EventDate = raw
	}
	if req.Recurrence != nil {
		switch *req.Recurrence {
		case "yearly", "once":
			base.Recurrence = *req.Recurrence
		default:
			return store.Anniversary{}, apierr.Validation("recurrence must be yearly or once")
		}
	} else if base.Recurrence == "" {
		base.Recurrence = "yearly"
	}
	if req.Remind != nil {
		base.Remind = *req.Remind
	}
	local := base.RemindLocalTime
	tz := base.Timezone
	days := base.RemindDaysBefore
	hasDays := true
	if creating {
		local = ""
		tz = ""
		days = 0
		hasDays = false
	}
	if req.RemindLocalTime != nil {
		if *req.RemindLocalTime == "" {
			return store.Anniversary{}, apierr.Validation("remind_local_time is invalid")
		}
		local = *req.RemindLocalTime
	}
	if req.Timezone != nil {
		if *req.Timezone == "" {
			return store.Anniversary{}, apierr.Validation("timezone is invalid")
		}
		tz = *req.Timezone
	}
	if req.RemindDaysBefore != nil {
		days = *req.RemindDaysBefore
		hasDays = true
	}
	rem, err := NormalizeReminder(local, tz, days, hasDays)
	if err != nil {
		return store.Anniversary{}, apierr.Validation(err.Error())
	}
	base.RemindLocalTime = rem.LocalTime
	base.Timezone = rem.Timezone
	base.RemindDaysBefore = rem.DaysBefore
	return base, nil
}
