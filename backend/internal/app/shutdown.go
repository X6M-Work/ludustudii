package app

import (
	"context"
	"fmt"
	"time"
)

type healthDrainer interface {
	StartDraining(context.Context)
}

type shutdownServer interface {
	Shutdown(context.Context) error
}

// shutdown marks the instance unready while HTTP is still serving, allows
// routing changes to propagate, then gives in-flight requests their full grace
// period. Signal handling stays with Run so this sequence is testable directly.
func shutdown(ctx context.Context, state healthDrainer, delay, timeout time.Duration, server shutdownServer) error {
	state.StartDraining(ctx)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return fmt.Errorf("http server drain: %w", ctx.Err())
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}
	return nil
}
