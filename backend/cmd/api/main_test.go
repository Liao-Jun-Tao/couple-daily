package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Liao-Jun-Tao/couple-daily/internal/app"
)

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	app.UnconnectedRouter().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("body %s", rr.Body.String())
	}
}

func TestContractRoutesAreMounted(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	routes := []struct {
		method string
		path   string
		public bool
	}{
		{http.MethodPost, "/v1/auth/register", true},
		{http.MethodPost, "/v1/auth/login", true},
		{http.MethodPost, "/v1/auth/refresh", false},
		{http.MethodGet, "/v1/auth/me", false},
		{http.MethodPost, "/v1/pairing/invites", false},
		{http.MethodPost, "/v1/pairing/invites/accept", false},
		{http.MethodGet, "/v1/pairing/space", false},
		{http.MethodPost, "/v1/media/uploads", false},
		{http.MethodGet, "/v1/posts", false},
		{http.MethodPost, "/v1/posts", false},
		{http.MethodGet, "/v1/posts/" + id, false},
		{http.MethodPatch, "/v1/posts/" + id, false},
		{http.MethodDelete, "/v1/posts/" + id, false},
		{http.MethodPut, "/v1/posts/" + id + "/likes", false},
		{http.MethodDelete, "/v1/posts/" + id + "/likes", false},
		{http.MethodGet, "/v1/posts/" + id + "/comments", false},
		{http.MethodPost, "/v1/posts/" + id + "/comments", false},
		{http.MethodDelete, "/v1/comments/" + id, false},
		{http.MethodGet, "/v1/checkins", false},
		{http.MethodPost, "/v1/checkins", false},
		{http.MethodDelete, "/v1/checkins/" + id, false},
		{http.MethodGet, "/v1/favorites", false},
		{http.MethodPost, "/v1/favorites", false},
		{http.MethodDelete, "/v1/favorites/" + id, false},
		{http.MethodGet, "/v1/anniversaries", false},
		{http.MethodPost, "/v1/anniversaries", false},
		{http.MethodPatch, "/v1/anniversaries/" + id, false},
		{http.MethodDelete, "/v1/anniversaries/" + id, false},
		{http.MethodPost, "/v1/devices", false},
		{http.MethodDelete, "/v1/devices", false},
	}

	h := app.UnconnectedRouter()
	for _, route := range routes {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(route.method, route.path, nil))
		want := http.StatusUnauthorized
		if route.public {
			want = http.StatusBadRequest
		}
		if rr.Code != want {
			t.Errorf("%s %s: status %d body %s", route.method, route.path, rr.Code, rr.Body.String())
		}
	}
}

func TestUnknownRoute(t *testing.T) {
	rr := httptest.NewRecorder()
	app.UnconnectedRouter().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/no-such-route", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d", rr.Code)
	}
}
