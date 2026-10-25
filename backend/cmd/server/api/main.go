package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/apotourlyan/ludustudii/internal/database"
	"github.com/apotourlyan/ludustudii/internal/environment"
	"github.com/apotourlyan/ludustudii/internal/services/lcards"
	"github.com/apotourlyan/ludustudii/internal/spa"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Application terminated with error", "error", err)
		os.Exit(1)
	}
}

func run() (err error) {
	vars, err := environment.Load()
	if err != nil {
		return fmt.Errorf("load environment: %w", err)
	}

	clientApp, err := spa.Load()
	if err != nil {
		return fmt.Errorf("load client app: %w", err)
	}

	db, err := database.Connect(database.Options{
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

	e := newRouter(clientApp, db)
	e.Use(middleware.BodyLimit(1 << 20)) // 1 MB limit

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

// newRouter keeps API fallbacks separate from the SPA so unknown API URLs never
// return the client application.
func newRouter(clientApp *spa.ClientApp, db *sql.DB) *echo.Echo {
	e := echo.NewWithConfig(echo.Config{
		Router: echo.NewRouter(echo.RouterConfig{AutoHandleHEAD: true}),
	})
	e.Logger = slog.Default()
	lcards.RegisterRoutes(e, db)
	apiNotFound := func(c *echo.Context) error {
		return c.String(http.StatusNotFound, "404 page not found\n")
	}
	e.RouteNotFound("/api/*", apiNotFound)
	// A child route prevents Echo's terminal :id parameter from consuming slashes.
	e.RouteNotFound("/api/lcards/:id/*", apiNotFound)
	spa.NewHandler(clientApp).RegisterRoutes(e)
	return e
}
