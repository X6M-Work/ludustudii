package app

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/infra/logging"
	"github.com/apotourlyan/ludustudii/internal/infra/spa"
	"github.com/apotourlyan/ludustudii/internal/services/lcards"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const MaxBodyBytes = 1 << 20

// NewRouter builds the production handler: middleware, API routes, API
// fallbacks, and the SPA.
func NewRouter(log *slog.Logger, clientApp *spa.ClientApp, db *sql.DB) *echo.Echo {
	e := echo.NewWithConfig(echo.Config{
		Router:           echo.NewRouter(echo.RouterConfig{AutoHandleHEAD: true}),
		Binder:           jsonBinder{},
		HTTPErrorHandler: httpErrorHandler,
	})
	e.Logger = log
	e.Use(middleware.Recover())
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		RequestIDHandler: func(c *echo.Context, id string) {
			r := c.Request()
			c.SetRequest(r.WithContext(logging.WithAttrs(r.Context(), slog.String("request_id", id))))
		},
	}))
	e.Use(requestLogger(log))
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

// requestLogger logs one "request" record per request, after the error
// handler has written the response so the status is final.
func requestLogger(log *slog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:    true,
		LogURIPath:   true,
		LogRoutePath: true,
		LogStatus:    true,
		LogLatency:   true,
		HandleError:  true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			lvl := slog.LevelInfo
			switch {
			case v.Status >= http.StatusInternalServerError:
				lvl = slog.LevelError
			case v.Status >= http.StatusBadRequest:
				lvl = slog.LevelWarn
			}
			log.LogAttrs(c.Request().Context(), lvl, "request",
				slog.String("method", v.Method),
				slog.String("route", v.RoutePath),
				slog.String("path", v.URIPath),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
			)
			return nil
		},
	})
}
