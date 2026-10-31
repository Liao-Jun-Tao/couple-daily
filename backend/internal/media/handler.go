package media

import (
	"net/http"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/objectstore"
	"github.com/Liao-Jun-Tao/couple-daily/internal/present"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/Liao-Jun-Tao/couple-daily/internal/web"
	"github.com/go-chi/chi/v5"
)

const presignTTL = 15 * time.Minute

// Handler serves upload intents.
type Handler struct {
	DB     *store.DB
	Authn  func(http.Handler) http.Handler
	Signer objectstore.Signer
}

// Mount registers the media route.
func Mount(r chi.Router, h *Handler) {
	r.Group(func(r chi.Router) {
		r.Use(h.Authn)
		r.Post("/media/uploads", h.create)
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, spaceID, ok := web.RequireSpace(w, r, h.DB, apierr.ErrNotPaired)
	if !ok {
		return
	}
	var req struct {
		ContentType string `json:"content_type"`
		ByteSize    int64  `json:"byte_size"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	if err := ValidateUpload(req.ContentType, req.ByteSize); err != nil {
		apierr.Write(w, apierr.Validation(err.Error()))
		return
	}
	id, key, err := h.DB.CreateMedia(r.Context(), uid, spaceID, req.ContentType, req.ByteSize)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	uploadURL, headers, err := h.Signer.PresignPut(r.Context(), key, req.ContentType, presignTTL)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, present.MediaUpload{
		MediaID:       id,
		UploadURL:     uploadURL,
		UploadMethod:  http.MethodPut,
		UploadHeaders: headers,
		ExpiresAt:     present.FormatTime(time.Now().UTC().Add(presignTTL)),
	})
}
