package lcards

import (
	"database/sql"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/create"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/delete"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/get"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/list"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/update"
)

func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	createHandler := create.NewHandler(create.NewStore(db))
	listHandler := list.NewHandler(list.NewStore(db))
	getHandler := get.NewHandler(get.NewStore(db))
	updateHandler := update.NewHandler(update.NewStore(db))
	deleteHandler := delete.NewHandler(delete.NewStore(db))

	mux.HandleFunc("POST /api/lcards", createHandler.Handle)
	mux.HandleFunc("GET /api/lcards", listHandler.Handle)
	mux.HandleFunc("GET /api/lcards/{id}", getHandler.Handle)
	mux.HandleFunc("PUT /api/lcards/{id}", updateHandler.Handle)
	mux.HandleFunc("DELETE /api/lcards/{id}", deleteHandler.Handle)
}

