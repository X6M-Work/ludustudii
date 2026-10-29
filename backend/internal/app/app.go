package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/apotourlyan/ludustudii/internal/infra/env"
	"github.com/apotourlyan/ludustudii/internal/infra/rdb"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
)

func Run() (err error) {
	ev, err := env.Load()
	if err != nil {
		return fmt.Errorf("load environment: %w", err)
	}

	ca, err := spa.Load()
	if err != nil {
		return fmt.Errorf("load client app: %w", err)
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

	router := NewRouter(ca, db)

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
		slog.Info("Shutting down HTTP server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), ev.AppShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	return nil
}
