//go:build integration

package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newRouterServer(t *testing.T) *httptest.Server {
	t.Helper()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := discardLogger()
	srv := httptest.NewServer(NewRouter(log, clientApp, nil, nil, health.NewState(log, nil)))
	t.Cleanup(srv.Close)
	return srv
}

func TestRouter_SPAIndex(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t)
	resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/"})
	testkit.AssertStatus(t, resp, http.StatusOK)
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type: got %q, want text/html", ct)
	}
	if len(resp.Body) == 0 {
		t.Error("expected non-empty HTML body")
	}
}

func TestMiddleware_PanicResponse(t *testing.T) {
	t.Parallel()
	log := discardLogger()
	onError := errorHandler(log)
	api := httpkit.NewAPI(onError)
	api.HandleFunc("GET /api/panic", func(http.ResponseWriter, *http.Request) error { panic("boom") })
	srv := httptest.NewServer(withMiddleware(log, onError, api))
	t.Cleanup(srv.Close)
	resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/panic"})
	testkit.AssertStatus(t, resp, http.StatusInternalServerError)
	testkit.AssertJSON(t, resp.Body, `{"code":"unexpected_error"}`)
}

func TestRouter_APIDocs(t *testing.T) {
	t.Parallel()

	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	docs := &apidocs.Docs{
		Page:   staticfile.File{ContentType: "text/html; charset=utf-8", ETag: `"page"`, Bytes: []byte("<!doctype html><title>docs</title>")},
		Spec:   staticfile.File{ContentType: "application/json", ETag: `"spec"`, Bytes: []byte(`{"openapi":"3.0.3"}`)},
		Script: staticfile.File{ContentType: "text/javascript; charset=utf-8", ETag: `"script"`, Bytes: []byte("console.log(1)")},
	}

	t.Run("disabled: /docs is left to the SPA", func(t *testing.T) {
		t.Parallel()
		log := discardLogger()
		srv := httptest.NewServer(NewRouter(log, clientApp, nil, nil, health.NewState(log, nil)))
		t.Cleanup(srv.Close)

		resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: apidocs.Path})
		testkit.AssertStatus(t, resp, http.StatusOK)
		if !bytes.Equal(resp.Body, clientApp.IndexFile.Bytes) {
			t.Errorf("GET %s: want the SPA index.html, got %q", apidocs.Path, resp.Body)
		}

		for _, path := range []string{apidocs.Path + "/openapi.json", apidocs.Path + "/scalar.js"} {
			resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: path})
			testkit.AssertStatus(t, resp, http.StatusNotFound)
		}
	})

	t.Run("enabled: /docs serves the reference", func(t *testing.T) {
		t.Parallel()
		log := discardLogger()
		srv := httptest.NewServer(NewRouter(log, clientApp, apidocs.NewHandler(docs), nil, health.NewState(log, nil)))
		t.Cleanup(srv.Close)

		for path, want := range map[string]staticfile.File{
			apidocs.Path:                   docs.Page,
			apidocs.Path + "/openapi.json": docs.Spec,
			apidocs.Path + "/scalar.js":    docs.Script,
		} {
			resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: path})
			testkit.AssertStatus(t, resp, http.StatusOK)
			if ct := resp.Header.Get("Content-Type"); ct != want.ContentType {
				t.Errorf("GET %s Content-Type: got %q, want %q", path, ct, want.ContentType)
			}
			if !bytes.Equal(resp.Body, want.Bytes) {
				t.Errorf("GET %s: got %q, want %q", path, resp.Body, want.Bytes)
			}
		}

		resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/nope"})
		testkit.AssertStatus(t, resp, http.StatusNotFound)
		testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
	})
}

func TestRouter_UnknownAPIPaths(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t)
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
		for _, path := range []string{"/api", "/api/", "/api/nope", "/api/unknown", "/api/lcards/one/x"} {
			t.Run(method+" "+path, func(t *testing.T) {
				t.Parallel()
				resp := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: path})
				testkit.AssertStatus(t, resp, http.StatusNotFound)
				if method == http.MethodHead {
					if len(resp.Body) != 0 {
						t.Fatal("HEAD error has body")
					}
				} else {
					testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
				}
			})
		}
	}
}

func TestRouter_FirstRequestIDHeader(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t)
	for _, tc := range []struct {
		name   string
		values []string
		reuse  bool
	}{
		{"valid first value", []string{"first", "second"}, true},
		{"empty first value", []string{"", "second"}, false},
		{"invalid first value", []string{"invalid/id", "second"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/nope", Headers: http.Header{"X-Request-Id": tc.values}})
			testkit.AssertStatus(t, resp, http.StatusNotFound)
			testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
			id := resp.Header.Get("X-Request-Id")
			if tc.reuse {
				if id != tc.values[0] {
					t.Fatalf("request ID: %q, want %q", id, tc.values[0])
				}
			} else if id == "" || id == tc.values[0] || id == tc.values[1] {
				t.Fatalf("generated ID: %q", id)
			}
		})
	}
}

func TestRouter_SPAErrors(t *testing.T) {
	t.Parallel()
	srv := newRouterServer(t)
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodPost, "/client", "method not allowed\n", http.StatusMethodNotAllowed},
		{http.MethodGet, "/missing.js", "404 page not found\n", http.StatusNotFound},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			resp := testkit.Do(t, srv, testkit.DoOpts{Method: tc.method, Path: tc.path})
			testkit.AssertStatus(t, resp, tc.status)
			if string(resp.Body) != tc.body {
				t.Fatalf("body: %q, want %q", resp.Body, tc.body)
			}
		})
	}
}
