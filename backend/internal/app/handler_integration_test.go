//go:build integration

package app

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/app/router"
	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func TestMain(m *testing.M) { os.Exit(testkit.Run(m)) }

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newHandlerServer(t *testing.T) *httptest.Server {
	t.Helper()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := discardLogger()
	srv := httptest.NewServer(newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHandler_SPAIndex(t *testing.T) {
	t.Parallel()
	srv := newHandlerServer(t)
	resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/"})
	testkit.AssertStatus(t, resp, http.StatusOK)
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type: got %q, want text/html", ct)
	}
	if len(resp.Body) == 0 {
		t.Error("expected non-empty HTML body")
	}
}

func TestHandler_APIDocs(t *testing.T) {
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
		srv := httptest.NewServer(newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp}))
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
		srv := httptest.NewServer(newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp, docs: docs}))
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

func TestHandler_UnknownAPIPaths(t *testing.T) {
	t.Parallel()
	srv := newHandlerServer(t)
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

func TestHandler_FirstRequestIDHeader(t *testing.T) {
	t.Parallel()
	srv := newHandlerServer(t)
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

func TestHandler_SPAErrors(t *testing.T) {
	t.Parallel()
	srv := newHandlerServer(t)
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

// Shared middleware errors use the JSON error shape on client-app paths too,
// unlike errors from the client app's own handler.
func TestHandler_SPABodyLimit(t *testing.T) {
	t.Parallel()
	srv := newHandlerServer(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/"},
		{http.MethodGet, "/client"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			resp := testkit.Do(t, srv, testkit.DoOpts{
				Method: tc.method,
				Path:   tc.path,
				Body:   strings.Repeat("a", router.MaxBodyBytes+1),
			})
			testkit.AssertStatus(t, resp, http.StatusRequestEntityTooLarge)
			if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type: got %q, want application/json", ct)
			}
			testkit.AssertJSON(t, resp.Body, `{"code":"payload_too_large"}`)
		})
	}
}

func TestHandler_HealthBodyLimit(t *testing.T) {
	t.Parallel()
	srv := newHandlerServer(t)
	for _, path := range []string{health.LivenessPath, health.ReadinessPath} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+" "+path, func(t *testing.T) {
				t.Parallel()
				resp := testkit.Do(t, srv, testkit.DoOpts{
					Method: method,
					Path:   path,
					Body:   strings.Repeat("a", router.MaxBodyBytes+1),
				})
				testkit.AssertStatus(t, resp, http.StatusRequestEntityTooLarge)
				if method == http.MethodHead {
					if len(resp.Body) != 0 {
						t.Fatal("HEAD error has a response body")
					}
				} else {
					testkit.AssertJSON(t, resp.Body, `{"code":"payload_too_large"}`)
				}
			})
		}
	}
}

func TestHandler_HealthEndpoints(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"database unavailable", "stopping with healthy database"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			db := testkit.NewDB(t)
			clientApp, err := spa.Load()
			if err != nil {
				t.Fatal(err)
			}
			log := discardLogger()
			state := health.NewState(log, db)
			srv := httptest.NewServer(newHandler(log, parts{db: db, readiness: state, clientApp: clientApp}))
			t.Cleanup(srv.Close)
			probe := func(path string, status int, body string) {
				t.Helper()
				for _, method := range []string{http.MethodGet, http.MethodHead} {
					resp := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: path})
					testkit.AssertStatus(t, resp, status)
					if method == http.MethodGet {
						testkit.AssertJSON(t, resp.Body, body)
					} else if len(resp.Body) != 0 {
						t.Fatal("HEAD has a response body")
					}
					if resp.Header.Get("Cache-Control") != "no-store" {
						t.Fatal("missing Cache-Control: no-store")
					}
				}
			}

			probe(health.LivenessPath, 200, `{"status":"ok"}`)
			probe(health.ReadinessPath, 200, `{"status":"ok"}`)
			for _, path := range []string{health.LivenessPath, health.ReadinessPath} {
				for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
					resp := testkit.Do(t, srv, testkit.DoOpts{Method: method, Path: path})
					testkit.AssertStatus(t, resp, http.StatusMethodNotAllowed)
					testkit.AssertJSON(t, resp.Body, `{"code":"method_not_allowed"}`)
					if resp.Header.Get("Allow") != "GET, HEAD" || resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("X-Request-Id") == "" {
						t.Fatalf("method error headers: %v", resp.Header)
					}
				}
			}
			if scenario == "database unavailable" {
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				state.MarkStopping(t.Context())
				if err := db.PingContext(t.Context()); err != nil {
					t.Fatalf("database must stay healthy while stopping: %v", err)
				}
			}
			probe(health.ReadinessPath, 503, `{"status":"unavailable"}`)
			probe(health.LivenessPath, 200, `{"status":"ok"}`)
		})
	}
}

