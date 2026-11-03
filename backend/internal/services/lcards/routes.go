// Package lcards provides the learning cards API. Each endpoint lives in its
// own subpackage with a Handler and a Store; shared rules are in domain.
package lcards

import (
	"database/sql"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/httpkit"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/create"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/get"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/list"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/remove"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/update"
)

// RegisterRoutes registers the learning card endpoints under domain.CardsPath:
//
//	GET    /api/lcards       list cards, newest first (list)
//	POST   /api/lcards       create a card (create)
//	GET    /api/lcards/{id}  get a card (get)
//	PUT    /api/lcards/{id}  replace a card's title and description (update)
//	DELETE /api/lcards/{id}  delete a card (remove)
//
// Other methods on these paths get 405 with an Allow header, and deeper paths
// get 404.
func RegisterRoutes(rt *httpkit.Router, db *sql.DB) {
	l := list.NewHandler(list.NewStore(db))
	g := get.NewHandler(get.NewStore(db))
	c := create.NewHandler(create.NewStore(db))
	u := update.NewHandler(update.NewStore(db))
	d := remove.NewHandler(remove.NewStore(db))

	rt.Resource(domain.CardsPath, map[string]httpkit.HandlerFunc{
		http.MethodGet:  l.Handle,
		http.MethodPost: c.Handle,
	})
	rt.Resource(domain.CardsPath+"/{id}", map[string]httpkit.HandlerFunc{
		http.MethodGet:    g.Handle,
		http.MethodPut:    u.Handle,
		http.MethodDelete: d.Handle,
	})
}
