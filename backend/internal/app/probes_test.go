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
	"sync"
	"testing"
	"time"

	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

type pingFunc func(context.Context) error

func (f pingFunc) PingContext(ctx context.Context) error { return f(ctx) }

func TestProbes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		path     string
		ping     string
		draining bool
		status   int
		body     string
		calls    int
	}{
		{"liveness ignores database failure", livenessPath, "fail", false, 200, `{"status":"ok"}`, 0},
		{"liveness during drain", livenessPath, "fail", true, 200, `{"status":"ok"}`, 0},
		{"ready", readinessPath, "ok", false, 200, `{"status":"ok"}`, 1},
		{"ping failed", readinessPath, "fail", false, 503, `{"status":"unavailable"}`, 1},
		{"ping timed out", readinessPath, "timeout", false, 503, `{"status":"unavailable"}`, 1},
		{"draining", readinessPath, "ok", true, 503, `{"status":"unavailable"}`, 0},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				t.Parallel()
				calls := 0
				state := newReadiness(slog.New(slog.DiscardHandler), pingFunc(func(ctx context.Context) error {
					calls++
					deadline, ok := ctx.Deadline()
					if !ok || time.Until(deadline) > time.Second {
						t.Error("ping must have a deadline within one second")
					}
					switch tc.ping {
					case "fail":
						return errors.New("private database error")
					case "timeout":
						<-ctx.Done()
						return ctx.Err()
					default:
						return nil
					}
				}))
				if tc.draining {
					state.markNotReady(t.Context())
				}
				mux := http.NewServeMux()
				state.registerProbes(mux)
				rec := httptest.NewRecorder()
				start := time.Now()
				mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, tc.path, nil))
				if tc.ping == "timeout" && time.Since(start) > 2*time.Second {
					t.Error("ping timeout took too long")
				}
				wantBody := tc.body
				if method == http.MethodHead {
					wantBody = ""
				}
				if rec.Code != tc.status || rec.Body.String() != wantBody {
					t.Fatalf("response: %d %q, want %d %q", rec.Code, rec.Body.String(), tc.status, wantBody)
				}
				if calls != tc.calls {
					t.Errorf("ping calls: %d, want %d", calls, tc.calls)
				}
				if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("headers: %v", rec.Header())
				}
			})
		}
	}
}

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

func TestProbes_LoggingTransitions(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	log := probeLogger(t, &logs)
	fail := false
	state := newReadiness(log, pingFunc(func(context.Context) error {
		if fail {
			return errors.New("database unreachable")
		}
		return nil
	}))
	clientApp, err := spa.Load()
	if err != nil {
		t.Fatal(err)
	}
	handler := newRouter(log, clientApp, nil, nil, state)
	requests := 0
	probe := func(path string, status int) {
		t.Helper()
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil))
			if rec.Code != status {
				t.Fatalf("status: %d, want %d", rec.Code, status)
			}
			requests++
		}
	}
	probe(livenessPath, 200)
	probe(readinessPath, 200)
	fail = true
	for range 5 {
		probe(readinessPath, 503)
	}
	fail = false
	for range 3 {
		probe(readinessPath, 200)
	}
	state.markNotReady(t.Context())
	state.markNotReady(t.Context())
	for range 3 {
		probe(readinessPath, 503)
	}
	probe(livenessPath, 200)

	counts := map[string]int{}
	for line := range strings.Lines(logs.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		msg, _ := record["msg"].(string)
		counts[msg]++
		wantLevel := "INFO"
		switch msg {
		case "request":
			wantLevel = "DEBUG"
			if record["request_id"] == "" || record["request_id"] == nil {
				t.Error("request ID missing")
			}
		case "Readiness unavailable":
			wantLevel = "WARN"
			if record["error"] != "database unreachable" || record["request_id"] == nil {
				t.Fatalf("unavailable transition missing reason or request ID: %v", record)
			}
		}
		if record["level"] != wantLevel {
			t.Errorf("record: %v, want level %s", record, wantLevel)
		}
	}
	for msg, want := range map[string]int{"request": requests, "Readiness unavailable": 1, "Readiness recovered": 1, "Readiness draining started": 1} {
		if counts[msg] != want {
			t.Errorf("%s records: %d, want %d", msg, counts[msg], want)
		}
	}
}

func TestReadiness_DrainingDuringPing(t *testing.T) {
	t.Parallel()
	started, release := make(chan struct{}), make(chan struct{})
	state := newReadiness(slog.New(slog.DiscardHandler), pingFunc(func(context.Context) error {
		close(started)
		<-release
		return nil
	}))
	result := make(chan bool, 1)
	go func() { result <- state.ready(t.Context()) }()
	<-started
	state.markNotReady(t.Context())
	close(release)
	if <-result || state.ready(t.Context()) {
		t.Fatal("a completed ping restored readiness during drain")
	}
}

func TestReadiness_ConcurrentFailuresLogOnce(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	state := newReadiness(probeLogger(t, &logs), pingFunc(func(context.Context) error {
		return errors.New("unreachable")
	}))
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if state.ready(t.Context()) {
				t.Error("failed ping reported ready")
			}
		})
	}
	wg.Wait()
	if got := strings.Count(logs.String(), `"msg":"Readiness unavailable"`); got != 1 {
		t.Fatalf("unavailable transitions: %d, want 1", got)
	}
}
