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

type Handler struct {
	store           deleter
	validateParamID func(id string) error
}

func NewHandler(s deleter) *Handler {
	return &Handler{store: s, validateParamID: validateParamID}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	id := strings.ToLower(c.Param("id"))
	if err := h.validateParamID(id); err != nil {
		slog.InfoContext(ctx, domain.TextInvalidID, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Param())
	}

	if err := h.store.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.InfoContext(ctx, domain.TextNotFound, "error", err)
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
