// Package apierr is the HTTP error contract shared by handlers.
package apierr

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// Error is a client-visible API failure.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func Validation(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: message}
}

var (
	ErrUnauthorized       = &Error{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "missing or invalid access token"}
	ErrTokenInvalid       = &Error{Status: http.StatusUnauthorized, Code: "TOKEN_INVALID", Message: "refresh token is invalid or expired"}
	ErrInvalidCredentials = &Error{Status: http.StatusUnauthorized, Code: "INVALID_CREDENTIALS", Message: "email or password is incorrect"}
	ErrEmailTaken         = &Error{Status: http.StatusConflict, Code: "EMAIL_TAKEN", Message: "email is already registered"}
	ErrAlreadyPaired      = &Error{Status: http.StatusConflict, Code: "ALREADY_PAIRED", Message: "user is already in a couple space"}
	ErrInviteUsed         = &Error{Status: http.StatusConflict, Code: "INVITE_USED", Message: "invite code has already been used"}
	ErrInviteExpired      = &Error{Status: http.StatusConflict, Code: "INVITE_EXPIRED", Message: "invite code has expired"}
	ErrInviteInvalid      = &Error{Status: http.StatusConflict, Code: "INVITE_INVALID", Message: "invite code is not valid"}
	ErrInviteNotFound     = &Error{Status: http.StatusNotFound, Code: "INVITE_NOT_FOUND", Message: "invite code was not found"}
	ErrSpaceFull          = &Error{Status: http.StatusConflict, Code: "SPACE_FULL", Message: "couple space already has two members"}
	ErrNotPaired          = &Error{Status: http.StatusForbidden, Code: "NOT_PAIRED", Message: "join a couple space first"}
	ErrSpaceMissing       = &Error{Status: http.StatusNotFound, Code: "NOT_PAIRED", Message: "not in a couple space"}
	ErrForbidden          = &Error{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: "only the author can do that"}
	ErrNotFound           = &Error{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: "resource not found"}
	ErrEmptyPost          = &Error{Status: http.StatusBadRequest, Code: "EMPTY_POST", Message: "post needs text or at least one image"}
	ErrMediaInvalid       = &Error{Status: http.StatusBadRequest, Code: "MEDIA_INVALID", Message: "media is missing, not yours, or already attached"}
	ErrFavoriteExists     = &Error{Status: http.StatusConflict, Code: "FAVORITE_EXISTS", Message: "this place is already saved"}
	ErrRateLimited        = &Error{Status: http.StatusTooManyRequests, Code: "RATE_LIMITED", Message: "too many login attempts; try again shortly"}
)

// Write sends err using its API status when err is an Error, otherwise 500.
func Write(w http.ResponseWriter, err error) {
	var ae *Error
	if errors.As(err, &ae) && ae != nil {
		write(w, ae.Status, ae.Code, ae.Message)
		return
	}
	log.Printf("internal error: %v", err)
	write(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func write(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}
