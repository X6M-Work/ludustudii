//go:build unit

package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
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
		{"GET", "/api/lcards/bad%2fx", 400, "{\"code\":\"validation_error\",\"errors\":[{\"field\":\"id\",\"code\":\"invalid_uuid7\"}]}\n", "", ""},
		{"GET", "/api//unknown", 307, "<a href=\"/api/unknown\">Temporary Redirect</a>.\n\n", "", "/api/unknown"},
		{"GET", "/%61pi/nope", 404, "{\"code\":\"not_found\"}\n", "", ""},
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
		{"PATCH", "/api/lcards/id", 405, "{\"code\":\"method_not_allowed\"}\n", "DELETE, GET, HEAD, PUT", ""},
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
	for _, tc := range []struct {
		name  string
		value string
		reuse bool
	}{
		{name: "missing"},
		{name: "one character", value: "a", reuse: true},
		{name: "allowed characters", value: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._-", reuse: true},
		{name: "maximum length", value: strings.Repeat("a", 128), reuse: true},
		{name: "over maximum length", value: strings.Repeat("a", 129)},
		{name: "spaces", value: "  client id  "},
		{name: "tab", value: "client\tid"},
		{name: "newline", value: "client\nid"},
		{name: "carriage return", value: "client\rid"},
		{name: "null", value: "client\x00id"},
		{name: "DEL", value: "client\x7fid"},
		{name: "non-ASCII", value: "client-é"},
		{name: "punctuation", value: "client/id"},
		{name: "comma separated", value: "first,second"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value := tc.value
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/nope", nil)
			r.Header.Set("X-Request-Id", value)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status: %d, want 404", rec.Code)
			}
			id := rec.Header().Get("X-Request-Id")
			if tc.reuse {
				if id != value {
					t.Fatalf("ID: %q, want %q", id, value)
				}
			} else {
				if len(id) < 26 || len(id) > 128 || strings.ContainsFunc(id, func(ch rune) bool { return (ch < 'A' || ch > 'Z') && (ch < '2' || ch > '7') }) {
					t.Fatalf("generated ID: %q", id)
				}
			}
			if r.Header.Get("X-Request-Id") != value {
				t.Fatal("incoming header changed")
			}
		})
	}
}

func TestRouter_ProbesReserveRootPaths(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	state := newReadiness(log, pingFunc(func(context.Context) error { return nil }))
	handler := newRouter(log, clientApp, nil, nil, state)
	for _, path := range []string{livenessPath, readinessPath} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions} {
			t.Run(method+" "+path, func(t *testing.T) {
				t.Parallel()
				for _, size := range []int64{0, MaxBodyBytes + 1} {
					rec := httptest.NewRecorder()
					req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
					req.ContentLength = size
					handler.ServeHTTP(rec, req)
					wantStatus, wantBody, wantAllow := http.StatusOK, `{"status":"ok"}`, ""
					if method == http.MethodHead {
						wantBody = ""
					} else if method != http.MethodGet {
						wantStatus, wantBody, wantAllow = http.StatusMethodNotAllowed, "Method Not Allowed\n", "GET, HEAD"
					}
					if rec.Code != wantStatus || rec.Body.String() != wantBody || rec.Header().Get("Allow") != wantAllow {
						t.Fatalf("response: %d %q, Allow %q", rec.Code, rec.Body.String(), rec.Header().Get("Allow"))
					}
					if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("X-Request-Id") == "" {
						t.Fatalf("missing probe headers: %v", rec.Header())
					}
				}
			})
		}
	}
	for _, path := range []string{"/api/livez", "/api/readyz"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status %d, want 404", path, rec.Code)
		}
	}
}
