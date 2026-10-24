package list

import "net/http"

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Step 2: Fetch state
	cards, err := h.store.List(ctx)
	if err != nil {
		writeError(w, err)
		return
	}

	// Step 5: Handle response transport
	writeResponse(w, cards)
}

