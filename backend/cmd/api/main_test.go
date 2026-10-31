package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	newRouter().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
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
	}{
		{http.MethodPost, "/v1/auth/register"},
		{http.MethodPost, "/v1/auth/login"},
		{http.MethodPost, "/v1/auth/refresh"},
		{http.MethodGet, "/v1/auth/me"},
		{http.MethodPost, "/v1/pairing/invites"},
		{http.MethodPost, "/v1/pairing/invites/accept"},
		{http.MethodGet, "/v1/pairing/space"},
		{http.MethodPost, "/v1/media/uploads"},
		{http.MethodGet, "/v1/posts"},
		{http.MethodPost, "/v1/posts"},
		{http.MethodGet, "/v1/posts/" + id},
		{http.MethodPatch, "/v1/posts/" + id},
		{http.MethodDelete, "/v1/posts/" + id},
		{http.MethodPut, "/v1/posts/" + id + "/likes"},
		{http.MethodDelete, "/v1/posts/" + id + "/likes"},
		{http.MethodGet, "/v1/posts/" + id + "/comments"},
		{http.MethodPost, "/v1/posts/" + id + "/comments"},
		{http.MethodDelete, "/v1/comments/" + id},
		{http.MethodGet, "/v1/checkins"},
		{http.MethodPost, "/v1/checkins"},
		{http.MethodDelete, "/v1/checkins/" + id},
		{http.MethodGet, "/v1/favorites"},
		{http.MethodPost, "/v1/favorites"},
		{http.MethodDelete, "/v1/favorites/" + id},
		{http.MethodGet, "/v1/anniversaries"},
		{http.MethodPost, "/v1/anniversaries"},
		{http.MethodPatch, "/v1/anniversaries/" + id},
		{http.MethodDelete, "/v1/anniversaries/" + id},
		{http.MethodPost, "/v1/devices"},
		{http.MethodDelete, "/v1/devices"},
	}

	h := newRouter()
	for _, route := range routes {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(route.method, route.path, nil))
		if rr.Code != http.StatusNotImplemented {
			t.Errorf("%s %s: status %d body %s", route.method, route.path, rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), `"code":"NOT_IMPLEMENTED"`) {
			t.Errorf("%s %s: body %s", route.method, route.path, rr.Body.String())
		}
	}
}

func TestUnknownRoute(t *testing.T) {
	rr := httptest.NewRecorder()
	newRouter().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/no-such-route", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status %d", rr.Code)
	}
}
