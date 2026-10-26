package get

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

type response struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type getter interface {
	Get(ctx context.Context, id string) (row, error)
}

type Handler struct {
	store           getter
	validateParamID func(id string) error
}

func NewHandler(s getter) *Handler {
	return &Handler{store: s, validateParamID: validateParamID}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	id := strings.ToLower(c.Param("id"))
	if err := h.validateParamID(id); err != nil {
		slog.InfoContext(ctx, domain.TextInvalidID, "error", err)
		return c.JSON(http.StatusBadRequest, erresp.Param())
	}

	data, err := h.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			slog.InfoContext(ctx, domain.TextNotFound, "error", err)
			return c.JSON(http.StatusNotFound, erresp.NotFound())
		}
		slog.ErrorContext(ctx, domain.TextUnexpected, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.Unexpected())
	}

	return c.JSON(http.StatusOK, toResp(data))
}

func validateParamID(id string) error {
	return validator.UUID7("id", id)
}

func toResp(data row) response {
	return response{
		ID:          data.ID,
		Title:       data.Title,
		Description: data.Description,
	}
}
