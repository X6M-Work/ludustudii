// Package app is the composition root of the server binary. It loads the
// configuration, builds each part the binary serves and runs them as one
// process. It is the only package that knows every part. Each surface is
// built on its own and mounted on the shared router, and the process lifecycle
// lives in package server, so a part can move to its own binary with a new
// composition root rather than a rewrite.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/apotourlyan/ludustudii/internal/app/server"
	"github.com/apotourlyan/ludustudii/internal/infra/apidocs"
	"github.com/apotourlyan/ludustudii/internal/infra/env"
	"github.com/apotourlyan/ludustudii/internal/infra/health"
	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/rdb"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

// Run loads the environment and sets the default logger, loads the client app
// and, when APP_API_DOCS is set, the API reference, connects to the database
// and serves HTTP until the server fails or the process receives SIGINT or
// SIGTERM (see server.Run). The database is closed before Run returns.
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

	clientApp, err := spa.Load()
	if err != nil {
		return fmt.Errorf("load client app: %w", err)
	}
	var docs *apidocs.Docs
	if ev.AppAPIDocs {
		// The API keeps running without its reference when only the Scalar
		// bundle is missing, as in a checkout without frontend dependencies.
		// A binary built without the apidocs tag fails to start instead, since
		// enabling the reference for it is a deployment mistake.
		docs, err = apidocs.Load()
		switch {
		case errors.Is(err, apidocs.ErrScriptNotEmbedded):
			slog.Warn("API reference disabled", "error", err)
		case err != nil:
			return fmt.Errorf("load api docs: %w", err)
		default:
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

	readiness := health.NewState(log, db)
	handler := newHandler(log, parts{
		db:        db,
		readiness: readiness,
		clientApp: clientApp,
		docs:      docs,
	})

	slog.Info("Starting HTTP server", "port", ev.AppPort)
	return server.Run(&http.Server{
		Addr:         net.JoinHostPort("", ev.AppPort),
		Handler:      handler,
		ReadTimeout:  ev.AppReadTimeout,
		WriteTimeout: ev.AppWriteTimeout,
		IdleTimeout:  ev.AppIdleTimeout,
	}, server.Options{
		OnDrain:         readiness.MarkStopping,
		DrainDelay:      ev.AppShutdownDrainDelay,
		ShutdownTimeout: ev.AppShutdownTimeout,
	})
}
