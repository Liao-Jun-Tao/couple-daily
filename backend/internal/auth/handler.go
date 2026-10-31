package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Liao-Jun-Tao/couple-daily/internal/apierr"
	"github.com/Liao-Jun-Tao/couple-daily/internal/httpapi"
	"github.com/Liao-Jun-Tao/couple-daily/internal/present"
	"github.com/Liao-Jun-Tao/couple-daily/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// Handler serves the email and password auth routes.
type Handler struct {
	DB         *store.DB
	Issuer     *Issuer
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Redis      *redis.Client
}

// Mount registers /auth routes.
func Mount(r chi.Router, h *Handler) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/refresh", h.refresh)
	r.With(Middleware(h.Secret)).Get("/auth/me", h.me)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	email, name, err := ValidateRegister(req.Email, req.Password, req.DisplayName)
	if err != nil {
		apierr.Write(w, apierr.Validation(err.Error()))
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	user, err := h.DB.CreateUser(r.Context(), email, hash, name)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	h.writeSession(w, r, http.StatusCreated, user)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil {
		apierr.Write(w, apierr.Validation("invalid JSON body"))
		return
	}
	if req.Password == "" {
		apierr.Write(w, apierr.Validation("password is required"))
		return
	}
	email, err := NormalizeEmail(req.Email)
	if err != nil {
		BurnMissingUser(req.Password)
		apierr.Write(w, apierr.ErrInvalidCredentials)
		return
	}
	user, err := h.DB.UserByEmail(r.Context(), email)
	if err != nil {
		BurnMissingUser(req.Password)
		apierr.Write(w, err)
		return
	}
	if !VerifyPassword(user.PasswordHash, req.Password) {
		apierr.Write(w, apierr.ErrInvalidCredentials)
		return
	}
	h.writeSession(w, r, http.StatusOK, user)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := httpapi.ReadJSON(w, r, &req); err != nil || req.RefreshToken == "" {
		apierr.Write(w, apierr.ErrTokenInvalid)
		return
	}
	raw, hash, err := NewRefresh()
	if err != nil {
		apierr.Write(w, err)
		return
	}
	now := time.Now().UTC()
	userID, err := h.DB.RotateRefresh(r.Context(), HashRefresh(req.RefreshToken), hash, now.Add(h.RefreshTTL), now)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	user, err := h.DB.UserByID(r.Context(), userID)
	if err != nil {
		apierr.Write(w, apierr.ErrTokenInvalid)
		return
	}
	h.remember(r.Context(), hash, user.ID)
	h.forget(r.Context(), HashRefresh(req.RefreshToken))
	access, err := h.Issuer.Sign(user.ID)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, present.AuthToken{
		AccessToken:  access,
		RefreshToken: raw,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.AccessTTL.Seconds()),
		User:         present.UserFrom(user),
	})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	uid, ok := httpapi.CurrentUser(w, r)
	if !ok {
		return
	}
	user, err := h.DB.UserByID(r.Context(), uid)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	body := present.Me{User: present.UserFrom(user), Pairing: present.Pairing{Status: "unpaired"}}
	membership, err := h.DB.Membership(r.Context(), uid)
	if err == nil {
		body.Pairing.SpaceID = &membership.SpaceID
		body.Pairing.MemberCount = &membership.MemberCount
		if membership.MemberCount >= 2 {
			body.Pairing.Status = "paired"
		} else {
			body.Pairing.Status = "pending_invite"
		}
	} else if !errors.Is(err, store.ErrNoSpace) {
		apierr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, body)
}

func (h *Handler) writeSession(w http.ResponseWriter, r *http.Request, status int, user store.User) {
	access, err := h.Issuer.Sign(user.ID)
	if err != nil {
		apierr.Write(w, err)
		return
	}
	raw, hash, err := NewRefresh()
	if err != nil {
		apierr.Write(w, err)
		return
	}
	if err := h.DB.InsertRefresh(r.Context(), user.ID, hash, time.Now().UTC().Add(h.RefreshTTL)); err != nil {
		apierr.Write(w, err)
		return
	}
	h.remember(r.Context(), hash, user.ID)
	httpapi.WriteJSON(w, status, present.AuthToken{
		AccessToken:  access,
		RefreshToken: raw,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.AccessTTL.Seconds()),
		User:         present.UserFrom(user),
	})
}

func (h *Handler) remember(ctx context.Context, hash, userID string) {
	if h.Redis == nil {
		return
	}
	if err := h.Redis.Set(ctx, "refresh:"+hash, userID, h.RefreshTTL).Err(); err != nil {
		log.Printf("redis refresh save: %v", err)
	}
}

func (h *Handler) forget(ctx context.Context, hash string) {
	if h.Redis == nil {
		return
	}
	if err := h.Redis.Del(ctx, "refresh:"+hash).Err(); err != nil {
		log.Printf("redis refresh delete: %v", err)
	}
}
