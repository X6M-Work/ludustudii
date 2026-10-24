package get

import "net/http"

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Step 1: Bind and validate request transport
	req, err := bindAndValidate(r)
	if err != nil {
		writeError(w, err)
		return
	}

	// Step 2: Fetch state
	card, err := h.store.Get(ctx, req.ID)
	if err != nil {
		writeError(w, err)
		return
	}

	// Step 5: Handle response transport
	writeResponse(w, card)
}

