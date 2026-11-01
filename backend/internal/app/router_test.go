//go:build integration

package app

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
	"github.com/apotourlyan/ludustudii/internal/testkit"
	"github.com/labstack/echo/v5"
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
	srv := httptest.NewServer(NewRouter(discardLogger(), clientApp, nil, nil))
	t.Cleanup(srv.Close)
	return srv
}

func TestNewRouter(t *testing.T) {
	t.Parallel()

	t.Run("GET / serves SPA index.html", func(t *testing.T) {
		t.Parallel()
		srv := newRouterServer(t)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method: http.MethodGet,
			Path:   "/",
		})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		if ct := resp.Header.Get(echo.HeaderContentType); !strings.HasPrefix(ct, echo.MIMETextHTML) {
			t.Errorf("Content-Type: got %q, want %s", ct, echo.MIMETextHTML)
		}
		if len(body) == 0 {
			t.Error("expected non-empty HTML body")
		}
	})

	for _, path := range []string{"/api", "/api/", "/api/nope"} {
		t.Run("unknown API path "+path+" returns 404 JSON, not the SPA", func(t *testing.T) {
			t.Parallel()
			srv := newRouterServer(t)

			resp, body := testkit.Do(t, srv, testkit.DoOpts{
				Method: http.MethodGet,
				Path:   path,
			})
			testkit.AssertStatus(t, resp, body, http.StatusNotFound)
			testkit.AssertJSON(t, body, `{"code":"not_found"}`)
		})
	}

	t.Run("panic in handler returns 500 unexpected_error", func(t *testing.T) {
		t.Parallel()
		clientApp, err := spa.Load()
		if err != nil {
			t.Fatal(err)
		}
		e := NewRouter(discardLogger(), clientApp, nil, nil) // keep the stack trace out of test output
		e.GET("/api/panic", func(c *echo.Context) error { panic("boom") })
		srv := httptest.NewServer(e)
		t.Cleanup(srv.Close)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/panic"})
		testkit.AssertStatus(t, resp, body, http.StatusInternalServerError)
		testkit.AssertJSON(t, body, `{"code":"unexpected_error"}`)
	})
}

func TestNewRouter_APIDocs(t *testing.T) {
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
		srv := httptest.NewServer(NewRouter(discardLogger(), clientApp, nil, nil))
		t.Cleanup(srv.Close)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: apidocs.Path})
		testkit.AssertStatus(t, resp, body, http.StatusOK)
		if !bytes.Equal(body, clientApp.IndexFile.Bytes) {
			t.Errorf("GET %s: want the SPA index.html, got %q", apidocs.Path, body)
		}

		for _, path := range []string{apidocs.Path + "/openapi.json", apidocs.Path + "/scalar.js"} {
			resp, body := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: path})
			testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		}
	})

	t.Run("enabled: /docs serves the reference", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(NewRouter(discardLogger(), clientApp, apidocs.NewHandler(docs), nil))
		t.Cleanup(srv.Close)

		for path, want := range map[string]staticfile.File{
			apidocs.Path:                   docs.Page,
			apidocs.Path + "/openapi.json": docs.Spec,
			apidocs.Path + "/scalar.js":    docs.Script,
		} {
			resp, body := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: path})
			testkit.AssertStatus(t, resp, body, http.StatusOK)
			if ct := resp.Header.Get(echo.HeaderContentType); ct != want.ContentType {
				t.Errorf("GET %s Content-Type: got %q, want %q", path, ct, want.ContentType)
			}
			if !bytes.Equal(body, want.Bytes) {
				t.Errorf("GET %s: got %q, want %q", path, body, want.Bytes)
			}
		}

		resp, body := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: "/api/nope"})
		testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		testkit.AssertJSON(t, body, `{"code":"not_found"}`)
	})
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

func TestNewRouter_RequestLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantLevel  string
		wantRoute  string
	}{
		{name: "200 logs INFO", path: "/", wantStatus: http.StatusOK, wantLevel: "INFO", wantRoute: "/*"},
		{name: "404 logs WARN", path: "/api/nope", wantStatus: http.StatusNotFound, wantLevel: "WARN", wantRoute: "/api/*"},
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
			srv := httptest.NewServer(NewRouter(log, clientApp, nil, nil))
			t.Cleanup(srv.Close)

			resp, body := testkit.Do(t, srv, testkit.DoOpts{Method: http.MethodGet, Path: tt.path})
			testkit.AssertStatus(t, resp, body, tt.wantStatus)

			id := resp.Header.Get(echo.HeaderXRequestID)
			if id == "" {
				t.Fatalf("missing %s header", echo.HeaderXRequestID)
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
		srv := httptest.NewServer(NewRouter(log, clientApp, nil, nil))
		t.Cleanup(srv.Close)

		resp, body := testkit.Do(t, srv, testkit.DoOpts{
			Method:  http.MethodGet,
			Path:    "/api/nope",
			Headers: http.Header{echo.HeaderXRequestID: {"client-supplied-id"}},
		})
		testkit.AssertStatus(t, resp, body, http.StatusNotFound)
		testkit.AssertJSON(t, body, `{"code":"not_found"}`)
		srv.Close() // waits for the handler to return, so the record is written

		if got := resp.Header.Get(echo.HeaderXRequestID); got != "client-supplied-id" {
			t.Errorf("header: got %q, want %q", got, "client-supplied-id")
		}
		if !strings.Contains(logs.String(), `"request_id":"client-supplied-id"`) {
			t.Errorf("log missing client request id:\n%s", logs.String())
		}
	})
}
