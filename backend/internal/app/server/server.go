// Package server runs an HTTP server process: it serves until the server fails
// or the process receives SIGINT or SIGTERM, then drains and shuts down
// gracefully. It knows nothing about what the server's handler serves.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// Options configures how Run stops the server.
type Options struct {
	// OnDrain is called when the first SIGINT or SIGTERM arrives, while the
	// server is still serving. Use it to start failing readiness so load
	// balancers stop sending new traffic here before shutdown begins. It may
	// be nil.
	OnDrain func(context.Context)
	// DrainDelay is how long the server keeps serving after OnDrain, so load
	// balancers can notice the failed readiness and route traffic elsewhere.
	DrainDelay time.Duration
	// ShutdownTimeout is the grace period for in-flight requests after the
	// drain delay.
	ShutdownTimeout time.Duration
}

// Run serves srv until it fails or the process receives SIGINT or SIGTERM. On
// a signal it calls OnDrain, keeps serving for the drain delay, then
// shuts the server down within the shutdown timeout. A second signal during
// draining or shutdown terminates the process immediately.
func Run(srv *http.Server, opts Options) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server error: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		// Restore default signal handling, so a second signal terminates.
		stop()
		slog.Info("Shutting down HTTP server...")
	}

	drain(opts.OnDrain, opts.DrainDelay)
	return shutdown(context.Background(), srv, opts.ShutdownTimeout)
}

// drain lets load balancers move traffic away before shutdown: onDrain makes
// readiness fail, and the server keeps serving for the delay.
func drain(onDrain func(context.Context), delay time.Duration) {
	if onDrain != nil {
		onDrain(context.Background())
	}
	time.Sleep(delay)
}

type shutter interface {
	Shutdown(context.Context) error
}

// shutdown gives in-flight requests their configured grace period.
func shutdown(ctx context.Context, server shutter, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}
	return nil
}
