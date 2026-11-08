//go:build unit

package router

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

// testSurface serves GET on each of its paths with 200.
type testSurface struct {
	paths []string
}

func (s testSurface) Mount(mux *http.ServeMux) {
	for _, path := range s.paths {
		mux.HandleFunc("GET "+path, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	}
}

// quietSurface is a testSurface whose paths are quiet.
type quietSurface struct {
	testSurface
}

func (s quietSurface) QuietPaths() []string { return s.paths }

func requestRecords(t *testing.T, logs string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for line := range strings.Lines(logs) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record["msg"] == "request" {
			records = append(records, record)
		}
	}
	return records
}

func TestNew_MountsSurfaces(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path, level, route string
		status             int
	}{
		{"/a", "INFO", "GET /a", http.StatusOK},
		{"/b", "INFO", "GET /b", http.StatusOK},
		{"/quiet", "DEBUG", "GET /quiet", http.StatusOK},
		// No surface claims the path, so the root mux answers.
		{"/none", "WARN", "", http.StatusNotFound},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			level := new(slog.LevelVar)
			level.Set(slog.LevelDebug)
			log, err := logging.New(logging.Options{Level: level, Format: logging.FormatJSON, Output: &logs})
			if err != nil {
				t.Fatal(err)
			}
			handler := New(log, ErrorHandler(log),
				testSurface{paths: []string{"/a"}},
				testSurface{paths: []string{"/b"}},
				quietSurface{testSurface{paths: []string{"/quiet"}}},
			)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil))
			if rec.Code != tc.status {
				t.Fatalf("status: %d, want %d", rec.Code, tc.status)
			}
			if rec.Header().Get("X-Request-Id") == "" {
				t.Fatal("middleware did not run")
			}
			records := requestRecords(t, logs.String())
			if len(records) != 1 || records[0]["level"] != tc.level || records[0]["route"] != tc.route {
				t.Fatalf("request records: %v, want one at %s for route %q", records, tc.level, tc.route)
			}
		})
	}
}

func TestNew_ConflictingSurfacesPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("conflicting surfaces mounted without a panic")
		}
	}()
	log := slog.New(slog.DiscardHandler)
	New(log, ErrorHandler(log), testSurface{paths: []string{"/a"}}, testSurface{paths: []string{"/a"}})
}

func TestMiddleware_FinalResponseLogging(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, path   string
		status       int
		level, route string
	}{
		{"panic", "/api/panic", 500, "ERROR", "GET /api/panic"},
		// BodyLimit rejects a known oversized body before the mux matches a route.
		{"body limit", "/api/committed", 413, "WARN", ""},
		{"committed error", "/api/committed", 202, "INFO", "GET /api/committed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			log, err := logging.New(logging.Options{Format: logging.FormatJSON, Output: &logs})
			if err != nil {
				t.Fatal(err)
			}
			onError := ErrorHandler(log)
			api := httpkit.NewAPI(onError)
			api.HandleFunc("GET /api/panic", func(http.ResponseWriter, *http.Request) error { panic("boom") })
			api.HandleFunc("GET /api/committed", func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusAccepted)
				return errors.New("after commit")
			})
			handler := withMiddleware(log, onError, nil, api)
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.path, nil)
			if tc.name == "body limit" {
				r.Method = http.MethodPost
				r.ContentLength = MaxBodyBytes + 1
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			if rec.Code != tc.status {
				t.Fatalf("status: %d, want %d", rec.Code, tc.status)
			}
			count := 0
			for line := range strings.Lines(logs.String()) {
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["msg"] != "request" {
					continue
				}
				count++
				if record["status"] != float64(tc.status) || record["level"] != tc.level || record["route"] != tc.route {
					t.Fatalf("record: %+v", record)
				}
				if record["request_id"] != rec.Header().Get("X-Request-Id") {
					t.Fatal("request ID missing from context")
				}
			}
			if count != 1 {
				t.Fatalf("request records: %d, want 1", count)
			}
			if tc.name == "committed error" && strings.Contains(logs.String(), "Unhandled error") {
				t.Fatal("committed error was logged")
			}
			if rec.Body.Len() != 0 && tc.name == "committed error" {
				t.Fatal("committed error changed body")
			}
		})
	}
}
