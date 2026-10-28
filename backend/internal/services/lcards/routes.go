package lcards

import (
	"database/sql"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/create"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/delete"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/get"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/list"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/update"
	"github.com/labstack/echo/v5"
)

func RegisterRoutes(e *echo.Echo, db *sql.DB) {
	l := list.NewHandler(list.NewStore(db))
	g := get.NewHandler(get.NewStore(db))
	c := create.NewHandler(create.NewStore(db))
	u := update.NewHandler(update.NewStore(db))
	d := delete.NewHandler(delete.NewStore(db))

	e.GET("/api/lcards", l.Handle)
	e.GET("/api/lcards/:id", g.Handle)
	e.POST("/api/lcards", c.Handle)
	e.PUT("/api/lcards/:id", u.Handle)
	e.DELETE("/api/lcards/:id", d.Handle)

	methodNotAllowed := func(allow string) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set(echo.HeaderAllow, allow)
			return echo.ErrMethodNotAllowed
		}
	}
	e.Any("/api/lcards", methodNotAllowed("GET, HEAD, POST"))
	e.Any("/api/lcards/:id", methodNotAllowed("GET, HEAD, PUT, DELETE"))

	// A child route prevents Echo's terminal :id parameter from consuming slashes.
	e.RouteNotFound("/api/lcards/:id/*", func(c *echo.Context) error {
		return echo.ErrNotFound
	})
}
