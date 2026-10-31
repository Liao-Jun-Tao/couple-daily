package posts

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/cursor"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/idutil"
	"github.com/Liao-Jun-Tao/couple-daily/internal/objectstore"
	"github.com/Liao-Jun-Tao/couple-daily/internal/present"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/Liao-Jun-Tao/couple-daily/internal/web"
	"github.com/go-chi/chi/v5"
)

// Handler serves posts, likes, and comments.
type Handler struct {
	DB     *store.DB
	Authn  func(http.Handler) http.Handler
	Signer objectstore.Signer
}

// Mount registers post, like, and comment routes.
func Mount(r chi.Router, h *Handler) {
	r.Group(func(r chi.Router) {
		r.Use(h.Authn)
		r.Get("/posts", h.list)
		r.Post("/posts", h.create)
		r.Get("/posts/{postId}", h.get)
		r.Patch("/posts/{postId}", h.update)
		r.Delete("/posts/{postId}", h.delete)
		r.Put("/posts/{postId}/likes", h.like)
		r.Delete("/posts/{postId}/likes", h.unlike)
		r.Get("/posts/{postId}/comments", h.listComments)
		r.Post("/posts/{postId}/comments", h.createComment)
		r.Delete("/comments/{commentId}", h.deleteComment)
	})
}

