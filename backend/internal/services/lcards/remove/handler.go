// Package remove implements deleting a learning card.
package remove

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type deleter interface {
	Delete(ctx context.Context, id string) error
}

// Handler serves the DELETE /api/lcards/:id endpoint.
type Handler struct {
	store           deleter
	validateParamID func(id string) error
}

// NewHandler returns a Handler that uses s for storage.
func NewHandler(s deleter) *Handler {
	return &Handler{store: s, validateParamID: validateParamID}
}

// Handle deletes the card whose ID is the :id path parameter and responds 204,
// 400 if the ID is not a UUIDv7 (case-insensitive) or 404 if no card has it.
func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	id := strings.ToLower(c.Param("id"))
	if err := h.validateParamID(id); err != nil {
		slog.DebugContext(ctx, domain.TextInvalidID, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Param())
	}

	if err := h.store.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.DebugContext(ctx, domain.TextNotFound, "error", err)
			return c.JSON(http.StatusNotFound, erresp.NotFound())
		}
		slog.ErrorContext(ctx, domain.TextUnexpected, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.Unexpected())
	}

	return c.NoContent(http.StatusNoContent)
}

func validateParamID(id string) error {
	return validator.UUID7("id", id)
}
