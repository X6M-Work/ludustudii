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

type drainFunc func(context.Context)

func (f drainFunc) StartDraining(ctx context.Context) { f(ctx) }

type shutdownFunc func(context.Context) error

func (f shutdownFunc) Shutdown(ctx context.Context) error { return f(ctx) }

func TestShutdown_DrainsBeforeShutdown(t *testing.T) {
	t.Parallel()
	const delay = 40 * time.Millisecond
	const timeout = time.Second
	state := health.NewState(slog.New(slog.DiscardHandler), pingFunc(func(context.Context) error { return nil }))
	marked := make(chan struct{})
	var markedAt time.Time
	switcher := drainFunc(func(ctx context.Context) {
		state.StartDraining(ctx)
		markedAt = time.Now()
		close(marked)
	})
	done := make(chan error, 1)
	go func() {
		done <- shutdown(t.Context(), switcher, delay, timeout, shutdownFunc(func(ctx context.Context) error {
			if state.Check(ctx) || time.Since(markedAt) < delay {
				t.Error("server shutdown started before draining finished")
			}
			if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) < timeout/2 {
				t.Error("shutdown did not receive its full grace period after draining")
			}
			return nil
		}))
	}()
	<-marked
	if state.Check(t.Context()) {
		t.Fatal("instance still available during drain")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestShutdown_ServerError(t *testing.T) {
	t.Parallel()
	want := errors.New("shutdown failed")
	marked := false
	err := shutdown(t.Context(), drainFunc(func(context.Context) { marked = true }), 0, time.Second,
		shutdownFunc(func(context.Context) error {
			if !marked {
				t.Error("server shutdown before instance marked unready")
			}
			return want
		}))
	if !errors.Is(err, want) {
		t.Fatalf("shutdown error: %v, want %v", err, want)
	}
}
