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
	createHandler := create.NewHandler(create.NewStore(db))
	listHandler := list.NewHandler(list.NewStore(db))
	getHandler := get.NewHandler(get.NewStore(db))
	updateHandler := update.NewHandler(update.NewStore(db))
	deleteHandler := delete.NewHandler(delete.NewStore(db))

	e.POST("/api/lcards", createHandler.Handle)
	e.GET("/api/lcards", listHandler.Handle)
	e.GET("/api/lcards/:id", getHandler.Handle)
	e.PUT("/api/lcards/:id", updateHandler.Handle)
	e.DELETE("/api/lcards/:id", deleteHandler.Handle)
}
