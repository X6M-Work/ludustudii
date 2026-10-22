package lcards

import "net/http"

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating a new todo item
	w.WriteHeader(http.StatusCreated)
}
