//go:build unit

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/app/router"
	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/staticfile"
)

func TestHandler_PathNormalization(t *testing.T) {
	t.Parallel()
	testHandlerResponses(t, []handlerResponseCase{
		{"GET", "/api/lcards/.", 307, "<a href=\"/api/lcards\">Temporary Redirect</a>.\n\n", "", "/api/lcards"},
		{"GET", "/api/lcards/..", 307, "<a href=\"/api\">Temporary Redirect</a>.\n\n", "", "/api"},
		{"GET", "/api//unknown", 307, "<a href=\"/api/unknown\">Temporary Redirect</a>.\n\n", "", "/api/unknown"},
		{"GET", "/docs//", 307, "<a href=\"/docs/\">Temporary Redirect</a>.\n\n", "", "/docs/"},
		{"GET", "/client//x", 307, "<a href=\"/client/x\">Temporary Redirect</a>.\n\n", "", "/client/x"},
		{"GET", "/client/../", 307, "<a href=\"/\">Temporary Redirect</a>.\n\n", "", "/"},
	})
}

func TestHandler_EscapedPaths(t *testing.T) {
	t.Parallel()
	testHandlerResponses(t, []handlerResponseCase{
		{"GET", "/api/lcards/bad%2fx", 400, "{\"code\":\"validation_error\",\"errors\":[{\"field\":\"id\",\"code\":\"invalid_uuid7\"}]}\n", "", ""},
		{"GET", "/%61pi/nope", 404, "{\"code\":\"not_found\"}\n", "", ""},
	})
}

func TestHandler_APIDocsRouting(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	testHandlerResponses(t, []handlerResponseCase{
		{"GET", "/docs/unknown", 200, string(clientApp.IndexFile.Bytes), "", ""},
		{"GET", "/docs/?q=x", 302, "", "", "/docs?q=x"},
		{"HEAD", "/docs/?q=x", 302, "", "", "/docs?q=x"},
	})
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	testHandlerResponses(t, []handlerResponseCase{
		{"POST", "/docs", 405, "method not allowed\n", "GET, HEAD", ""},
		{"POST", "/docs/", 405, "method not allowed\n", "GET, HEAD", ""},
		{"POST", "/docs/openapi.json", 405, "method not allowed\n", "GET, HEAD", ""},
		{"POST", "/docs/scalar.js", 405, "method not allowed\n", "GET, HEAD", ""},
		{"OPTIONS", "/api/lcards", 405, "{\"code\":\"method_not_allowed\"}\n", "GET, HEAD, POST", ""},
		{"PATCH", "/api/lcards/id", 405, "{\"code\":\"method_not_allowed\"}\n", "DELETE, GET, HEAD, PUT", ""},
	})
}

type handlerResponseCase struct {
	method, path          string
	status                int
	body, allow, location string
}

func testHandlerResponses(t *testing.T, cases []handlerResponseCase) {
	t.Helper()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	docs := &apidocs.Docs{Page: staticfile.File{ContentType: "text/html", Bytes: []byte("reference")}}
	log := slog.New(slog.DiscardHandler)
	handler := newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp, docs: docs})
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil))
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

func TestHandler_RequestIDHeader(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	handler := newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp})
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

type pingFunc func(context.Context) error

func (f pingFunc) PingContext(ctx context.Context) error { return f(ctx) }

func probeLogger(t *testing.T, logs *bytes.Buffer) *slog.Logger {
	t.Helper()
	level := new(slog.LevelVar)
	level.Set(slog.LevelDebug)
	log, err := logging.New(logging.Options{Level: level, Format: logging.FormatJSON, Output: logs})
	if err != nil {
		t.Fatal(err)
	}
	return log
}

