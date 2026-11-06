//go:build integration

package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/testkit"
)

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

func TestRouter_RequestLogging(t *testing.T) {
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
			srv := httptest.NewServer(NewRouter(log, clientApp, nil, nil, health.NewState(log, nil)))
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
		srv := httptest.NewServer(NewRouter(log, clientApp, nil, nil, health.NewState(log, nil)))
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