// syncBuffer guards a bytes.Buffer, since the server goroutine writes logs
// while the test goroutine may read them.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestHandler_RequestLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantLevel  string
		wantRoute  string
		requestID  string
	}{
		{name: "200 logs INFO", path: "/", wantStatus: http.StatusOK, wantLevel: "INFO", wantRoute: "/"},
		{name: "404 logs WARN", path: "/api/nope", wantStatus: http.StatusNotFound, wantLevel: "WARN", wantRoute: "/api/"},
		{name: "oversized request ID replaced", path: "/api/nope", wantStatus: http.StatusNotFound, wantLevel: "WARN", wantRoute: "/api/", requestID: strings.Repeat("a", 129)},
		{name: "invalid request ID replaced", path: "/api/nope", wantStatus: http.StatusNotFound, wantLevel: "WARN", wantRoute: "/api/", requestID: "client/id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clientApp, err := spa.Load()
			if err != nil {
				t.Fatal(err)
			}
			var logs syncBuffer
			log, err := logging.New(logging.Options{Format: logging.FormatJSON, Output: &logs})
			if err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp}))
			t.Cleanup(srv.Close)

			resp := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: tt.path, Headers: http.Header{"X-Request-Id": {tt.requestID}}})
			testkit.AssertStatus(t, resp, tt.wantStatus)

			id := resp.Header.Get("X-Request-Id")
			if id == "" {
				t.Fatalf("missing %s header", "X-Request-Id")
			}

			if len(id) > 128 || (tt.requestID != "" && id == tt.requestID) {
				t.Fatalf("invalid generated request ID: %q", id)
			}

			// Close waits for the handler to return, so the record is written.
			srv.Close()

			var got []map[string]any
			for line := range strings.Lines(logs.String()) {
				var rec map[string]any
				if err := json.Unmarshal([]byte(line), &rec); err != nil {
					t.Fatalf("unmarshal %q: %v", line, err)
				}
				if rec["msg"] == "request" {
					got = append(got, rec)
				}
			}
			if len(got) != 1 {
				t.Fatalf("request records: got %d, want 1\n%s", len(got), logs.String())
			}
			rec := got[0]

			if rec["request_id"] != id {
				t.Errorf("request_id: got %v, want %q", rec["request_id"], id)
			}
			if rec["level"] != tt.wantLevel {
				t.Errorf("level: got %v, want %s", rec["level"], tt.wantLevel)
			}
			if rec["status"] != float64(tt.wantStatus) {
				t.Errorf("status: got %v, want %d", rec["status"], tt.wantStatus)
			}
			if rec["method"] != http.MethodGet {
				t.Errorf("method: got %v, want %s", rec["method"], http.MethodGet)
			}
			if rec["path"] != tt.path {
				t.Errorf("path: got %v, want %s", rec["path"], tt.path)
			}
			if rec["route"] != tt.wantRoute {
				t.Errorf("route: got %v, want %s", rec["route"], tt.wantRoute)
			}
			if _, ok := rec["latency"].(float64); !ok {
				t.Errorf("latency: got %v, want number", rec["latency"])
			}
		})
	}

	t.Run("incoming X-Request-Id is reused", func(t *testing.T) {
		t.Parallel()
		clientApp, err := spa.Load()
		if err != nil {
			t.Fatal(err)
		}
		var logs syncBuffer
		log, err := logging.New(logging.Options{Format: logging.FormatJSON, Output: &logs})
		if err != nil {
			t.Fatal(err)
		}
		srv := httptest.NewServer(newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp}))
		t.Cleanup(srv.Close)

		resp := testkit.Do(t, srv, testkit.DoOpts{
			Method:  http.MethodGet,
			Path:    "/api/nope",
			Headers: http.Header{"X-Request-Id": {"client-supplied-id"}},
		})
		testkit.AssertStatus(t, resp, http.StatusNotFound)
		testkit.AssertJSON(t, resp.Body, `{"code":"not_found"}`)
		srv.Close() // waits for the handler to return, so the record is written

		if got := resp.Header.Get("X-Request-Id"); got != "client-supplied-id" {
			t.Errorf("header: got %q, want %q", got, "client-supplied-id")
		}
		if !strings.Contains(logs.String(), `"request_id":"client-supplied-id"`) {
			t.Errorf("log missing client request id:\n%s", logs.String())
		}
	})
}