func TestHandler_HealthEndpointsReserveRootPaths(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	state := health.NewState(log, pingFunc(func(context.Context) error { return nil }))
	handler := newHandler(log, parts{readiness: state, clientApp: clientApp})
	for _, path := range []string{health.LivenessPath, health.ReadinessPath} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions} {
			t.Run(method+" "+path, func(t *testing.T) {
				t.Parallel()
				for _, size := range []int64{0, router.MaxBodyBytes + 1} {
					rec := httptest.NewRecorder()
					req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
					req.ContentLength = size
					handler.ServeHTTP(rec, req)
					wantStatus, wantBody, wantAllow, wantCache := http.StatusOK, `{"status":"ok"}`, "", "no-store"
					if method == http.MethodHead {
						wantBody = ""
					} else if method != http.MethodGet {
						wantStatus, wantBody, wantAllow = http.StatusMethodNotAllowed, "{\"code\":\"method_not_allowed\"}\n", "GET, HEAD"
					}
					if size > router.MaxBodyBytes {
						wantStatus, wantBody, wantAllow, wantCache = http.StatusRequestEntityTooLarge, "{\"code\":\"payload_too_large\"}\n", "", ""
						if method == http.MethodHead {
							wantBody = ""
						}
					}
					if rec.Code != wantStatus || rec.Body.String() != wantBody || rec.Header().Get("Allow") != wantAllow {
						t.Fatalf("response: %d %q, Allow %q", rec.Code, rec.Body.String(), rec.Header().Get("Allow"))
					}
					if rec.Header().Get("Cache-Control") != wantCache || rec.Header().Get("X-Request-Id") == "" || rec.Header().Get("Content-Type") != "application/json" {
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

func TestHandler_HealthRequestLogging(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, path     string
		fail, stopping bool
		status         int
	}{
		{"liveness", health.LivenessPath, false, false, http.StatusOK},
		{"liveness during database outage", health.LivenessPath, true, false, http.StatusOK},
		{"liveness while stopping", health.LivenessPath, false, true, http.StatusOK},
		{"available", health.ReadinessPath, false, false, http.StatusOK},
		{"database unavailable", health.ReadinessPath, true, false, http.StatusServiceUnavailable},
		{"stopping", health.ReadinessPath, false, true, http.StatusServiceUnavailable},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				t.Parallel()
				var logs bytes.Buffer
				log := probeLogger(t, &logs)
				state := health.NewState(log, pingFunc(func(context.Context) error {
					if tc.fail {
						return errors.New("database unreachable")
					}
					return nil
				}))
				if tc.stopping {
					state.MarkStopping(t.Context())
				}
				clientApp, err := spa.Load()
				if err != nil {
					t.Fatal(err)
				}
				handler := newHandler(log, parts{readiness: state, clientApp: clientApp})
				req := httptest.NewRequestWithContext(t.Context(), method, tc.path, nil)
				req.Header.Set("X-Request-Id", "health-request")
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				status := tc.status
				if method == http.MethodPost {
					status = http.StatusMethodNotAllowed
				}
				if rec.Code != status {
					t.Fatalf("status: %d, want %d", rec.Code, status)
				}
				requests, transitions := 0, 0
				for line := range strings.Lines(logs.String()) {
					var record map[string]any
					if err := json.Unmarshal([]byte(line), &record); err != nil {
						t.Fatal(err)
					}
					switch record["msg"] {
					case "request":
						requests++
						if record["level"] != "DEBUG" || record["status"] != float64(status) || record["request_id"] != rec.Header().Get("X-Request-Id") || record["request_id"] != "health-request" {
							t.Fatalf("request record: %v", record)
						}
					case "Readiness unavailable":
						transitions++
						if record["request_id"] != "health-request" {
							t.Fatalf("transition lost request ID: %v", record)
						}
					}
				}
				if requests != 1 {
					t.Fatalf("request records: %d, want 1", requests)
				}
				if tc.path == health.ReadinessPath && tc.fail && method != http.MethodPost && transitions != 1 {
					t.Fatal("missing correlated health transition")
				}
			})
		}
	}
}

func TestHandler_LoggedRouteIsMatchedPattern(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		method, path string
		status       int
		route        string
	}{
		// The ID is invalid, so the handler responds without the database.
		{http.MethodGet, "/api/lcards/not-an-id", 400, "GET /api/lcards/{id}"},
		{http.MethodPatch, "/api/lcards/not-an-id", 405, "/api/"},
		{http.MethodGet, "/api/lcards/not-an-id/x", 404, "/api/"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			clientApp, err := spa.Load()
			if err != nil {
				t.Fatal(err)
			}
			var logs bytes.Buffer
			log, err := logging.New(logging.Options{Format: logging.FormatJSON, Output: &logs})
			if err != nil {
				t.Fatal(err)
			}
			rec := httptest.NewRecorder()
			newHandler(log, parts{readiness: health.NewState(log, nil), clientApp: clientApp}).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil))
			if rec.Code != tc.status {
				t.Fatalf("status: %d, want %d", rec.Code, tc.status)
			}
			var routes []any
			for line := range strings.Lines(logs.String()) {
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["msg"] == "request" {
					routes = append(routes, record["route"])
				}
			}
			if len(routes) != 1 || routes[0] != tc.route {
				t.Fatalf("logged routes: %v, want [%s]", routes, tc.route)
			}
		})
	}
}
