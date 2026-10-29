package main

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

	"github.com/apotourlyan/ludustudii/internal/app"
	"github.com/apotourlyan/ludustudii/internal/infra/env"
	"github.com/apotourlyan/ludustudii/internal/infra/rdb"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Application terminated with error", "error", err)
		os.Exit(1)
	}
}

func run() (err error) {
	vars, err := env.Load()
	if err != nil {
		return fmt.Errorf("load environment: %w", err)
	}

	clientApp, err := spa.Load()
	if err != nil {
		return fmt.Errorf("load client app: %w", err)
	}

	db, err := rdb.Connect(rdb.Options{
		ConnectionString: vars.DBConnectionString,
		MaxOpenConns:     vars.DBMaxOpenConns,
		MaxIdleConns:     vars.DBMaxIdleConns,
		ConnMaxLifetime:  vars.DBConnMaxLifetime,
		ConnMaxIdleTime:  vars.DBConnMaxIdleTime,
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close database: %w", closeErr))
		}
	}()

	e := app.NewRouter(clientApp, db)

	server := &http.Server{
		Addr:         net.JoinHostPort("", vars.AppPort),
		Handler:      e,
		ReadTimeout:  vars.AppReadTimeout,
		WriteTimeout: vars.AppWriteTimeout,
		IdleTimeout:  vars.AppIdleTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("Starting HTTP server", "port", vars.AppPort)
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
		slog.Info("Shutting down server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), vars.AppShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	return nil
}
