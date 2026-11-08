// Package app wires the server together: configuration, logging, database,
// router and HTTP server lifecycle.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/env"
	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/rdb"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

// Run loads the environment, sets the default logger, loads the embedded
// client app and, when APP_API_DOCS is set, the API reference, connects to
// the database and serves HTTP until the server fails or the process receives
// SIGINT or SIGTERM. On a signal it shuts the server down gracefully within
// the configured shutdown timeout after the drain delay; a second signal
// during draining or shutdown terminates the process immediately. The database is closed before Run returns.
func Run() (err error) {
	ev, err := env.Load()
	if err != nil {
		return fmt.Errorf("load environment: %w", err)
	}

	log, err := logging.New(logging.Options{
		Level:  ev.LogLevel,
		Format: ev.LogFormat,
		Output: os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	slog.SetDefault(log)
	slog.LogAttrs(context.Background(), slog.LevelInfo, "Environment variables loaded", ev.LogAttrs()...)

	ca, err := spa.Load()
	if err != nil {
		return fmt.Errorf("load client app: %w", err)
	}

	var adh *apidocs.Handler
	if ev.AppAPIDocs {
		// The API keeps running without its reference when only the Scalar
		// bundle is missing, as in a checkout without frontend dependencies.
		// A binary built without the apidocs tag fails to start instead, since
		// enabling the reference for it is a deployment mistake.
		d, err := apidocs.Load()
		switch {
		case errors.Is(err, apidocs.ErrScriptNotEmbedded):
			slog.Warn("API reference disabled", "error", err)
		case err != nil:
			return fmt.Errorf("load api docs: %w", err)
		default:
			adh = apidocs.NewHandler(d)
			slog.Info("API reference enabled", "path", apidocs.Path)
		}
	}

	db, err := rdb.Connect(rdb.Options{
		ConnectionString: ev.DBConnectionString,
		MaxOpenConns:     ev.DBMaxOpenConns,
		MaxIdleConns:     ev.DBMaxIdleConns,
		ConnMaxLifetime:  ev.DBConnMaxLifetime,
		ConnMaxIdleTime:  ev.DBConnMaxIdleTime,
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close database: %w", closeErr))
		}
	}()

	state := health.NewState(log, db)
	router := NewRouter(log, ca, adh, db, state)

	server := &http.Server{
		Addr:         net.JoinHostPort("", ev.AppPort),
		Handler:      router,
		ReadTimeout:  ev.AppReadTimeout,
		WriteTimeout: ev.AppWriteTimeout,
		IdleTimeout:  ev.AppIdleTimeout,
	}

	slog.Info("Starting HTTP server", "port", ev.AppPort)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve(ctx, stop, server, state, ev.AppShutdownDrainDelay, ev.AppShutdownTimeout)
}

type server interface {
	shutter
	ListenAndServe() error
}

func serve(ctx context.Context, stop func(), srv server, state stopper, drainDelay, shutdownTimeout time.Duration) error {
	errChan := make(chan error, 1)
	served := make(chan struct{})
	go func() {
		defer close(served)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server error: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		if stop != nil {
			stop()
		}
		slog.Info("Shutting down HTTP server...")
	}

	shutdownCtx := context.Background()
	drainErr := drain(shutdownCtx, state, drainDelay)
	shutdownErr := shutdown(shutdownCtx, srv, shutdownTimeout)

	<-served

	var listenErr error
	select {
	case listenErr = <-errChan:
	default:
	}

	return errors.Join(drainErr, shutdownErr, listenErr)
}

type stopper interface {
	MarkStopping(context.Context)
}

// drain marks the instance unready while HTTP is still serving and waits for
// routing changes to propagate before shutdown begins.
func drain(ctx context.Context, state stopper, delay time.Duration) error {
	state.MarkStopping(ctx)
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
