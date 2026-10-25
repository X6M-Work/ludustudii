package delete

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/kit/validator"
	"github.com/labstack/echo/v5"
)

const (
	TextInvalidParamID  = "invalid param id"
	TextUnexpectedError = "unexpected error"
	TextNotFound        = "card not found"
)

type executor interface {
	Delete(ctx context.Context, id string) error
}

type Handler struct {
	store executor
}

func NewHandler(s executor) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	id := c.Param("id")
	if err := validator.UUID7("id", id); err != nil {
		slog.ErrorContext(ctx, TextInvalidParamID, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.New(TextInvalidParamID))
	}

	if err := h.store.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			slog.ErrorContext(ctx, TextNotFound, "error", err)
			return c.JSON(http.StatusNotFound, erresp.New(TextNotFound))
		}
		slog.ErrorContext(ctx, TextUnexpectedError, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.New(TextUnexpectedError))
	}

	return c.NoContent(http.StatusNoContent)
}
