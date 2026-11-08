//go:build unit

package server

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/apotourlyan/ludustudii/internal/infra/health"
)

type pingFunc func(context.Context) error

func (f pingFunc) PingContext(ctx context.Context) error { return f(ctx) }

type shutdownFunc func(context.Context) error

func (f shutdownFunc) Shutdown(ctx context.Context) error { return f(ctx) }

func TestDrain_CallsOnDrainBeforeDelay(t *testing.T) {
	t.Parallel()
	for _, delay := range []time.Duration{0, 40 * time.Millisecond} {
		t.Run(delay.String(), func(t *testing.T) {
			t.Parallel()
			state := health.NewState(slog.New(slog.DiscardHandler), pingFunc(func(context.Context) error { return nil }))
			marked := make(chan time.Time, 1)
			onDrain := func(ctx context.Context) {
				state.MarkStopping(ctx)
				marked <- time.Now()
			}
			done := make(chan struct{})
			go func() {
				drain(onDrain, delay)
				close(done)
			}()
			markedAt := <-marked
			if state.Check(t.Context()) {
				t.Error("instance still available during drain")
			}
			<-done
			if time.Since(markedAt) < delay {
				t.Error("drain returned before its delay elapsed after OnDrain")
			}
		})
	}
}

func TestDrain_NilOnDrainWaitsForDelay(t *testing.T) {
	t.Parallel()
	const delay = 40 * time.Millisecond
	start := time.Now()
	drain(nil, delay)
	if elapsed := time.Since(start); elapsed < delay {
		t.Errorf("drain returned after %v, want at least %v", elapsed, delay)
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
