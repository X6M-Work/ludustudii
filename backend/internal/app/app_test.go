//go:build unit

package app

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/apotourlyan/ludustudii/internal/infra/health"
)

type stopFunc func(context.Context)

func (f stopFunc) MarkStopping(ctx context.Context) { f(ctx) }

type shutdownFunc func(context.Context) error

func (f shutdownFunc) Shutdown(ctx context.Context) error { return f(ctx) }

func TestDrain_MarksStoppingBeforeDelay(t *testing.T) {
	t.Parallel()
	for _, delay := range []time.Duration{0, 40 * time.Millisecond} {
		t.Run(delay.String(), func(t *testing.T) {
			t.Parallel()
			state := health.NewState(slog.New(slog.DiscardHandler), pingFunc(func(context.Context) error { return nil }))
			marked := make(chan time.Time, 1)
			switcher := stopFunc(func(ctx context.Context) {
				state.MarkStopping(ctx)
				marked <- time.Now()
			})
			done := make(chan error, 1)
			go func() { done <- drain(t.Context(), switcher, delay) }()
			markedAt := <-marked
			if state.Check(t.Context()) {
				t.Error("instance still available during drain")
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if time.Since(markedAt) < delay {
				t.Error("drain returned before its delay elapsed after marking stopping")
			}
		})
	}
}

func TestDrain_Canceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	marked := false
	err := drain(ctx, stopFunc(func(context.Context) {
		marked = true
		cancel()
	}), time.Second)
	if !marked {
		t.Error("instance was not marked stopping")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("drain error: %v, want context.Canceled", err)
	}
}

func TestShutdown_GracePeriod(t *testing.T) {
	t.Parallel()
	const timeout = time.Second
	err := shutdown(t.Context(), shutdownFunc(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("shutdown context has no deadline")
		} else if remaining := time.Until(deadline); remaining < timeout/2 || remaining > timeout {
			t.Errorf("shutdown grace period: %v, want approximately %v", remaining, timeout)
		}
		return nil
	}), timeout)
	if err != nil {
		t.Fatal(err)
	}
}

func TestShutdown_ServerError(t *testing.T) {
	t.Parallel()
	want := errors.New("shutdown failed")
	err := shutdown(t.Context(), shutdownFunc(func(context.Context) error { return want }), time.Second)
	if !errors.Is(err, want) {
		t.Fatalf("shutdown error: %v, want %v", err, want)
	}
}
