package lcards

import (
	"database/sql"
	"net/http"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/lcards", h.create)
	mux.HandleFunc("GET /api/lcards", h.list)
	mux.HandleFunc("GET /api/lcards/{id}", h.get)
	mux.HandleFunc("PUT /api/lcards/{id}", h.update)
	mux.HandleFunc("DELETE /api/lcards/{id}", h.delete)
}
