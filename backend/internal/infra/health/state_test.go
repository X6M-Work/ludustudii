//go:build unit

package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func stateLogger(t *testing.T, logs *bytes.Buffer) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestState_RequestCanceledDuringPing(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	started := make(chan struct{})
	state := NewState(stateLogger(t, &logs), pingFunc(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan bool, 1)
	go func() { result <- state.Check(ctx) }()
	<-started
	cancel()
	if <-result {
		t.Error("canceled probe reported available")
	}
	if state.currentState() != available {
		t.Error("canceled probe changed availability")
	}
	if logs.Len() != 0 {
		t.Errorf("canceled probe logged a transition: %s", logs.String())
	}
}

func TestState_DrainingDuringPing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		pingError error
	}{
		{name: "successful ping"},
		{name: "failed ping", pingError: errors.New("database unreachable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			started, release := make(chan struct{}), make(chan struct{})
			state := NewState(stateLogger(t, &logs), pingFunc(func(context.Context) error {
				close(started)
				<-release
				return tc.pingError
			}))
			result := make(chan bool, 1)
			go func() { result <- state.Check(t.Context()) }()
			<-started
			state.StartDraining(t.Context())
			close(release)
			if <-result || state.Check(t.Context()) {
				t.Fatal("a completed ping restored readiness during drain")
			}
			if strings.Contains(logs.String(), "Readiness unavailable") || strings.Contains(logs.String(), "Readiness recovered") {
				t.Fatalf("late ping logged a transition after draining started: %s", logs.String())
			}
		})
	}
}

func TestState_ConcurrentFailuresLogOnce(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	state := NewState(stateLogger(t, &logs), pingFunc(func(context.Context) error {
		return errors.New("unreachable")
	}))
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if state.Check(t.Context()) {
				t.Error("failed ping reported available")
			}
		})
	}
	wg.Wait()
	if got := strings.Count(logs.String(), `"msg":"Readiness unavailable"`); got != 1 {
		t.Fatalf("unavailable transitions: %d, want 1", got)
	}
}

func TestState_LoggingTransitions(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	fail := false
	state := NewState(stateLogger(t, &logs), pingFunc(func(context.Context) error {
		if fail {
			return errors.New("database unreachable")
		}
		return nil
	}))
	check := func(want bool) {
		t.Helper()
		if got := state.Check(t.Context()); got != want {
			t.Fatalf("available: %v, want %v", got, want)
		}
	}
	check(true)
	fail = true
	for range 5 {
		check(false)
	}
	fail = false
	for range 3 {
		check(true)
	}
	state.StartDraining(t.Context())
	state.StartDraining(t.Context())
	for range 3 {
		check(false)
	}
	counts := map[string]int{}
	for line := range strings.Lines(logs.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		msg, _ := record["msg"].(string)
		counts[msg]++
		level := "INFO"
		if msg == "Readiness unavailable" {
			level = "WARN"
			if record["error"] != "database unreachable" {
				t.Fatalf("missing transition reason: %v", record)
			}
		}
		if record["level"] != level {
			t.Errorf("record: %v, want level %s", record, level)
		}
	}
	for _, msg := range []string{"Readiness unavailable", "Readiness recovered", "Readiness draining started"} {
		if counts[msg] != 1 {
			t.Errorf("%s records: %d, want 1", msg, counts[msg])
		}
	}
}
