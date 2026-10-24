package list

import "github.com/labstack/echo/v5"

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	// Step 2: Fetch state
	cards, err := h.store.List(ctx)
	if err != nil {
		return writeError(c, err)
	}

	// Step 5: Handle response transport
	return writeResponse(c, cards)
}