type locationReq struct {
	Lat        *float64 `json:"lat"`
	Lng        *float64 `json:"lng"`
	Name       *string  `json:"name"`
	PlaceID    *string  `json:"place_id"`
	Visibility string   `json:"visibility"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	limit, before, ok := pageQuery(w, r)
	if !ok {
		return
	}
	items, err := h.DB.ListPosts(r.Context(), spaceID, uid, limit+1, before)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	items, next := cursor.Trim(items, limit, func(p store.Post) (time.Time, string) {
		return p.CreatedAt, p.ID
	})
	out, err := present.PostsFrom(items, h.url(r))
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, present.NewList(out, next))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	var req struct {
		Body     string       `json:"body"`
		MediaIDs []string     `json:"media_ids"`
		Location *locationReq `json:"location"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	body := storedBody(req.Body)
	ids, err := cleanMediaIDs(req.MediaIDs)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	if err := validatePost(body, ids); err != nil {
		apierr.Write(w, err)
		return
	}
	var loc *store.Geo
	if req.Location != nil {
		loc, err = parseLocation(*req.Location)
		if err != nil {
			apierr.Write(w, err)
			return
		}
	}
	post, err := h.DB.CreatePost(r.Context(), spaceID, uid, body, loc, ids)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	h.writePost(w, r, http.StatusCreated, post)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	postID, ok := web.PathID(w, r, "postId")
	if !ok {
		return
	}
	post, err := h.DB.GetPost(r.Context(), spaceID, uid, postID)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	h.writePost(w, r, http.StatusOK, post)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	postID, ok := web.PathID(w, r, "postId")
	if !ok {
		return
	}
	var raw struct {
		Body     json.RawMessage `json:"body"`
		MediaIDs json.RawMessage `json:"media_ids"`
		Location json.RawMessage `json:"location"`
	}
	if err := httpapi.ReadJSON(w, r, &raw); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	current, err := h.DB.GetPost(r.Context(), spaceID, uid, postID)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	body := current.Body
	if len(raw.Body) > 0 {
		var next string
		if err := json.Unmarshal(raw.Body, &next); err != nil {
			apierr.Write(w, apierr.Validation("body must be a string"))
			return
		}
		body = storedBody(next)
	}
	ids := make([]string, 0, len(current.Media))
	for _, media := range current.Media {
		ids = append(ids, media.ID)
	}
	if len(raw.MediaIDs) > 0 {
		var next []string
		if err := json.Unmarshal(raw.MediaIDs, &next); err != nil {
			apierr.Write(w, apierr.Validation("media_ids must be an array"))
			return
		}
		ids, err = cleanMediaIDs(next)
		if err != nil {
			apierr.Write(w, err)
			return
		}
	}
	loc := current.Location
	if len(raw.Location) > 0 {
		if string(raw.Location) == "null" {
			loc = nil
		} else {
			var next locationReq
			if err := json.Unmarshal(raw.Location, &next); err != nil {
				apierr.Write(w, apierr.Validation("invalid location"))
				return
			}
			loc, err = parseLocation(next)
			if err != nil {
				apierr.Write(w, err)
				return
			}
		}
	}
	if err := validatePost(body, ids); err != nil {
		apierr.Write(w, err)
		return
	}
	post, err := h.DB.UpdatePost(r.Context(), spaceID, postID, uid, body, loc, ids)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	h.writePost(w, r, http.StatusOK, post)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	postID, ok := web.PathID(w, r, "postId")
	if !ok {
		return
	}
	if err := h.DB.DeletePost(r.Context(), spaceID, postID, uid); err != nil {
		apierr.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) like(w http.ResponseWriter, r *http.Request) { h.setLike(w, r, true) }

func (h *Handler) unlike(w http.ResponseWriter, r *http.Request) { h.setLike(w, r, false) }

func (h *Handler) setLike(w http.ResponseWriter, r *http.Request, like bool) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	postID, ok := web.PathID(w, r, "postId")
	if !ok {
		return
	}
	liked, count, err := h.DB.SetLike(r.Context(), spaceID, postID, uid, like)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, present.LikeStatus{PostID: postID, LikedByMe: liked, LikeCount: count})
}

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	_, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	postID, ok := web.PathID(w, r, "postId")
	if !ok {
		return
	}
	limit, after, ok := pageQuery(w, r)
	if !ok {
		return
	}
	items, err := h.DB.ListComments(r.Context(), spaceID, postID, limit+1, after)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	items, next := cursor.Trim(items, limit, func(c store.Comment) (time.Time, string) {
		return c.CreatedAt, c.ID
	})
	out := make([]present.Comment, 0, len(items))
	for _, item := range items {
		out = append(out, present.CommentFrom(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, present.NewList(out, next))
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	postID, ok := web.PathID(w, r, "postId")
	if !ok {
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	body := strings.TrimSpace(req.Body)
	if n := utf8.RuneCountInString(body); n < 1 || n > 2000 {
		apierr.Write(w, apierr.Validation("comment body must be 1 to 2000 characters"))
		return
	}
	comment, err := h.DB.CreateComment(r.Context(), spaceID, postID, uid, body)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, present.CommentFrom(comment))
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	commentID, ok := web.PathID(w, r, "commentId")
	if !ok {
		return
	}
	if err := h.DB.DeleteComment(r.Context(), spaceID, commentID, uid); err != nil {
		apierr.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writePost(w http.ResponseWriter, r *http.Request, status int, post store.Post) {
	out, err := present.PostFrom(post, h.url(r))
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, status, out)
}

func (h *Handler) url(r *http.Request) func(string) (string, error) {
	return func(key string) (string, error) {
		return h.Signer.PresignGet(r.Context(), key, time.Hour)
	}
}

func pageQuery(w http.ResponseWriter, r *http.Request) (int, store.Page, bool) {
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

func storedBody(body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return body
}

func validatePost(body string, ids []string) error {
	if utf8.RuneCountInString(body) > MaxBodyLength {
		return apierr.Validation("body must be at most 5000 characters")
	}
	if err := ValidateContent(body, len(ids)); err != nil {
		if errors.Is(err, ErrEmptyPost) {
			return apierr.ErrEmptyPost
		}
		return apierr.Validation(err.Error())
	}
	return nil
}

func cleanMediaIDs(ids []string) ([]string, error) {
	if ids == nil {
		return []string{}, nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !idutil.IsUUID(id) {
			return nil, apierr.Validation("media_ids must be uuids")
		}
		if _, ok := seen[id]; ok {
			return nil, apierr.Validation("media_ids must be unique")
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func parseLocation(raw locationReq) (*store.Geo, error) {
	if raw.Lat == nil || raw.Lng == nil {
		return nil, apierr.Validation("location.lat and location.lng are required")
	}
	if *raw.Lat < -90 || *raw.Lat > 90 || *raw.Lng < -180 || *raw.Lng > 180 {
		return nil, apierr.Validation("location is out of range")
	}
	vis, err := NormalizeVisibility(raw.Visibility)
	if err != nil {
		return nil, apierr.Validation(err.Error())
	}
	name, err := web.Optional(raw.Name, 256, "location.name")
	if err != nil {
		return nil, err
	}
	place, err := web.Optional(raw.PlaceID, 128, "location.place_id")
	if err != nil {
		return nil, err
	}
	return &store.Geo{Lat: *raw.Lat, Lng: *raw.Lng, Name: name, PlaceID: place, Visibility: vis}, nil
}
