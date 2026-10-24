package delete

import "github.com/labstack/echo/v5"

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	// Step 1: Bind and validate request transport
	req, err := bindAndValidate(c)
	if err != nil {
		return writeError(c, err)
	}

	// Step 4: Persist
	if err := h.store.Delete(ctx, req.ID); err != nil {
		return writeError(c, err)
	}

	// Step 5: Handle response transport
	return writeResponse(c)
}
