//go:build unit

package httpkit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPI(t *testing.T) {
	t.Parallel()
	ok := func(w http.ResponseWriter, _ *http.Request) error { return NoContent(w, http.StatusOK) }
	var gotErr error
	api := NewAPI(func(w http.ResponseWriter, _ *http.Request, err error) {
		gotErr = err
		w.WriteHeader(StatusCode(err))
	})
	api.HandleFunc("GET /items", ok)
	api.HandleFunc("POST /items", ok)
	api.HandleFunc("GET /items/due", ok)
	api.HandleFunc("GET /items/{id}", ok)
	api.HandleFunc("PUT /items/{id}", ok)
	api.HandleFunc("DELETE /items/{id}", ok)
	api.HandleFunc("POST /jobs", ok)
	api.HandleFunc("PATCH /jobs", ok)

	for _, tc := range []struct {
		method, path string
		status       int
		pattern      string
		allow        string
		err          error
	}{
		{http.MethodGet, "/items", 200, "GET /items", "", nil},
		{http.MethodHead, "/items", 200, "GET /items", "", nil},
		{http.MethodPost, "/items", 200, "POST /items", "", nil},
		{http.MethodGet, "/items/due", 200, "GET /items/due", "", nil},
		{http.MethodDelete, "/items/1", 200, "DELETE /items/{id}", "", nil},
		{http.MethodPut, "/items", 405, "", "GET, HEAD, POST", ErrMethodNotAllowed},
		// PUT and DELETE /items/due reach the {id} handlers, so Allow lists them.
		{http.MethodPost, "/items/due", 405, "", "DELETE, GET, HEAD, PUT", ErrMethodNotAllowed},
		{http.MethodPatch, "/items/1", 405, "", "DELETE, GET, HEAD, PUT", ErrMethodNotAllowed},
		{http.MethodHead, "/jobs", 405, "", "PATCH, POST", ErrMethodNotAllowed},
		{http.MethodGet, "/items/1/x", 404, "", "", ErrNotFound},
		{http.MethodGet, "/nope", 404, "", "", ErrNotFound},
	} {
		gotErr = nil
		rec := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil)
		api.ServeHTTP(rec, r)
		if rec.Code != tc.status || r.Pattern != tc.pattern {
			t.Errorf("%s %s: %d %q, want %d %q", tc.method, tc.path, rec.Code, r.Pattern, tc.status, tc.pattern)
		}
		if got := rec.Header().Get("Allow"); got != tc.allow {
			t.Errorf("%s %s Allow: %q, want %q", tc.method, tc.path, got, tc.allow)
		}
		if !errors.Is(gotErr, tc.err) {
			t.Errorf("%s %s error: %v, want %v", tc.method, tc.path, gotErr, tc.err)
		}
	}
}

func TestAPI_PathValues(t *testing.T) {
	t.Parallel()
	var deckID, cardID string
	api := NewAPI(func(http.ResponseWriter, *http.Request, error) { t.Error("unexpected handler error") })
	api.HandleFunc("GET /decks/{deckId}/cards/{cardId}", func(w http.ResponseWriter, r *http.Request) error {
		deckID, cardID = r.PathValue("deckId"), r.PathValue("cardId")
		return NoContent(w, http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/decks/d%2F1/cards/c2", nil))
	if rec.Code != http.StatusNoContent || deckID != "d/1" || cardID != "c2" {
		t.Fatalf("got %d deckId=%q cardId=%q, want 204 deckId=%q cardId=%q", rec.Code, deckID, cardID, "d/1", "c2")
	}
}
