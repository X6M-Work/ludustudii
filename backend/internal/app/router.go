package app

import (
	"database/sql"
	"log/slog"

	"github.com/apotourlyan/ludustudii/internal/services/lcards"
	"github.com/apotourlyan/ludustudii/internal/spa"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const MaxBodyBytes = 1 << 20

// NewRouter builds the production handler: middleware, API routes, API
// fallbacks, and the SPA. Shared by main and the API tests.
func NewRouter(clientApp *spa.ClientApp, db *sql.DB) *echo.Echo {
	e := echo.NewWithConfig(echo.Config{
		Router:           echo.NewRouter(echo.RouterConfig{AutoHandleHEAD: true}),
		Binder:           jsonBinder{},
		HTTPErrorHandler: httpErrorHandler,
	})
	e.Logger = slog.Default()
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit(MaxBodyBytes))

	lcards.RegisterRoutes(e, db)

	// Keeps API fallbacks separate from the SPA so unknown API URLs never
	// return the client application.
	e.RouteNotFound("/api/*", func(c *echo.Context) error {
		return echo.ErrNotFound
	})

	spa.NewHandler(clientApp).RegisterRoutes(e)
	return e
}
