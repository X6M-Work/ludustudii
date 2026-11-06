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

func TestProbes_LoggingTransitions(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	log := probeLogger(t, &logs)
	fail := false
	state := health.NewState(log, pingFunc(func(context.Context) error {
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
	probe(health.LivenessPath, 200)
	probe(health.ReadinessPath, 200)
	fail = true
	for range 5 {
		probe(health.ReadinessPath, 503)
	}
	fail = false
	for range 3 {
		probe(health.ReadinessPath, 200)
	}
	state.StartDraining(t.Context())
	state.StartDraining(t.Context())
	for range 3 {
		probe(health.ReadinessPath, 503)
	}
	probe(health.LivenessPath, 200)

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
