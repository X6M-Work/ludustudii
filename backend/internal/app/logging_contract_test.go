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
		{"body limit", "/api/lcards", 413, "WARN", "POST /api/lcards"},
		{"committed error", "/api/committed", 202, "INFO", "GET /api/committed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			handler := NewRouter(log, clientApp, nil, nil)
			rt := handler.(*router)
			rt.mux.HandleFunc("GET /api/panic", func(http.ResponseWriter, *http.Request) { panic("boom") })
			rt.mux.Handle("GET /api/committed", httpkit.Adapt(func(w http.ResponseWriter, _ *http.Request) error {
				w.WriteHeader(http.StatusAccepted)
				return errors.New("after commit")
			}, rt.onError))
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
