// Package lcards provides the learning cards API. Each endpoint lives in its
// own subpackage with a Handler and a Store; shared rules are in domain.
package lcards

import (
	"database/sql"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/create"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/get"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/list"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/remove"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/update"
	"github.com/labstack/echo/v5"
)

// RegisterRoutes registers the learning card endpoints under domain.CardsPath:
//
//	GET    /api/lcards       list cards, newest first (list)
//	POST   /api/lcards       create a card (create)
//	GET    /api/lcards/:id   get a card (get)
//	PUT    /api/lcards/:id   replace a card's title and description (update)
//	DELETE /api/lcards/:id   delete a card (remove)
//
// Other methods on these paths get 405 with an Allow header, and deeper paths
// get 404.
func RegisterRoutes(e *echo.Echo, db *sql.DB) {
	l := list.NewHandler(list.NewStore(db))
	g := get.NewHandler(get.NewStore(db))
	c := create.NewHandler(create.NewStore(db))
	u := update.NewHandler(update.NewStore(db))
	d := remove.NewHandler(remove.NewStore(db))

	e.GET(domain.CardsPath, l.Handle)
	e.GET(domain.CardsPath+"/:id", g.Handle)
	e.POST(domain.CardsPath, c.Handle)
	e.PUT(domain.CardsPath+"/:id", u.Handle)
	e.DELETE(domain.CardsPath+"/:id", d.Handle)

	methodNotAllowed := func(allow string) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set(echo.HeaderAllow, allow)
			return echo.ErrMethodNotAllowed
		}
	}
	e.Any(domain.CardsPath, methodNotAllowed("GET, HEAD, POST"))
	e.Any(domain.CardsPath+"/:id", methodNotAllowed("GET, HEAD, PUT, DELETE"))

	// A child route prevents Echo's terminal :id parameter from consuming slashes.
	e.RouteNotFound(domain.CardsPath+"/:id/*", func(c *echo.Context) error {
		return echo.ErrNotFound
	})
}
