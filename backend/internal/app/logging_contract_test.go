//go:build unit

package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
)

func TestRouter_FinalResponseLogging(t *testing.T) {
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
			onError := errorHandler(log)
			api := httpkit.NewAPI(onError)
			api.HandleFunc("GET /api/panic", func(http.ResponseWriter, *http.Request) error { panic("boom") })
			api.HandleFunc("GET /api/committed", func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusAccepted)
				return errors.New("after commit")
			})
			handler := withMiddleware(log, onError, api)
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

func TestRouter_LoggedRouteIsMatchedPattern(t *testing.T) {
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
			NewRouter(log, clientApp, nil, nil).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil))
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
