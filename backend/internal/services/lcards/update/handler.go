package update

import (
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
)

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

	// Step 3: Execute pure domain rules
	card, err := domain.UpdateCard(req.ID, req.Title, req.Description)
	if err != nil {
		writeError(w, err)
		return
	}

	// Step 4: Persist
	if err := h.store.Update(ctx, card); err != nil {
		writeError(w, err)
		return
	}

	// Step 5: Handle response transport
	writeResponse(w)
}

