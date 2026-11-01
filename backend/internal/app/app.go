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

	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/env"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/rdb"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

// Run loads the environment, sets the default logger, loads the embedded
// client app and, when APP_API_DOCS is set, the API reference, connects to
// the database and serves HTTP until the server fails or the process receives
// SIGINT or SIGTERM. On a signal it shuts the server down gracefully within
// the configured shutdown timeout; a second signal during shutdown terminates
// the process immediately. The database is closed before Run returns.
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

	var apiDocs *apidocs.Handler
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
			apiDocs = apidocs.NewHandler(d)
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

	router := NewRouter(log, ca, apiDocs, db)

	server := &http.Server{
		Addr:         net.JoinHostPort("", ev.AppPort),
		Handler:      router,
		ReadTimeout:  ev.AppReadTimeout,
		WriteTimeout: ev.AppWriteTimeout,
		IdleTimeout:  ev.AppIdleTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("Starting HTTP server", "port", ev.AppPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server error: %w", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		stop()
		slog.Info("Shutting down HTTP server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ev.AppShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	return nil
}
