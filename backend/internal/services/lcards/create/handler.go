package create

import (
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

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

	// Step 3: Execute pure domain rules
	card, err := domain.CreateCard(req.Title, req.Description)
	if err != nil {
		return writeError(c, err)
	}

	// Step 4: Persist
	card, err = h.store.Create(ctx, card)
	if err != nil {
		return writeError(c, err)
	}

	// Step 5: Handle response transport
	return writeResponse(c, card)
}
