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

	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

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

func TestRouter_HealthEndpointsReserveRootPaths(t *testing.T) {
	t.Parallel()
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	state := health.NewState(log, pingFunc(func(context.Context) error { return nil }))
	handler := NewRouter(log, clientApp, nil, nil, state)
	for _, path := range []string{health.LivenessPath, health.ReadinessPath} {
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
						wantStatus, wantBody, wantAllow = http.StatusMethodNotAllowed, "{\"code\":\"method_not_allowed\"}\n", "GET, HEAD"
					}
					if rec.Code != wantStatus || rec.Body.String() != wantBody || rec.Header().Get("Allow") != wantAllow {
						t.Fatalf("response: %d %q, Allow %q", rec.Code, rec.Body.String(), rec.Header().Get("Allow"))
					}
					if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("X-Request-Id") == "" || rec.Header().Get("Content-Type") != "application/json" {
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

func TestRouter_HealthRequestLogging(t *testing.T) {
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
				handler := NewRouter(log, clientApp, nil, nil, state)
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
