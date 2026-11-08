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
	// DrainDelay is how long the server keeps serving after Stopper is told
	// it is stopping, so routing changes can propagate.
	DrainDelay time.Duration
	// ShutdownTimeout is the grace period for in-flight requests after the
	// drain delay.
	ShutdownTimeout time.Duration
}

// Stopper is told that the server is stopping, before the drain delay starts,
// typically to make readiness report unavailable.
type Stopper interface {
	MarkStopping(context.Context)
}

// Run serves srv until it fails or the process receives SIGINT or SIGTERM. On
// a signal it marks stopper stopping, keeps serving for the drain delay, then
// shuts the server down within the shutdown timeout. A second signal during
// draining or shutdown terminates the process immediately.
func Run(srv *http.Server, stopper Stopper, opts Options) error {
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

	shutdownCtx := context.Background()
	if err := drain(shutdownCtx, stopper, opts.DrainDelay); err != nil {
		return err
	}
	return shutdown(shutdownCtx, srv, opts.ShutdownTimeout)
}

// drain marks the instance unready while HTTP is still serving and waits for
// routing changes to propagate before shutdown begins.
func drain(ctx context.Context, stopper Stopper, delay time.Duration) error {
	stopper.MarkStopping(ctx)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("http server drain: %w", ctx.Err())
	}
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
