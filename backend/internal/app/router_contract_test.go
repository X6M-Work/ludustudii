//go:build unit

package app

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
)

func TestRouter_PathSpellingAndReference(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	docs := apidocs.NewHandler(&apidocs.Docs{Page: staticfile.File{ContentType: "text/html", Bytes: []byte("reference")}})
	handler := NewRouter(slog.New(slog.DiscardHandler), clientApp, docs, nil)
	for _, tc := range []struct {
		method, path          string
		status                int
		body, allow, location string
	}{
		{"GET", "/api/lcards/.", 307, "<a href=\"/api/lcards\">Temporary Redirect</a>.\n\n", "", "/api/lcards"},
		{"GET", "/api/lcards/..", 307, "<a href=\"/api\">Temporary Redirect</a>.\n\n", "", "/api"},
		{"GET", "/api/lcards/bad%2fx", 400, "{\"code\":\"param_error\"}\n", "", ""},
		{"GET", "/api//unknown", 307, "<a href=\"/api/unknown\">Temporary Redirect</a>.\n\n", "", "/api/unknown"},
		{"GET", "/%61pi/nope", 200, string(clientApp.IndexFile.Bytes), "", ""},
		{"GET", "/docs//", 307, "<a href=\"/docs/\">Temporary Redirect</a>.\n\n", "", "/docs/"},
		{"GET", "/client//x", 307, "<a href=\"/client/x\">Temporary Redirect</a>.\n\n", "", "/client/x"},
		{"GET", "/client/../", 307, "<a href=\"/\">Temporary Redirect</a>.\n\n", "", "/"},
		{"GET", "/docs/unknown", 200, string(clientApp.IndexFile.Bytes), "", ""},
		{"GET", "/docs/?q=x", 302, "", "", "/docs?q=x"},
		{"HEAD", "/docs/?q=x", 302, "", "", "/docs?q=x"},
		{"POST", "/docs", 405, "method not allowed\n", "GET, HEAD", ""},
		{"POST", "/docs/", 405, "method not allowed\n", "GET, HEAD", ""},
		{"POST", "/docs/openapi.json", 405, "method not allowed\n", "GET, HEAD", ""},
		{"POST", "/docs/scalar.js", 405, "method not allowed\n", "GET, HEAD", ""},
		{"OPTIONS", "/api/lcards", 405, "{\"code\":\"method_not_allowed\"}\n", "GET, HEAD, POST", ""},
		{"PATCH", "/api/lcards/id", 405, "{\"code\":\"method_not_allowed\"}\n", "GET, HEAD, PUT, DELETE", ""},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			if rec.Code != tc.status || rec.Body.String() != tc.body {
				t.Fatalf("response: %d %q, want %d %q", rec.Code, rec.Body.String(), tc.status, tc.body)
			}
			if got := rec.Header().Get("Allow"); got != tc.allow {
				t.Errorf("Allow: %q, want %q", got, tc.allow)
			}
			if got := rec.Header().Get("Location"); got != tc.location {
				t.Errorf("Location: %q, want %q", got, tc.location)
			}
		})
	}
}

func TestRouter_RequestIDHeader(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(slog.New(slog.DiscardHandler), clientApp, nil, nil)
	for _, value := range []string{"", "  client id  "} {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/nope", nil)
		r.Header.Set("X-Request-Id", value)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		id := rec.Header().Get("X-Request-Id")
		if value != "" {
			if id != value {
				t.Fatalf("ID: %q, want %q", id, value)
			}
		} else {
			if len(id) != 32 || strings.ContainsFunc(id, func(ch rune) bool { return (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') }) {
				t.Fatalf("generated ID: %q", id)
			}
		}
		if r.Header.Get("X-Request-Id") != value {
			t.Fatal("incoming header changed")
		}
	}
}

func TestRouter_PathValuesFromMux(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	var deckID, cardID string
	handler := newRouter(slog.New(slog.DiscardHandler), clientApp, nil, nil, func(rt *httpkit.Router) {
		rt.Resource("/api/decks/{deckId}/cards/{cardId}", map[string]httpkit.HandlerFunc{
			http.MethodGet: func(w http.ResponseWriter, r *http.Request) error {
				deckID, cardID = r.PathValue("deckId"), r.PathValue("cardId")
				return httpkit.NoContent(w, http.StatusNoContent)
			},
		})
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/decks/d%2F1/cards/c2", nil))
	if rec.Code != http.StatusNoContent || deckID != "d/1" || cardID != "c2" {
		t.Fatalf("got %d deckId=%q cardId=%q, want 204 deckId=%q cardId=%q", rec.Code, deckID, cardID, "d/1", "c2")
	}
}
