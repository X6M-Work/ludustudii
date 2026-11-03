//go:build unit

package httpkit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_Resource(t *testing.T) {
	t.Parallel()
	ok := func(w http.ResponseWriter, _ *http.Request) error { return NoContent(w, http.StatusOK) }
	var gotErr error
	rt := NewRouter(func(w http.ResponseWriter, _ *http.Request, err error) {
		gotErr = err
		w.WriteHeader(StatusCode(err))
	})
	rt.Resource("/items", map[string]HandlerFunc{http.MethodPost: ok, http.MethodGet: ok})
	rt.Resource("/items/{id}", map[string]HandlerFunc{http.MethodDelete: ok, http.MethodPut: ok, http.MethodGet: ok})
	rt.Resource("/jobs", map[string]HandlerFunc{http.MethodPatch: ok, http.MethodPost: ok})

	for _, tc := range []struct {
		method, path string
		status       int
		pattern      string
		allow        string
	}{
		{http.MethodGet, "/items", 200, "GET /items", ""},
		{http.MethodHead, "/items", 200, "GET /items", ""},
		{http.MethodPost, "/items", 200, "POST /items", ""},
		{http.MethodDelete, "/items/1", 200, "DELETE /items/{id}", ""},
		{http.MethodPut, "/items", 405, "/items", "GET, HEAD, POST"},
		{http.MethodPatch, "/items/1", 405, "/items/{id}", "GET, HEAD, PUT, DELETE"},
		{http.MethodHead, "/jobs", 405, "/jobs", "POST, PATCH"},
	} {
		gotErr = nil
		rec := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil)
		rt.ServeHTTP(rec, r)
		if rec.Code != tc.status || r.Pattern != tc.pattern {
			t.Errorf("%s %s: %d %q, want %d %q", tc.method, tc.path, rec.Code, r.Pattern, tc.status, tc.pattern)
		}
		if got := rec.Header().Get("Allow"); got != tc.allow {
			t.Errorf("%s %s Allow: %q, want %q", tc.method, tc.path, got, tc.allow)
		}
		if tc.status == http.StatusMethodNotAllowed && !errors.Is(gotErr, ErrMethodNotAllowed) {
			t.Errorf("%s %s error: %v, want ErrMethodNotAllowed", tc.method, tc.path, gotErr)
		}
	}
}

func TestRouter_ResourcePanics(t *testing.T) {
	t.Parallel()
	ok := func(http.ResponseWriter, *http.Request) error { return nil }
	for name, handlers := range map[string]map[string]HandlerFunc{
		"empty":       {},
		"unsupported": {http.MethodOptions: ok},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recover() == nil {
					t.Fatal("Resource did not panic")
				}
			}()
			NewRouter(nil).Resource("/x", handlers)
		})
	}
}
