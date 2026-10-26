package list

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/apotourlyan/ludustudii/internal/kit/erresp"
	"github.com/apotourlyan/ludustudii/internal/services/lcards/domain"
	"github.com/labstack/echo/v5"
)

type response struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type lister interface {
	List(ctx context.Context) ([]row, error)
}

type Handler struct {
	store lister
}

func NewHandler(s lister) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Handle(c *echo.Context) error {
	ctx := c.Request().Context()

	data, err := h.store.List(ctx)
	if err != nil {
		slog.ErrorContext(ctx, domain.TextUnexpected, "error", err)
		return c.JSON(http.StatusInternalServerError, erresp.Unexpected())
	}

	return c.JSON(http.StatusOK, toResp(data))
}

func toResp(data []row) []response {
	resp := make([]response, 0, len(data))
	for _, d := range data {
		resp = append(resp, response{
			ID:          d.ID,
			Title:       d.Title,
			Description: d.Description,
		})
	}
	return resp
}
